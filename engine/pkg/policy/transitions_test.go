package policy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type TestState string

const (
	StateInit     TestState = "init"
	StateRunning  TestState = "running"
	StateComplete TestState = "complete"
	StateFailed   TestState = "failed"
)

func TestCanTransition(t *testing.T) {
	transitions := policy.StateTransitions[TestState]{
		StateInit:     util.SetOf(StateRunning, StateFailed),
		StateRunning:  util.SetOf(StateRunning, StateComplete, StateFailed),
		StateComplete: {},
		StateFailed:   {},
	}

	can := func(from, to TestState) bool {
		return transitions.CanTransition(
			policy.TransitionArgs[TestState]{From: from, To: to},
		)
	}

	// The table lists StateRunning as its own successor, but staying put is
	// not a transition
	assert.False(t, can(StateRunning, StateRunning))

	assert.True(t, can(StateInit, StateRunning))
	assert.True(t, can(StateInit, StateFailed))
	assert.True(t, can(StateRunning, StateComplete))
	assert.True(t, can(StateRunning, StateFailed))

	assert.False(t, can(StateInit, StateComplete))
	assert.False(t, can(StateComplete, StateRunning))
	assert.False(t, can(StateFailed, StateRunning))

	assert.False(t, can("unknown", StateRunning))
}

func TestIsTerminal(t *testing.T) {
	transitions := policy.StateTransitions[TestState]{
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
