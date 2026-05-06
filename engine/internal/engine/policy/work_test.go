package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestStepParallelism(t *testing.T) {
	assert.Equal(t, 1, StepParallelism(&api.Step{}))
	assert.Equal(t, 1, StepParallelism(&api.Step{
		WorkConfig: &api.WorkConfig{Parallelism: 0},
	}))
	assert.Equal(t, 3, StepParallelism(&api.Step{
		WorkConfig: &api.WorkConfig{Parallelism: 3},
	}))
}

func TestCountActiveWorkItems(t *testing.T) {
	items := api.WorkItems{
		"a": {Status: api.WorkActive},
		"b": {Status: api.WorkPending},
		"c": {Status: api.WorkActive},
	}
	assert.Equal(t, 2, CountActiveWorkItems(items))
}

func TestStepWorkCompletion(t *testing.T) {
	pending := StepWorkCompletion(api.WorkItems{
		"a": {Status: api.WorkSucceeded},
		"b": {Status: api.WorkPending},
	})
	assert.False(t, pending.Done)
	assert.False(t, pending.Failed)

	failed := StepWorkCompletion(api.WorkItems{
		"a": {Status: api.WorkSucceeded},
		"b": {Status: api.WorkFailed, Error: "bad"},
	})
	assert.True(t, failed.Done)
	assert.True(t, failed.Failed)
	assert.Equal(t, "bad", failed.FailureError)

	succeeded := StepWorkCompletion(api.WorkItems{
		"a": {Status: api.WorkSucceeded},
	})
	assert.True(t, succeeded.Done)
	assert.False(t, succeeded.Failed)
}
