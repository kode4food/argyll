package policy

import (
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// StateTransitions maps states to their valid next states. The engine uses
// these tables to validate event-driven status changes and to define terminal
// states consistently
type StateTransitions[T comparable] map[T]util.Set[T]

var (
	// FlowTransitions defines the valid flow lifecycle. Active flows may
	// become completed or failed; completed and failed flows are terminal
	FlowTransitions = StateTransitions[api.FlowStatus]{
		api.FlowActive: util.SetOf(
			api.FlowCompleted,
			api.FlowFailed,
		),
		api.FlowCompleted: {},
		api.FlowFailed:    {},
	}

	// StepTransitions defines the valid step lifecycle. Pending steps may
	// start, skip, or fail; active steps may complete or fail
	StepTransitions = StateTransitions[api.StepStatus]{
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

	// WorkTransitions defines the valid work item lifecycle. Not-completed
	// work may be restarted or resolved. Succeeded work may begin compensation
	// if the step ultimately fails
	WorkTransitions = StateTransitions[api.WorkStatus]{
		api.WorkPending: util.SetOf(
			api.WorkActive,
			api.WorkSucceeded,
			api.WorkFailed,
		),
		api.WorkActive: util.SetOf(
			api.WorkSucceeded,
			api.WorkFailed,
			api.WorkNotCompleted,
		),
		api.WorkSucceeded: util.SetOf(
			api.WorkCompensating,
		),
		api.WorkFailed: {},
		api.WorkNotCompleted: util.SetOf(
			api.WorkActive,
			api.WorkSucceeded,
			api.WorkFailed,
		),
		api.WorkCompensating: util.SetOf(
			api.WorkCompensating,
			api.WorkCompensated,
			api.WorkCompFailed,
		),
		api.WorkCompensated: {},
		api.WorkCompFailed:  {},
	}
)

// CanTransition reports whether a status change is valid according to the
// transition table
func (t StateTransitions[T]) CanTransition(from, to T) bool {
	allowed, ok := t[from]
	if !ok {
		return false
	}
	return allowed.Contains(to)
}

// IsTerminal reports whether a status has no valid outgoing transitions
func (t StateTransitions[T]) IsTerminal(state T) bool {
	allowed, ok := t[state]
	return ok && allowed.IsEmpty()
}
