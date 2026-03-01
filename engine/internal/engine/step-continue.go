package engine

import (
	"errors"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

func (e *Engine) scheduleTimeouts(flow *api.FlowState, now time.Time) {
	if !flowHasTimeouts(flow) {
		return
	}
	e.CancelPrefixedTasks(timeoutFlowPrefix(flow.ID))
	if flowTransitions.IsTerminal(flow.Status) {
		return
	}

	for stepID := range flow.Executions {
		e.scheduleStepTimeouts(flow, stepID, now, false)
	}
}

func (e *Engine) scheduleConsumerTimeouts(
	flow *api.FlowState, producerID api.StepID, now time.Time,
) {
	if flowTransitions.IsTerminal(flow.Status) {
		if flowHasTimeouts(flow) {
			e.CancelPrefixedTasks(timeoutFlowPrefix(flow.ID))
		}
		return
	}

	producer, ok := flow.Plan.Steps[producerID]
	if !ok || producer == nil {
		return
	}

	seen := util.Set[api.StepID]{}
	for name, attr := range producer.Attributes {
		if !attr.IsOutput() {
			continue
		}
		deps, ok := flow.Plan.Attributes[name]
		if !ok {
			continue
		}
		for _, stepID := range deps.Consumers {
			if seen.Contains(stepID) {
				continue
			}
			seen.Add(stepID)
			e.scheduleStepTimeouts(flow, stepID, now, true)
		}
	}
}

func (e *Engine) scheduleStepTimeouts(
	flow *api.FlowState, stepID api.StepID, now time.Time, clearExisting bool,
) {
	step, ok := flow.Plan.Steps[stepID]
	if !ok || !stepHasTimeouts(step) {
		return
	}

	fs := api.FlowStep{FlowID: flow.ID, StepID: stepID}
	if clearExisting {
		e.CancelPrefixedTasks(timeoutStepPrefix(fs))
	}

	if flowTransitions.IsTerminal(flow.Status) {
		return
	}
	exec, ok := flow.Executions[stepID]
	if !ok || exec.Status != api.StepPending {
		return
	}

	s := e.newStepEval(stepID, flow, now)
	anchor := s.requiredReadyAt()
	if anchor.IsZero() {
		return
	}

	for name, attr := range s.step.Attributes {
		if !attr.IsOptional() || attr.Timeout <= 0 {
			continue
		}
		ready, at := s.optionalReadyAt(name, attr, anchor)
		if ready || at.IsZero() {
			continue
		}
		e.scheduleTimeoutTask(fs, name, at)
	}
}

func flowHasTimeouts(flow *api.FlowState) bool {
	for _, step := range flow.Plan.Steps {
		if stepHasTimeouts(step) {
			return true
		}
	}
	return false
}

func stepHasTimeouts(step *api.Step) bool {
	if step == nil {
		return false
	}
	for _, attr := range step.Attributes {
		if attr.IsOptional() && attr.Timeout > 0 {
			return true
		}
	}
	return false
}

func (e *Engine) scheduleTimeoutTask(
	fs api.FlowStep, name api.Name, at time.Time,
) {
	e.ScheduleTask(timeoutKey(fs, name), at, func() error {
		return e.runTimeoutTaskAt(fs, e.Now())
	})
}

func (e *Engine) runTimeoutTaskAt(fs api.FlowStep, now time.Time) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		flow := tx.Value()
		if flowTransitions.IsTerminal(flow.Status) {
			return nil
		}

		exec, ok := flow.Executions[fs.StepID]
		if !ok || exec.Status != api.StepPending {
			return nil
		}

		ready, _ := tx.canStartStepAt(fs.StepID, flow, now)
		if !ready {
			return nil
		}

		err := tx.prepareStep(fs.StepID)
		if err != nil && errors.Is(err, ErrStepAlreadyPending) {
			return nil
		}
		return err
	})
}

func timeoutKey(fs api.FlowStep, name api.Name) []string {
	return []string{"timeout", string(fs.FlowID), string(fs.StepID),
		string(name)}
}

func timeoutFlowPrefix(flowID api.FlowID) []string {
	return []string{"timeout", string(flowID)}
}

func timeoutStepPrefix(fs api.FlowStep) []string {
	return []string{"timeout", string(fs.FlowID), string(fs.StepID)}
}
