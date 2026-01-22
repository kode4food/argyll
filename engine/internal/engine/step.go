package engine

import (
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// FlowStep identifies a step execution within a flow
type FlowStep struct {
	FlowID api.FlowID
	StepID api.StepID
}

var (
	stepTransitions = StateTransitions[api.StepStatus]{
		api.StepPending: util.SetOf(
			api.StepActive,
			api.StepSkipped,
			api.StepFailed,
		),
		api.StepActive: util.SetOf(
			api.StepCompleted,
			api.StepFailed,
		),
		api.StepCompleted: {},
		api.StepFailed:    {},
		api.StepSkipped:   {},
	}

	asyncStepTypes = util.SetOf(
		api.StepTypeAsync,
		api.StepTypeFlow,
	)
)

// Step state transition methods

// FailStepExecution transitions a step to the failed state with the specified
// error message
func (e *Engine) FailStepExecution(fs FlowStep, errMsg string) error {
	return e.transitionStepExecution(
		fs, api.StepFailed, "fail", api.EventTypeStepFailed,
		api.StepFailedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Error:  errMsg,
		},
	)
}

func (e *Engine) transitionStepExecution(
	fs FlowStep, toStatus api.StepStatus, action string,
	eventType api.EventType, eventData any,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		exec, ok := st.Executions[fs.StepID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrStepNotInPlan, fs.StepID)
		}

		if !stepTransitions.CanTransition(exec.Status, toStatus) {
			return fmt.Errorf("%w: step %s cannot %s from status %s",
				ErrInvalidTransition, fs.StepID, action, exec.Status)
		}

		return events.Raise(ag, eventType, eventData)
	}

	_, err := e.execFlow(flowKey(fs.FlowID), cmd)
	return err
}

// Step state checking methods

func (e *Engine) isStepComplete(stepID api.StepID, flow *api.FlowState) bool {
	exec, ok := flow.Executions[stepID]
	if !ok {
		return false
	}
	return exec.Status == api.StepCompleted || exec.Status == api.StepSkipped
}

func (e *Engine) canStepComplete(stepID api.StepID, flow *api.FlowState) bool {
	exec, ok := flow.Executions[stepID]
	if !ok {
		return false
	}

	if stepTransitions.IsTerminal(exec.Status) {
		return exec.Status == api.StepCompleted
	}

	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return false
	}

	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			if _, hasAttr := flow.Attributes[name]; hasAttr {
				continue
			}
			if !e.HasInputProvider(name, flow) {
				return false
			}
		}
	}

	return true
}

func isAsyncStep(stepType api.StepType) bool {
	return asyncStepTypes.Contains(stepType)
}
