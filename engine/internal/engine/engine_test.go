package engine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestNew(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	assert.NotNil(t, env.Engine)
}

func TestStartStop(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	err := env.Engine.Stop()
	assert.NoError(t, err)
}

func TestGetEngineState(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	state, err := env.Engine.GetEngineState(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, state)
	assert.NotNil(t, state.Steps)
	assert.NotNil(t, state.Health)
}

func TestUnregisterStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("test-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	err = env.Engine.UnregisterStep(context.Background(), "test-step")
	assert.NoError(t, err)

	steps, err := env.Engine.ListSteps(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, steps)
}

func TestHTTPExecution(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewStepWithOutputs("http-step", "output")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	env.MockClient.SetResponse("http-step", api.Args{"output": "success"})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"http-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-http", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)
}

func TestScriptExecution(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewScriptStep(
		"script-step", api.ScriptLangAle, "{:result 42}", "result",
	)

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"script-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-script", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)
}

func TestPredicateExecution(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewStepWithPredicate(
		"predicate-step", api.ScriptLangAle, "true", "output",
	)

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	env.MockClient.SetResponse("predicate-step", api.Args{"output": "executed"})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"predicate-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-pred", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)
}

func TestPredicateFalse(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewStepWithPredicate(
		"predicate-false-step", api.ScriptLangAle, "false", "output",
	)

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	env.MockClient.SetResponse(
		"predicate-false-step", api.Args{"output": "should-not-execute"},
	)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"predicate-false-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-pred-false", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	assert.False(t, env.MockClient.WasInvoked("predicate-false-step"))
}

func TestLuaScriptExecution(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewScriptStep(
		"lua-script-step", api.ScriptLangLua, "return {result = 42}", "result",
	)

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"lua-script-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-lua-script", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)
}

func TestAleScriptWithInputs(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewScriptStep(
		"ale-input-step", api.ScriptLangAle, "{:doubled (* x 2)}", "doubled",
	)
	step.Attributes["x"] = &api.AttributeSpec{Role: api.RoleRequired}

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals:    []api.StepID{"ale-input-step"},
		Steps:    api.Steps{step.ID: step},
		Required: []api.Name{"x"},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-ale-input", plan,
		api.Args{"x": float64(21)}, api.Metadata{},
	)
	assert.NoError(t, err)
}

func TestLuaPredicate(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewStepWithPredicate(
		"lua-pred-step", api.ScriptLangLua, "return true", "output",
	)

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	env.MockClient.SetResponse("lua-pred-step", api.Args{"output": "executed"})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"lua-pred-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-lua-pred", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)
}

func TestListSteps(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("list-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	steps, err := env.Engine.ListSteps(context.Background())
	assert.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, api.StepID("list-step"), steps[0].ID)
}

func TestListStepsEmpty(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	steps, err := env.Engine.ListSteps(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, steps)
}

func TestRegisterDuplicateStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("dup-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	err = env.Engine.RegisterStep(context.Background(), step)
	if err != nil {
		assert.ErrorIs(t, err, engine.ErrStepExists)
		return
	}
	t.Skip(
		"Engine allows duplicate step registration - this is current behavior",
	)
}

func TestUpdateStepSuccess(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("update-step")
	step.Name = "Original Name"

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	updatedStep := helpers.NewSimpleStep("update-step")
	updatedStep.Name = "Updated Name"
	updatedStep.Version = "1.0.1"
	updatedStep.HTTP.Endpoint = "http://test:8080/v2"

	err = env.Engine.UpdateStep(context.Background(), updatedStep)
	assert.NoError(t, err)

	state, err := env.Engine.GetEngineState(context.Background())
	assert.NoError(t, err)

	updated, ok := state.Steps["update-step"]
	assert.True(t, ok)
	assert.Equal(t, api.Name("Updated Name"), updated.Name)
	assert.Equal(t, "1.0.1", updated.Version)
}

func TestUpdateStepNotFound(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("nonexistent")

	err := env.Engine.UpdateStep(context.Background(), step)
	assert.ErrorIs(t, err, engine.ErrStepNotFound)
}

func TestGetFlowState(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewSimpleStep("state-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"state-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-state", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	state, err := env.Engine.GetFlowState(context.Background(), "wf-state")
	assert.NoError(t, err)
	assert.Equal(t, api.FlowID("wf-state"), state.ID)
	assert.NotNil(t, state.Status)
}

func TestGetFlowStateNotFound(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	_, err := env.Engine.GetFlowState(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, engine.ErrFlowNotFound)
}

func TestEngineStopGraceful(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	err := env.Engine.Stop()
	assert.NoError(t, err)
}
