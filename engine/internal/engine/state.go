package engine

import "github.com/kode4food/spuds/engine/pkg/util"

// StateTransitions maps states to their set of valid next states
type StateTransitions[T comparable] map[T]util.Set[T]

// CanTransition returns whether transition from one state to another is valid
func (st StateTransitions[T]) CanTransition(from, to T) bool {
	allowed, ok := st[from]
	if !ok {
		return false
	}
	return allowed.Contains(to)
}

// IsTerminal returns true if the state has no valid transitions
func (st StateTransitions[T]) IsTerminal(state T) bool {
	allowed, ok := st[state]
	return ok && allowed.IsEmpty()
}
