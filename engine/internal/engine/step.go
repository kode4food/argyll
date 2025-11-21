package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

// FlowStep identifies a step execution within a workflow
type FlowStep struct {
	FlowID timebox.ID
	StepID timebox.ID
}

var (
	stepTransitions = util.StateTransitions[api.StepStatus]{
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
	)
)

// Step state transition methods

// StartStepExecution transitions a step to the active state and begins its
// execution with the provided input arguments
func (e *Engine) StartStepExecution(
	ctx context.Context, fs FlowStep, step *api.Step, inputs api.Args,
) error {
	workItems := computeWorkItems(step, inputs)

	workItemsMap := make(map[api.Token]api.Args)
	for _, workInputs := range workItems {
		token := api.Token(uuid.New().String())
		workItemsMap[token] = workInputs
	}

	return e.transitionStepExecution(
		ctx, fs, api.StepActive, "start", api.EventTypeStepStarted,
		api.StepStartedEvent{
			FlowID:    fs.FlowID,
			StepID:    fs.StepID,
			Inputs:    inputs,
			WorkItems: workItemsMap,
		},
	)
}

// CompleteStepExecution transitions a step to the completed state with the
// provided output values and execution duration
func (e *Engine) CompleteStepExecution(
	ctx context.Context, fs FlowStep, outputs api.Args, dur int64,
) error {
	return e.transitionStepExecution(
		ctx, fs, api.StepCompleted, "complete", api.EventTypeStepCompleted,
		api.StepCompletedEvent{
			FlowID:   fs.FlowID,
			StepID:   fs.StepID,
			Outputs:  outputs,
			Duration: dur,
		},
	)
}

// FailStepExecution transitions a step to the failed state with the specified
// error message
func (e *Engine) FailStepExecution(
	ctx context.Context, fs FlowStep, errMsg string,
) error {
	return e.transitionStepExecution(
		ctx, fs, api.StepFailed, "fail", api.EventTypeStepFailed,
		api.StepFailedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Error:  errMsg,
		},
	)
}

// SkipStepExecution transitions a step to the skipped state with the provided
// reason for skipping
func (e *Engine) SkipStepExecution(
	ctx context.Context, fs FlowStep, reason string,
) error {
	return e.transitionStepExecution(
		ctx, fs, api.StepSkipped, "skip", api.EventTypeStepSkipped,
		api.StepSkippedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Reason: reason,
		},
	)
}

func (e *Engine) transitionStepExecution(
	ctx context.Context, fs FlowStep, toStatus api.StepStatus, action string,
	eventType timebox.EventType, eventData any,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		exec, ok := st.Executions[fs.StepID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrStepNotInPlan, fs.StepID)
		}

		if !stepTransitions.CanTransition(exec.Status, toStatus) {
			return fmt.Errorf("%s: step %s cannot %s from status %s",
				ErrInvalidTransition, fs.StepID, action, exec.Status)
		}

		return util.Raise(ag, eventType, eventData)
	}

	_, err := e.flowExec.Exec(ctx, flowKey(fs.FlowID), cmd)
	return err
}

// Step state checking methods

func (e *Engine) isStepComplete(
	stepID timebox.ID, flow *api.FlowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if !ok {
		return false
	}
	return exec.Status == api.StepCompleted || exec.Status == api.StepSkipped
}

func (e *Engine) canStepComplete(
	stepID timebox.ID, flow *api.FlowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if !ok {
		return false
	}

	if stepTransitions.IsTerminal(exec.Status) {
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

func isAsyncStep(stepType api.StepType) bool {
	return asyncStepTypes.Contains(stepType)
}
