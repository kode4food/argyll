package policy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
)

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
