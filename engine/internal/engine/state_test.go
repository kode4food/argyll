package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/spuds/engine/pkg/util"
)

type TestState string

const (
	StateInit     TestState = "init"
	StateRunning  TestState = "running"
	StateComplete TestState = "complete"
	StateFailed   TestState = "failed"
)

func TestStateTransitionsCanTransition(t *testing.T) {
	transitions := StateTransitions[TestState]{
		StateInit:     util.SetOf(StateRunning, StateFailed),
		StateRunning:  util.SetOf(StateComplete, StateFailed),
		StateComplete: {},
		StateFailed:   {},
	}

	assert.True(t, transitions.CanTransition(StateInit, StateRunning))
	assert.True(t, transitions.CanTransition(StateInit, StateFailed))
	assert.True(t, transitions.CanTransition(StateRunning, StateComplete))
	assert.True(t, transitions.CanTransition(StateRunning, StateFailed))

	assert.False(t, transitions.CanTransition(StateInit, StateComplete))
	assert.False(t, transitions.CanTransition(StateComplete, StateRunning))
	assert.False(t, transitions.CanTransition(StateFailed, StateRunning))

	assert.False(t, transitions.CanTransition("unknown", StateRunning))
}

func TestStateTransitionsIsTerminal(t *testing.T) {
	transitions := StateTransitions[TestState]{
		StateInit:     util.SetOf(StateRunning, StateFailed),
		StateRunning:  util.SetOf(StateComplete, StateFailed),
		StateComplete: {},
		StateFailed:   {},
	}

	assert.False(t, transitions.IsTerminal(StateInit))
	assert.False(t, transitions.IsTerminal(StateRunning))

	assert.True(t, transitions.IsTerminal(StateComplete))
	assert.True(t, transitions.IsTerminal(StateFailed))
}
