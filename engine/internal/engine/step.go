package engine

import (
	"context"
	"fmt"
	"maps"

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
		ctx, fs.FlowID, fs.StepID, api.StepActive, "start",
		api.EventTypeStepStarted,
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
		ctx, fs.FlowID, fs.StepID, api.StepCompleted, "complete",
		api.EventTypeStepCompleted,
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
		ctx, fs.FlowID, fs.StepID, api.StepFailed, "fail",
		api.EventTypeStepFailed,
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
		ctx, fs.FlowID, fs.StepID, api.StepSkipped, "skip",
		api.EventTypeStepSkipped,
		api.StepSkippedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Reason: reason,
		},
	)
}

func (e *Engine) transitionStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, toStatus api.StepStatus,
	action string, eventType timebox.EventType, eventData any,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		exec, ok := st.Executions[stepID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
		}

		if !stepTransitions.CanTransition(exec.Status, toStatus) {
			return fmt.Errorf("%s: step %s cannot %s from status %s",
				ErrInvalidTransition, stepID, action, exec.Status)
		}

		return util.Raise(ag, eventType, eventData)
	}

	_, err := e.flowExec.Exec(ctx, flowKey(flowID), cmd)
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

// computeWorkItems determines all work items that a step needs to execute
func computeWorkItems(step *api.Step, inputs api.Args) []api.Args {
	argNames := step.MultiArgNames()
	multiArgs := getMultiArgs(argNames, inputs)
	if len(multiArgs) == 0 {
		return []api.Args{inputs}
	}
	return cartesianProduct(multiArgs, inputs)
}

func cartesianProduct(multiArgs MultiArgs, baseInputs api.Args) []api.Args {
	if len(multiArgs) == 0 {
		return nil
	}

	names, arrays := extractMultiArgs(multiArgs)

	var result []api.Args
	var generate func(int, api.Args)
	generate = func(depth int, current api.Args) {
		if depth == len(arrays) {
			result = append(result,
				combineInputs(baseInputs, current, multiArgs),
			)
			return
		}

		name := names[depth]
		for _, val := range arrays[depth] {
			next := current.Set(name, val)
			generate(depth+1, next)
		}
	}

	generate(0, nil)
	return result
}

func extractMultiArgs(multiArgs MultiArgs) ([]api.Name, [][]any) {
	var names []api.Name
	var arrays [][]any
	for name, arr := range multiArgs {
		names = append(names, name)
		arrays = append(arrays, arr)
	}
	return names, arrays
}

func combineInputs(baseInputs, current api.Args, multiArgs MultiArgs) api.Args {
	inputs := api.Args{}
	for k, v := range baseInputs {
		if _, isMulti := multiArgs[k]; !isMulti {
			inputs[k] = v
		}
	}
	maps.Copy(inputs, current)
	return inputs
}
