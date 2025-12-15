package engine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRegisterStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := &api.Step{
		ID:   "test-step",
		Name: "Test Step",
		Type: api.StepTypeSync,
		Attributes: api.AttributeSpecs{
			"input":  {Role: api.RoleRequired, Type: api.TypeString},
			"output": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080/execute",
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	steps, err := env.Engine.ListSteps(context.Background())
	assert.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, api.StepID("test-step"), steps[0].ID)
}

func TestUpdateStepHealth(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("health-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	err = env.Engine.UpdateStepHealth(
		context.Background(), "health-step", api.HealthHealthy, "",
	)
	assert.NoError(t, err)

	state, err := env.Engine.GetEngineState(context.Background())
	assert.NoError(t, err)

	health, ok := state.Health["health-step"]
	assert.True(t, ok)
	assert.Equal(t, api.HealthHealthy, health.Status)
}

func TestUpdateUnhealthy(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("unhealthy-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	err = env.Engine.UpdateStepHealth(
		context.Background(), "unhealthy-step", api.HealthUnhealthy,
		"connection refused",
	)
	assert.NoError(t, err)

	state, err := env.Engine.GetEngineState(context.Background())
	assert.NoError(t, err)

	health, ok := state.Health["unhealthy-step"]
	assert.True(t, ok)
	assert.Equal(t, api.HealthUnhealthy, health.Status)
	assert.Equal(t, "connection refused", health.Error)
}

func TestUpdateStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("update-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	updated := helpers.NewSimpleStep("update-step")
	updated.Name = "Updated"

	err = env.Engine.UpdateStep(context.Background(), updated)
	assert.NoError(t, err)

	state, err := env.Engine.GetEngineState(context.Background())
	assert.NoError(t, err)

	retrievedStep, ok := state.Steps["update-step"]
	assert.True(t, ok)
	assert.Equal(t, api.Name("Updated"), retrievedStep.Name)
}
