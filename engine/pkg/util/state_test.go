package util

import (
	"testing"
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
		StateInit:     SetOf(StateRunning, StateFailed),
		StateRunning:  SetOf(StateComplete, StateFailed),
		StateComplete: {},
		StateFailed:   {},
	}

	// Valid transitions
	if !transitions.CanTransition(StateInit, StateRunning) {
		t.Error("should allow init -> running")
	}
	if !transitions.CanTransition(StateInit, StateFailed) {
		t.Error("should allow init -> failed")
	}
	if !transitions.CanTransition(StateRunning, StateComplete) {
		t.Error("should allow running -> complete")
	}
	if !transitions.CanTransition(StateRunning, StateFailed) {
		t.Error("should allow running -> failed")
	}

	// Invalid transitions
	if transitions.CanTransition(StateInit, StateComplete) {
		t.Error("should not allow init -> complete")
	}
	if transitions.CanTransition(StateComplete, StateRunning) {
		t.Error("should not allow complete -> running")
	}
	if transitions.CanTransition(StateFailed, StateRunning) {
		t.Error("should not allow failed -> running")
	}

	// Unknown state
	if transitions.CanTransition("unknown", StateRunning) {
		t.Error("should not allow transition from unknown state")
	}
}

func TestStateTransitionsIsTerminal(t *testing.T) {
	transitions := StateTransitions[TestState]{
		StateInit:     SetOf(StateRunning, StateFailed),
		StateRunning:  SetOf(StateComplete, StateFailed),
		StateComplete: {},
		StateFailed:   {},
	}

	// Non-terminal states
	if transitions.IsTerminal(StateInit) {
		t.Error("init should not be terminal")
	}
	if transitions.IsTerminal(StateRunning) {
		t.Error("running should not be terminal")
	}

	// Terminal states
	if !transitions.IsTerminal(StateComplete) {
		t.Error("complete should be terminal")
	}
	if !transitions.IsTerminal(StateFailed) {
		t.Error("failed should be terminal")
	}
}
