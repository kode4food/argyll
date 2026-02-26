package engine

import (
	"errors"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func (e *Engine) scheduleTimeouts(flow *api.FlowState, now time.Time) {
	e.CancelScheduledTaskPrefix(timeoutFlowPrefix(flow.ID))
	if flowTransitions.IsTerminal(flow.Status) {
		return
	}

	for stepID, exec := range flow.Executions {
		if exec.Status != api.StepPending {
			continue
		}

		s := e.newStepEval(stepID, flow, now)
		anchor := s.requiredReadyAt()
		if anchor.IsZero() {
			continue
		}

		for name, attr := range s.step.Attributes {
			if !attr.IsOptional() || attr.Timeout <= 0 {
				continue
			}
			ready, at := s.optionalReadyAt(name, attr, anchor)
			if ready || at.IsZero() {
				continue
			}
			e.scheduleTimeoutTask(api.FlowStep{
				FlowID: flow.ID,
				StepID: stepID,
			}, name, at)
		}
	}
}

func (e *Engine) scheduleTimeoutTask(
	fs api.FlowStep, name api.Name, at time.Time,
) {
	e.ScheduleTaskKeyed(timeoutKey(fs, name),
		func() error {
			return e.runTimeoutTask(fs)
		},
		at,
	)
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
