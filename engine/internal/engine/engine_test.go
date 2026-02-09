package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestNew(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		assert.NotNil(t, eng)
	})
}

func TestStartStop(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		err := eng.Stop()
		assert.NoError(t, err)
	})
}

func TestGetEngineState(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		state, err := eng.GetEngineState()
		assert.NoError(t, err)
		assert.NotNil(t, state)
		assert.NotNil(t, state.Steps)
		assert.NotNil(t, state.Health)
	})
}

func TestUnregisterStep(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("test-step")

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		err = eng.UnregisterStep("test-step")
		assert.NoError(t, err)

		steps, err := eng.ListSteps()
		assert.NoError(t, err)
		assert.Empty(t, steps)
	})
}

func TestHTTPExecution(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewStepWithOutputs("http-step", "output")

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse("http-step", api.Args{"output": "success"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"http-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-http", plan)
		assert.NoError(t, err)
	})
}

func TestScriptExecution(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		step := helpers.NewScriptStep(
			"script-step", api.ScriptLangAle, "{:result 42}", "result",
		)

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"script-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow("wf-script", plan)
		assert.NoError(t, err)
	})
}

func TestPredicateExecution(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewStepWithPredicate(
			"predicate-step", api.ScriptLangAle, "true", "output",
		)

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"predicate-step", api.Args{"output": "executed"},
		)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-pred", plan)
		assert.NoError(t, err)
	})
}

func TestPredicateFalse(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewStepWithPredicate(
			"predicate-false-step", api.ScriptLangAle, "false", "output",
		)

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"predicate-false-step", api.Args{"output": "should-not-execute"},
		)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-false-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-pred-false", plan)
		assert.NoError(t, err)

		assert.False(t, env.MockClient.WasInvoked("predicate-false-step"))
	})
}

func TestLuaScriptExecution(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		step := helpers.NewScriptStep(
			"lua-script-step", api.ScriptLangLua, "return {result = 42}",
			"result",
		)

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"lua-script-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow("wf-lua-script", plan)
		assert.NoError(t, err)
	})
}

func TestAleScriptWithInputs(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		step := helpers.NewScriptStep(
			"ale-input-step", api.ScriptLangAle, "{:doubled (* x 2)}",
			"doubled",
		)
		step.Attributes["x"] = &api.AttributeSpec{Role: api.RoleRequired}

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals:    []api.StepID{"ale-input-step"},
			Steps:    api.Steps{step.ID: step},
			Required: []api.Name{"x"},
		}

		err = eng.StartFlow("wf-ale-input", plan,
			flowopt.WithInit(api.Args{"x": float64(21)}),
		)
		assert.NoError(t, err)
	})
}

func TestLuaPredicate(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewStepWithPredicate(
			"lua-pred-step", api.ScriptLangLua, "return true", "output",
		)

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"lua-pred-step", api.Args{"output": "executed"},
		)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"lua-pred-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-lua-pred", plan)
		assert.NoError(t, err)
	})
}

func TestListSteps(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("list-step")

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		steps, err := eng.ListSteps()
		assert.NoError(t, err)
		assert.Len(t, steps, 1)
		assert.Equal(t, api.StepID("list-step"), steps[0].ID)
	})
}

func TestListStepsEmpty(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		steps, err := eng.ListSteps()
		assert.NoError(t, err)
		assert.Empty(t, steps)
	})
}

func TestRegisterDuplicateStep(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("dup-step")

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		err = eng.RegisterStep(step)
		assert.NoError(t, err)

		steps, err := eng.ListSteps()
		assert.NoError(t, err)
		assert.Len(t, steps, 1)
		assert.Equal(t, api.StepID("dup-step"), steps[0].ID)
	})
}

func TestRegisterConflictingStep(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("dup-step")

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		updatedStep := helpers.NewSimpleStep("dup-step")
		updatedStep.Name = "Updated Name"

		err = eng.RegisterStep(updatedStep)
		assert.ErrorIs(t, err, engine.ErrStepExists)
	})
}

func TestUpdateStepSuccess(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("update-step")
		step.Name = "Original Name"

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		updatedStep := helpers.NewSimpleStep("update-step")
		updatedStep.Name = "Updated Name"
		updatedStep.HTTP.Endpoint = "http://test:8080/v2"

		err = eng.UpdateStep(updatedStep)
		assert.NoError(t, err)

		state, err := eng.GetEngineState()
		assert.NoError(t, err)

		updated, ok := state.Steps["update-step"]
		assert.True(t, ok)
		assert.Equal(t, api.Name("Updated Name"), updated.Name)
	})
}

func TestUpdateStepNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("nonexistent")

		err := eng.UpdateStep(step)
		assert.ErrorIs(t, err, engine.ErrStepNotFound)
	})
}

func TestGetFlowState(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("state-step")

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"state-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = eng.StartFlow("wf-state", plan)
		assert.NoError(t, err)

		state, err := eng.GetFlowState("wf-state")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowID("wf-state"), state.ID)
		assert.NotNil(t, state.Status)
	})
}

func TestGetFlowStateNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		_, err := eng.GetFlowState("nonexistent")
		assert.ErrorIs(t, err, engine.ErrFlowNotFound)
	})
}

func TestEngineStopGraceful(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		err := eng.Stop()
		assert.NoError(t, err)
	})
}
