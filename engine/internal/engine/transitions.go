package engine

import (
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// StateTransitions maps states to their set of valid next states
//
// Generic state transition tables are used to validate flow, step, and work
// status changes
type StateTransitions[T comparable] map[T]util.Set[T]

var (
	flowTransitions = StateTransitions[api.FlowStatus]{
		api.FlowActive: util.SetOf(
			api.FlowCompleted,
			api.FlowFailed,
		),
		api.FlowCompleted: {},
		api.FlowFailed:    {},
	}

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

	workTransitions = StateTransitions[api.WorkStatus]{
		api.WorkPending: util.SetOf(
			api.WorkActive,
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
			api.WorkSucceeded,
			api.WorkFailed,
		),
	}
)

// CanTransition returns whether transition from one state to another is valid
func (t StateTransitions[T]) CanTransition(from, to T) bool {
	allowed, ok := t[from]
	if !ok {
		return false
	}
	return allowed.Contains(to)
}

// IsTerminal returns true if the state has no valid transitions
func (t StateTransitions[T]) IsTerminal(state T) bool {
	allowed, ok := t[state]
	return ok && allowed.IsEmpty()
}
