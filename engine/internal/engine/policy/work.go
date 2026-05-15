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
// pending, active, or not-completed work keeps the step open; any failed work
// makes the step fail once all work is otherwise done
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
	if res.Failed && res.FailureError == "" {
		res.FailureError = "work item failed"
	}
	return res
}
