package policy

import "github.com/kode4food/argyll/engine/pkg/api"

// WorkCompletion summarizes whether a step's work items have all reached a
// step-level outcome. It is used to decide whether to raise StepCompleted or
// StepFailed after individual work items finish
type WorkCompletion struct {
	FailureError string
	Done         bool
	Failed       bool
}

// StepParallelism returns the effective dispatch parallelism for a step. A
// missing, zero, or negative setting means one work item at a time
func StepParallelism(step *api.Step) int {
	if step.WorkConfig == nil || step.WorkConfig.Parallelism <= 0 {
		return 1
	}
	return step.WorkConfig.Parallelism
}

// CountActiveWorkItems counts currently executing work items for dispatch
// throttling
func CountActiveWorkItems(items api.WorkItems) int {
	active := 0
	for _, work := range items {
		if work.Status == api.WorkActive {
			active++
		}
	}
	return active
}

// StepWorkCompletion classifies a step's work items for step completion. Any
// permanently failed work item makes the step fail immediately: Done is true
// as soon as a failure is present, regardless of pending or active items.
// Pending items that have not yet started are abandoned; in-flight active
// items may still report results but the step is already terminal. Without any
// failure, the step completes only when all items have succeeded
func StepWorkCompletion(items api.WorkItems) WorkCompletion {
	res := WorkCompletion{Done: true}
	for _, work := range items {
		switch work.Status {
		case api.WorkSucceeded:
		case api.WorkFailed:
			res.Failed = true
			if res.FailureError == "" {
				res.FailureError = work.Error
			}
		case api.WorkNotCompleted, api.WorkPending, api.WorkActive:
			res.Done = false
		}
	}
	if res.Failed {
		res.Done = true
		if res.FailureError == "" {
			res.FailureError = "work item failed"
		}
	}
	return res
}
