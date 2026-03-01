package engine

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

// prepareStep validates and prepares a step to execute within a transaction,
// raising the StepStarted event via aggregator and scheduling work execution
// after commit
func (tx *flowTx) prepareStep(stepID api.StepID) error {
	flow := tx.Value()

	exec := flow.Executions[stepID]
	if exec.Status != api.StepPending {
		return fmt.Errorf("%w: %s (status=%s)",
			ErrStepAlreadyPending, stepID, exec.Status)
	}

	step := flow.Plan.Steps[stepID]
	inputs := tx.collectStepInputs(step, flow)

	shouldExecute, err := tx.evaluateStepPredicate(step, inputs)
	if err != nil {
		return tx.handlePredicateFailure(stepID, err)
	}
	if !shouldExecute {
		if err := events.Raise(tx.FlowAggregator, api.EventTypeStepSkipped,
			api.StepSkippedEvent{
				FlowID: tx.flowID,
				StepID: stepID,
				Reason: "predicate returned false",
			},
		); err != nil {
			return err
		}
		return call.Perform(
			tx.checkUnreachable,
			tx.checkTerminal,
			tx.startReadyPendingSteps,
		)
	}

	workItemsList, err := computeWorkItems(step, inputs)
	if err != nil {
		return err
	}
	workItemsMap := map[api.Token]api.Args{}
	for _, workInputs := range workItemsList {
		token := api.Token(uuid.New().String())
		workItemsMap[token] = workInputs
	}

	if err := events.Raise(tx.FlowAggregator, api.EventTypeStepStarted,
		api.StepStartedEvent{
			FlowID:    tx.flowID,
			StepID:    stepID,
			Inputs:    inputs,
			WorkItems: workItemsMap,
		},
	); err != nil {
		return err
	}

	started, err := tx.startPendingWork(step)
	if err != nil {
		return err
	}

	if len(started) > 0 {
		tx.OnSuccess(func(flow *api.FlowState) {
			tx.CancelPrefixedTasks(
				timeoutStepPrefix(api.FlowStep{
					FlowID: tx.flowID,
					StepID: step.ID,
				}),
			)
			tx.handleWorkItemsExecution(step, inputs, flow.Metadata, started)
		})
	}

	return nil
}

func (tx *flowTx) canStartStep(stepID api.StepID, flow *api.FlowState) bool {
	ready, _ := tx.canStartStepAt(stepID, flow, tx.Now())
	return ready
}

func (tx *flowTx) canStartStepAt(
	stepID api.StepID, flow *api.FlowState, now time.Time,
) (bool, time.Time) {
	return tx.newStepEval(stepID, flow, now).canStart()
}

func (tx *flowTx) findInitialSteps(flow *api.FlowState) []api.StepID {
	res := make([]api.StepID, 0, len(flow.Executions))
	for stepID, exec := range flow.Executions {
		if exec.Status != api.StepPending {
			continue
		}
		if tx.canStartStep(stepID, flow) {
			res = append(res, stepID)
		}
	}
	return res
}

func (e *Engine) newStepEval(
	stepID api.StepID, flow *api.FlowState, now time.Time,
) *stepEval {
	return &stepEval{
		e:      e,
		flow:   flow,
		stepID: stepID,
		step:   flow.Plan.Steps[stepID],
		now:    now,
	}
}

func (e *Engine) evaluateStepPredicate(
	step *api.Step, inputs api.Args,
) (bool, error) {
	if step.Predicate == nil {
		return true, nil
	}

	comp, err := e.scripts.Compile(step, step.Predicate)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrPredicateCompileFailed, err)
	}

	env, err := e.scripts.Get(step.Predicate.Language)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrPredicateEnvFailed, err)
	}

	shouldExecute, err := env.EvaluatePredicate(comp, step, inputs)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrPredicateEvalFailed, err)
	}

	return shouldExecute, nil
}

func (tx *flowTx) collectStepInputs(
	step *api.Step, flow *api.FlowState,
) api.Args {
	inputs := api.Args{}
	now := tx.Now()
	ev := tx.newStepEval(step.ID, flow, now)

	for name, attr := range step.Attributes {
		if !attr.IsRuntimeInput() {
			continue
		}

		if attr.IsConst() {
			inputs[name] = gjson.Parse(attr.Default).Value()
			continue
		}

		if attr.IsOptional() {
			ready, fallback := ev.optionalFallback(name, attr)
			if !ready {
				continue
			}
			if fallback {
				if attr.Default != "" {
					value := gjson.Parse(attr.Default).Value()
					val := tx.mapper.MapInput(step, name, attr, value)
					paramName := tx.mapper.InputParamName(name, attr)
					inputs[paramName] = val
				}
				continue
			}
		}

		attrVal, ok := flow.Attributes[name]
		if !ok {
			if !attr.IsRequired() && attr.Default != "" {
				value := gjson.Parse(attr.Default).Value()
				val := tx.mapper.MapInput(step, name, attr, value)
				paramName := tx.mapper.InputParamName(name, attr)
				inputs[paramName] = val
				continue
			}
			continue
		}

		val := tx.mapper.MapInput(step, name, attr, attrVal.Value)
		paramName := tx.mapper.InputParamName(name, attr)
		inputs[paramName] = val
	}

	return inputs
}

type stepEval struct {
	e      *Engine
	flow   *api.FlowState
	stepID api.StepID
	step   *api.Step
	now    time.Time
}

func (s *stepEval) canStart() (bool, time.Time) {
	exec := s.flow.Executions[s.stepID]
	if exec.Status != api.StepPending {
		return false, time.Time{}
	}
	if !s.e.areOutputsNeeded(s.stepID, s.flow) {
		return false, time.Time{}
	}

	anchor := s.requiredReadyAt()
	if anchor.IsZero() {
		return false, time.Time{}
	}

	optReady, nextAt := s.hasOptionalReady(anchor)
	if !optReady {
		return false, nextAt
	}
	return true, time.Time{}
}

func (s *stepEval) hasOptionalReady(anchor time.Time) (bool, time.Time) {
	blocked := false
	var nextAt time.Time

	for name, attr := range s.step.Attributes {
		if !attr.IsOptional() {
			continue
		}
		ok, _, at := s.optionalDecisionAt(name, attr, anchor)
		if !ok {
			blocked = true
		}
		if !at.IsZero() && (nextAt.IsZero() || at.Before(nextAt)) {
			nextAt = at
		}
	}

	return !blocked, nextAt
}

func (s *stepEval) optionalReadyAt(
	name api.Name, attr *api.AttributeSpec, anchor time.Time,
) (bool, time.Time) {
	ready, _, at := s.optionalDecisionAt(name, attr, anchor)
	return ready, at
}

func (s *stepEval) optionalFallback(
	name api.Name, attr *api.AttributeSpec,
) (bool, bool) {
	anchor := s.requiredReadyAt()
	ready, fallback, _ := s.optionalDecisionAt(name, attr, anchor)
	return ready, fallback
}

func (s *stepEval) optionalDecisionAt(
	name api.Name, attr *api.AttributeSpec, anchor time.Time,
) (bool, bool, time.Time) {
	attrVal, hasAttr := s.flow.Attributes[name]
	deps, ok := s.flow.Plan.Attributes[name]

	if hasAttr {
		if attrVal.Step == "" {
			return true, false, time.Time{}
		}

		at := s.optionalAt(anchor, attr.Timeout)
		if at.IsZero() {
			return true, false, time.Time{}
		}

		setAt := attrVal.SetAt
		if !setAt.IsZero() && setAt.After(at) {
			return true, true, time.Time{}
		}
		return true, false, time.Time{}
	}

	if !ok || len(deps.Providers) == 0 {
		return true, true, time.Time{}
	}

	if attr.Timeout <= 0 {
		return true, true, time.Time{}
	}

	at := s.optionalAt(anchor, attr.Timeout)
	if at.IsZero() {
		return false, false, time.Time{}
	}
	if !at.After(s.now) {
		return true, true, time.Time{}
	}
	return false, false, at
}

func (s *stepEval) optionalAt(anchor time.Time, timeoutMS int64) time.Time {
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
