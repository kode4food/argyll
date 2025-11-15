package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

var (
	stepTransitions = map[api.StepStatus]map[api.StepStatus]bool{
		api.StepPending: {
			api.StepActive:  true,
			api.StepSkipped: true,
			api.StepFailed:  true,
		},
		api.StepActive: {
			api.StepCompleted: true,
			api.StepFailed:    true,
		},
		api.StepCompleted: {},
		api.StepFailed:    {},
		api.StepSkipped:   {},
	}

	asyncStepTypes = map[api.StepType]bool{
		api.StepTypeAsync: true,
	}
)

// Step state transition methods

func (e *Engine) StartStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, inputs api.Args,
) error {
	return e.transitionStepExecution(
		ctx, flowID, stepID, api.StepActive, "start",
		api.EventTypeStepStarted,
		api.StepStartedEvent{
			FlowID: flowID,
			StepID: stepID,
			Inputs: inputs,
		},
	)
}

func (e *Engine) CompleteStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, outputs api.Args, dur int64,
) error {
	return e.transitionStepExecution(
		ctx, flowID, stepID, api.StepCompleted, "complete",
		api.EventTypeStepCompleted,
		api.StepCompletedEvent{
			FlowID:   flowID,
			StepID:   stepID,
			Outputs:  outputs,
			Duration: dur,
		},
	)
}

func (e *Engine) FailStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, errMsg string,
) error {
	return e.transitionStepExecution(
		ctx, flowID, stepID, api.StepFailed, "fail",
		api.EventTypeStepFailed,
		api.StepFailedEvent{
			FlowID: flowID,
			StepID: stepID,
			Error:  errMsg,
		},
	)
}

func (e *Engine) SkipStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, reason string,
) error {
	return e.transitionStepExecution(
		ctx, flowID, stepID, api.StepSkipped, "skip",
		api.EventTypeStepSkipped,
		api.StepSkippedEvent{
			FlowID: flowID,
			StepID: stepID,
			Reason: reason,
		},
	)
}

func (e *Engine) transitionStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, toStatus api.StepStatus,
	action string, eventType timebox.EventType, eventData any,
) error {
	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		exec, ok := st.Executions[stepID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
		}

		if !canStepTransitionTo(exec.Status, toStatus) {
			return fmt.Errorf("%s: step %s cannot %s from status %s",
				ErrInvalidTransition, stepID, action, exec.Status)
		}

		ev, err := json.Marshal(eventData)
		if err != nil {
			return err
		}
		ag.Raise(eventType, ev)
		return nil
	}

	_, err := e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
}

// Step state checking methods

func (e *Engine) isStepComplete(
	stepID timebox.ID, flow *api.WorkflowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if !ok {
		return false
	}
	return exec.Status == api.StepCompleted || exec.Status == api.StepSkipped
}

func (e *Engine) canStepComplete(
	stepID timebox.ID, flow *api.WorkflowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if !ok {
		return false
	}

	if isStepTerminal(exec.Status) {
		return exec.Status == api.StepCompleted
	}

	step := flow.Plan.GetStep(stepID)
	if step == nil {
		return false
	}

	for requiredInputName, attr := range step.Attributes {
		if attr.IsRequired() {
			if _, hasAttr := flow.Attributes[requiredInputName]; hasAttr {
				continue
			}
			if !e.HasInputProvider(requiredInputName, flow) {
				return false
			}
		}
	}

	return true
}

func (e *Engine) StepProvidesInput(
	step *api.Step, name api.Name, flow *api.WorkflowState,
) bool {
	for attrName, attr := range step.Attributes {
		if attrName == name && attr.IsOutput() {
			return e.canStepComplete(step.ID, flow)
		}
	}
	return false
}

func (e *Engine) appendFailedStep(
	failed []string, stepID timebox.ID, exec *api.ExecutionState,
) []string {
	if exec.Status != api.StepFailed {
		return failed
	}

	if exec.Error == "" {
		return append(failed, string(stepID))
	}
	return append(failed, fmt.Sprintf("%s (%s)", stepID, exec.Error))
}

func canStepTransitionTo(from, to api.StepStatus) bool {
	allowed, ok := stepTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}

func isStepTerminal(status api.StepStatus) bool {
	allowed, ok := stepTransitions[status]
	return ok && len(allowed) == 0
}

func isAsyncStep(stepType api.StepType) bool {
	return asyncStepTypes[stepType]
}
