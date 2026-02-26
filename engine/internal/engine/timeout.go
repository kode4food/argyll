package engine

import (
	"errors"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

func (e *Engine) scheduleTimeouts(flow *api.FlowState, now time.Time) {
	e.CancelScheduledTaskPrefix(timeoutFlowPrefix(flow.ID))
	if flowTransitions.IsTerminal(flow.Status) {
		return
	}

	for stepID := range flow.Executions {
		e.scheduleStepTimeouts(flow, stepID, now)
	}
}

func (e *Engine) scheduleConsumerTimeouts(
	flow *api.FlowState, producerID api.StepID, now time.Time,
) {
	if flowTransitions.IsTerminal(flow.Status) {
		e.CancelScheduledTaskPrefix(timeoutFlowPrefix(flow.ID))
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
			e.scheduleStepTimeouts(flow, stepID, now)
		}
	}
}

func (e *Engine) scheduleStepTimeouts(
	flow *api.FlowState, stepID api.StepID, now time.Time,
) {
	fs := api.FlowStep{FlowID: flow.ID, StepID: stepID}
	e.CancelScheduledTaskPrefix(timeoutStepPrefix(fs))

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

func (e *Engine) scheduleTimeoutTask(
	fs api.FlowStep, name api.Name, at time.Time,
) {
	e.ScheduleTaskKeyed(timeoutKey(fs, name), at, func() error {
		return e.runTimeoutTask(fs)
	})
}

func (e *Engine) runTimeoutTask(fs api.FlowStep) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		flow := tx.Value()
		if flowTransitions.IsTerminal(flow.Status) {
			return nil
		}

		exec, ok := flow.Executions[fs.StepID]
		if !ok || exec.Status != api.StepPending {
			return nil
		}

		ready, _ := tx.canStartStepAt(fs.StepID, flow, time.Now())
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
	return []string{"timeout", string(fs.FlowID), string(fs.StepID), string(name)}
}

func timeoutFlowPrefix(flowID api.FlowID) []string {
	return []string{"timeout", string(flowID)}
}

func timeoutStepPrefix(fs api.FlowStep) []string {
	return []string{"timeout", string(fs.FlowID), string(fs.StepID)}
}
