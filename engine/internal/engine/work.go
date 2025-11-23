package engine

import (
	"context"
	"maps"
	"time"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

var workTransitions = StateTransitions[api.WorkStatus]{
	api.WorkPending: util.SetOf(
		api.WorkActive,
		api.WorkSucceeded,
		api.WorkFailed,
		api.WorkNotCompleted,
	),
	api.WorkActive: util.SetOf(
		api.WorkSucceeded,
		api.WorkFailed,
		api.WorkNotCompleted,
	),
	api.WorkSucceeded: {},
	api.WorkFailed:    {},
	api.WorkNotCompleted: util.SetOf(
		api.WorkActive,
	),
}

func (e *Engine) checkCompletableSteps(
	ctx context.Context, flowID api.FlowID, flow *api.FlowState,
) {
	for stepID, exec := range flow.Executions {
		if exec.Status != api.StepActive {
			continue
		}

		allDone := true
		hasFailed := false
		var failureError string

		for _, item := range exec.WorkItems {
			switch item.Status {
			case api.WorkSucceeded:
				// Work succeeded, continue
			case api.WorkFailed:
				// Work failed permanently
				hasFailed = true
				if failureError == "" {
					failureError = item.Error
				}
			case api.WorkNotCompleted, api.WorkPending, api.WorkActive:
				// Not done yet (not completed, pending retry, or still active)
				allDone = false
			}
		}

		if !allDone {
			continue
		}

		fs := FlowStep{FlowID: flowID, StepID: stepID}
		if hasFailed {
			if failureError == "" {
				failureError = "work item failed"
			}
			_ = e.FailStepExecution(ctx, fs, failureError)
		} else {
			step := flow.Plan.GetStep(stepID)
			outputs := aggregateWorkItemOutputs(exec.WorkItems, step)
			dur := time.Since(exec.StartedAt).Milliseconds()
			e.EnqueueStepResult(fs, outputs, dur)
		}
	}
}

func aggregateWorkItemOutputs(
	items map[api.Token]*api.WorkState, step *api.Step,
) api.Args {
	completed := make([]*api.WorkState, 0, len(items))
	for _, item := range items {
		if item.Status == api.WorkSucceeded {
			completed = append(completed, item)
		}
	}

	switch len(completed) {
	case 0:
		return nil
	case 1:
		return completed[0].Outputs
	default:
		aggregated := map[api.Name][]map[string]any{}
		var multiArgNames []api.Name
		if step != nil {
			multiArgNames = step.MultiArgNames()
		}

		for _, item := range completed {
			for outputName, outputValue := range item.Outputs {
				entry := map[string]any{}
				for _, argName := range multiArgNames {
					if val, ok := item.Inputs[argName]; ok {
						entry[string(argName)] = val
					}
				}
				entry["value"] = outputValue

				aggregated[outputName] = append(aggregated[outputName], entry)
			}
		}

		outputs := api.Args{}
		for name, values := range aggregated {
			outputs[name] = values
		}
		return outputs
	}
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
