package engine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestRegisterStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := &api.Step{
		ID:      "test-step",
		Name:    "Test Step",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: api.AttributeSpecs{
			"input":  {Role: api.RoleRequired, Type: api.TypeString},
			"output": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080/execute",
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	steps, err := env.Engine.ListSteps(context.Background())
	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, api.StepID("test-step"), steps[0].ID)
}

func TestUpdateStepHealth(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("health-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	err = env.Engine.UpdateStepHealth(
		context.Background(), "health-step", api.HealthHealthy, "",
	)
	require.NoError(t, err)

	state, err := env.Engine.GetEngineState(context.Background())
	require.NoError(t, err)

	health, ok := state.Health["health-step"]
	require.True(t, ok)
	assert.Equal(t, api.HealthHealthy, health.Status)
}

func TestUpdateUnhealthy(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("unhealthy-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	err = env.Engine.UpdateStepHealth(
		context.Background(), "unhealthy-step", api.HealthUnhealthy,
		"connection refused",
	)
	require.NoError(t, err)

	state, err := env.Engine.GetEngineState(context.Background())
	require.NoError(t, err)

	health, ok := state.Health["unhealthy-step"]
	require.True(t, ok)
	assert.Equal(t, api.HealthUnhealthy, health.Status)
	assert.Equal(t, "connection refused", health.Error)
}

func TestUpdateStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("update-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	updated := helpers.NewSimpleStep("update-step")
	updated.Name = "Updated"
	updated.Version = "2.0.0"

	err = env.Engine.UpdateStep(context.Background(), updated)
	require.NoError(t, err)

	state, err := env.Engine.GetEngineState(context.Background())
	require.NoError(t, err)

	retrievedStep, ok := state.Steps["update-step"]
	require.True(t, ok)
	assert.Equal(t, api.Name("Updated"), retrievedStep.Name)
	assert.Equal(t, "2.0.0", retrievedStep.Version)
}
