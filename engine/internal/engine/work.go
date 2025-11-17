package engine

import (
	"context"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

var workTransitions = util.StateTransitions[api.WorkStatus]{
	api.WorkPending: util.SetOf(
		api.WorkActive,
		api.WorkCompleted,
		api.WorkFailed,
	),
	api.WorkActive: util.SetOf(
		api.WorkCompleted,
		api.WorkFailed,
	),
	api.WorkCompleted: {},
	api.WorkFailed:    {},
}

func (e *Engine) checkCompletableSteps(
	ctx context.Context, flowID timebox.ID, flow *api.WorkflowState,
) {
	for stepID, exec := range flow.Executions {
		if exec.Status != api.StepActive || exec.WorkItems == nil {
			continue
		}

		allDone := true
		hasFailed := false
		var failureError string

		for _, item := range exec.WorkItems {
			switch item.Status {
			case api.WorkFailed:
				hasFailed = true
				if failureError == "" {
					failureError = item.Error
				}
			case api.WorkPending:
				if !item.NextRetryAt.IsZero() {
					allDone = false
				}
			case api.WorkActive:
				allDone = false
			}
		}

		if !allDone {
			continue
		}

		if hasFailed {
			if failureError == "" {
				failureError = "work item failed"
			}
			_ = e.FailStepExecution(ctx, flowID, stepID, failureError)
		} else {
			outputs := aggregateWorkItemOutputs(
				exec.WorkItems, flow.Plan.GetStep(stepID),
			)
			dur := time.Since(exec.StartedAt).Milliseconds()
			e.EnqueueStepResult(flowID, stepID, outputs, dur)
		}
	}
}

func aggregateWorkItemOutputs(
	items map[api.Token]*api.WorkState, step *api.Step,
) api.Args {
	completed := make([]*api.WorkState, 0, len(items))
	for _, item := range items {
		if item.Status == api.WorkCompleted {
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
