package engine

import (
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	stepEval struct {
		tx     *flowTx
		flow   *api.FlowState
		stepID api.StepID
		step   *api.Step
		now    time.Time
	}

	optDecision struct {
		ready    bool
		fallback bool
		deadline time.Time
	}
)

func (tx *flowTx) canStartStep(stepID api.StepID, flow *api.FlowState) bool {
	ready, _ := tx.canStartStepAt(stepID, flow, time.Now())
	return ready
}

func (tx *flowTx) canStartStepAt(
	stepID api.StepID, flow *api.FlowState, now time.Time,
) (bool, time.Time) {
	return tx.newStepEval(stepID, flow, now).canStart()
}

// findInitialSteps finds steps that can start when a flow begins
func (tx *flowTx) findInitialSteps(flow *api.FlowState) []api.StepID {
	var ready []api.StepID

	for stepID := range flow.Plan.Steps {
		if tx.canStartStep(stepID, flow) {
			ready = append(ready, stepID)
		}
	}

	return ready
}

func (tx *flowTx) newStepEval(
	stepID api.StepID, flow *api.FlowState, now time.Time,
) *stepEval {
	return &stepEval{
		tx:     tx,
		flow:   flow,
		stepID: stepID,
		step:   flow.Plan.Steps[stepID],
		now:    now,
	}
}

func (s *stepEval) canStart() (bool, time.Time) {
	exec := s.flow.Executions[s.stepID]
	if exec.Status != api.StepPending {
		return false, time.Time{}
	}
	if !s.tx.areOutputsNeeded(s.stepID, s.flow) {
		return false, time.Time{}
	}
	anchor := s.requiredReadyAt()
	if anchor.IsZero() {
		return false, time.Time{}
	}
	optReady, nextDeadline := s.hasOptionalReady(anchor)
	if !optReady {
		return false, nextDeadline
	}
	return true, time.Time{}
}

func (s *stepEval) hasOptionalReady(anchor time.Time) (bool, time.Time) {
	blocked := false
	var nextDeadline time.Time

	for name, attr := range s.step.Attributes {
		if !attr.IsOptional() {
			continue
		}
		ready, deadline := s.optionalReadyDeadline(name, attr, anchor)
		if !ready {
			blocked = true
		}
		if !deadline.IsZero() && (nextDeadline.IsZero() ||
			deadline.Before(nextDeadline)) {
			nextDeadline = deadline
		}
	}

	return !blocked, nextDeadline
}

func (s *stepEval) optionalReadyDeadline(
	name api.Name, attr *api.AttributeSpec, anchor time.Time,
) (bool, time.Time) {
	d := s.optionalDecisionAt(name, attr, anchor)
	return d.ready, d.deadline
}

func (s *stepEval) optionalFallback(
	name api.Name, attr *api.AttributeSpec,
) (bool, bool) {
	anchor := s.requiredReadyAt()
	d := s.optionalDecisionAt(name, attr, anchor)
	return d.ready, d.fallback
}

func (s *stepEval) optionalDecisionAt(
	name api.Name, attr *api.AttributeSpec,
	anchor time.Time,
) optDecision {
	attrVal, hasAttr := s.flow.Attributes[name]
	deps, ok := s.flow.Plan.Attributes[name]

	if hasAttr {
		if attrVal.Step == "" {
			return optDecision{ready: true}
		}

		deadline := s.optionalDeadline(anchor, attr.Timeout)
		ok := !deadline.IsZero()
		if !ok {
			return optDecision{ready: true}
		}

		setAt := attrVal.SetAt
		if !setAt.IsZero() && setAt.After(deadline) {
			return optDecision{ready: true, fallback: true}
		}
		return optDecision{ready: true}
	}

	if !ok || len(deps.Providers) == 0 {
		return optDecision{ready: true, fallback: true}
	}

	if attr.Timeout <= 0 {
		return optDecision{ready: true, fallback: true}
	}

	deadline := s.optionalDeadline(anchor, attr.Timeout)
	ok = !deadline.IsZero()
	if !ok {
		return optDecision{}
	}
	if !deadline.After(s.now) {
		return optDecision{ready: true, fallback: true}
	}
	return optDecision{deadline: deadline}
}

func (s *stepEval) optionalDeadline(
	anchor time.Time, timeoutMS int64,
) time.Time {
	if anchor.IsZero() {
		return time.Time{}
	}
	return anchor.Add(time.Duration(timeoutMS) * time.Millisecond)
}

func (s *stepEval) requiredReadyAt() time.Time {
	anchor := s.flow.CreatedAt

	for name, attr := range s.step.Attributes {
		if !attr.IsRequired() {
			continue
		}

		attrVal, ok := s.flow.Attributes[name]
		if !ok {
			return time.Time{}
		}

		setAt := attrVal.SetAt
		if setAt.IsZero() {
			return time.Time{}
		}
		if anchor.IsZero() || setAt.After(anchor) {
			anchor = setAt
		}
	}

	return anchor
}
