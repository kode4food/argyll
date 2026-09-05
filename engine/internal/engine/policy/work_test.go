package policy_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const deadlineFallback = 30 * time.Second

func TestStepParallelism(t *testing.T) {
	assert.Equal(t, 1, policy.StepParallelism(&api.Step{}))
	assert.Equal(t, 1, policy.StepParallelism(&api.Step{
		WorkConfig: &api.WorkConfig{Parallelism: 0},
	}))
	assert.Equal(t, 3, policy.StepParallelism(&api.Step{
		WorkConfig: &api.WorkConfig{Parallelism: 3},
	}))
}

func TestCountActiveWorkItems(t *testing.T) {
	items := api.WorkItems{
		"a": {Status: api.WorkActive},
		"b": {Status: api.WorkPending},
		"c": {Status: api.WorkActive},
	}
	assert.Equal(t, 2, policy.CountActiveWorkItems(items))
}

func TestStepWorkCompletion(t *testing.T) {
	pending := policy.StepWorkCompletion(api.WorkItems{
		"a": {Status: api.WorkSucceeded},
		"b": {Status: api.WorkPending},
	})
	assert.False(t, pending.Done)
	assert.False(t, pending.Failed)

	failed := policy.StepWorkCompletion(api.WorkItems{
		"a": {Status: api.WorkSucceeded},
		"b": {Status: api.WorkFailed, Error: "bad"},
	})
	assert.True(t, failed.Done)
	assert.True(t, failed.Failed)
	assert.Equal(t, "bad", failed.FailureError)

	// Fail-fast: a single failure marks Done immediately,
	// even with active or pending siblings
	failFast := policy.StepWorkCompletion(api.WorkItems{
		"a": {Status: api.WorkActive},
		"b": {Status: api.WorkFailed, Error: "boom"},
		"c": {Status: api.WorkPending},
	})
	assert.True(t, failFast.Done)
	assert.True(t, failFast.Failed)
	assert.Equal(t, "boom", failFast.FailureError)

	succeeded := policy.StepWorkCompletion(api.WorkItems{
		"a": {Status: api.WorkSucceeded},
	})
	assert.True(t, succeeded.Done)
	assert.False(t, succeeded.Failed)
}

func TestWorkDeadline(t *testing.T) {
	started := time.Unix(1000, 0)
	st := &api.Step{
		ID:   "http-step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Timeout: 5000},
		},
	}

	at, ok := policy.WorkDeadline(st, api.WorkState{
		Status:    api.WorkActive,
		StartedAt: started,
	}, deadlineFallback)
	assert.True(t, ok)
	assert.Equal(t, started.Add(5*time.Second), at)

	// Async uses the same value, where it bounds the callback
	async := &api.Step{
		ID:   "async-step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Timeout: 5000,
				Mode:    api.ActionModeAsync,
			},
		},
	}
	at, ok = policy.WorkDeadline(async, api.WorkState{
		Status:    api.WorkActive,
		StartedAt: started,
	}, deadlineFallback)
	assert.True(t, ok)
	assert.Equal(t, started.Add(5*time.Second), at)

	// An unset timeout falls back the way the client does
	bare := &api.Step{
		ID:   "bare-step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{Invoke: api.HTTPAction{}},
	}
	at, ok = policy.WorkDeadline(bare, api.WorkState{
		Status:    api.WorkActive,
		StartedAt: started,
	}, deadlineFallback)
	assert.True(t, ok)
	assert.Equal(t, started.Add(deadlineFallback), at)

	// A script has no timeout of its own, so the fallback bounds a lost
	// invocation
	script := &api.Step{ID: "script-step", Type: api.StepTypeScript}
	at, ok = policy.WorkDeadline(script, api.WorkState{
		Status:    api.WorkActive,
		StartedAt: started,
	}, deadlineFallback)
	assert.True(t, ok)
	assert.Equal(t, started.Add(deadlineFallback), at)
}

func TestCompensationDeadline(t *testing.T) {
	started := time.Unix(1000, 0)
	st := &api.Step{
		ID:   "comp-step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke:     api.HTTPAction{Timeout: 5000},
			Compensate: &api.HTTPAction{Timeout: 9000},
		},
	}

	at, ok := policy.WorkDeadline(st, api.WorkState{
		Status:    api.WorkCompensating,
		StartedAt: started,
	}, deadlineFallback)
	assert.True(t, ok)
	assert.Equal(t, started.Add(9*time.Second), at)

	// Compensation without its own timeout inherits the invoke timeout
	inherited := &api.Step{
		ID:   "comp-inherit-step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke:     api.HTTPAction{Timeout: 5000},
			Compensate: &api.HTTPAction{},
		},
	}
	at, ok = policy.WorkDeadline(inherited, api.WorkState{
		Status:    api.WorkCompensating,
		StartedAt: started,
	}, deadlineFallback)
	assert.True(t, ok)
	assert.Equal(t, started.Add(5*time.Second), at)
}

func TestNoWorkDeadline(t *testing.T) {
	started := time.Unix(1000, 0)
	active := api.WorkState{Status: api.WorkActive, StartedAt: started}
	http := &api.Step{
		ID:   "http-step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{Invoke: api.HTTPAction{Timeout: 5000}},
	}

	// Without a fallback there is nothing to derive a deadline from
	script := &api.Step{ID: "script-step", Type: api.StepTypeScript}
	_, ok := policy.WorkDeadline(script, active, 0)
	assert.False(t, ok)

	_, ok = policy.WorkDeadline(nil, active, deadlineFallback)
	assert.False(t, ok)

	// Nothing is in flight, so nothing is awaited
	_, ok = policy.WorkDeadline(http, api.WorkState{
		Status:    api.WorkPending,
		StartedAt: started,
	}, deadlineFallback)
	assert.False(t, ok)

	_, ok = policy.WorkDeadline(http, api.WorkState{
		Status:    api.WorkSucceeded,
		StartedAt: started,
	}, deadlineFallback)
	assert.False(t, ok)

	_, ok = policy.WorkDeadline(http, api.WorkState{
		Status: api.WorkActive,
	}, deadlineFallback)
	assert.False(t, ok)
}

func TestWorkReadyToDispatch(t *testing.T) {
	now := time.Unix(1000, 0)
	st := &api.Step{ID: "step"}

	assert.True(t, policy.WorkReadyToDispatch(st, api.ExecutionState{
		WorkItems: api.WorkItems{"a": {Status: api.WorkPending}},
	}, now))

	// Parallelism is already spent on an active item
	assert.False(t, policy.WorkReadyToDispatch(st, api.ExecutionState{
		WorkItems: api.WorkItems{
			"a": {Status: api.WorkActive},
			"b": {Status: api.WorkPending},
		},
	}, now))

	// Pending, but not until its retry time arrives
	assert.False(t, policy.WorkReadyToDispatch(st, api.ExecutionState{
		WorkItems: api.WorkItems{
			"a": {Status: api.WorkPending, NextRetryAt: now.Add(time.Minute)},
		},
	}, now))
	assert.True(t, policy.WorkReadyToDispatch(st, api.ExecutionState{
		WorkItems: api.WorkItems{
			"a": {Status: api.WorkPending, NextRetryAt: now.Add(-time.Minute)},
		},
	}, now))

	assert.False(t, policy.WorkReadyToDispatch(st, api.ExecutionState{
		WorkItems: api.WorkItems{"a": {Status: api.WorkSucceeded}},
	}, now))
}
