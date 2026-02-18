package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRegisterStep(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
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

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		steps, err := eng.ListSteps()
		assert.NoError(t, err)
		assert.Len(t, steps, 1)
		assert.Equal(t, api.StepID("test-step"), steps[0].ID)
	})
}

func TestUpdateStepHealth(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("health-step")

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		err = eng.UpdateStepHealth("health-step", api.HealthHealthy, "")
		assert.NoError(t, err)

		state, err := eng.GetEngineState()
		assert.NoError(t, err)

		health, ok := state.Health["health-step"]
		assert.True(t, ok)
		assert.Equal(t, api.HealthHealthy, health.Status)
	})
}

func TestUpdateUnhealthy(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("unhealthy-step")

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		err = eng.UpdateStepHealth(
			"unhealthy-step", api.HealthUnhealthy, "connection refused",
		)
		assert.NoError(t, err)

		state, err := eng.GetEngineState()
		assert.NoError(t, err)

		health, ok := state.Health["unhealthy-step"]
		assert.True(t, ok)
		assert.Equal(t, api.HealthUnhealthy, health.Status)
		assert.Equal(t, "connection refused", health.Error)
	})
}

func TestUpdateStep(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("update-step")

		err := eng.RegisterStep(step)
		assert.NoError(t, err)

		updated := helpers.NewSimpleStep("update-step")
		updated.Name = "Updated"

		err = eng.UpdateStep(updated)
		assert.NoError(t, err)

		state, err := eng.GetEngineState()
		assert.NoError(t, err)

		retrievedStep, ok := state.Steps["update-step"]
		assert.True(t, ok)
		assert.Equal(t, api.Name("Updated"), retrievedStep.Name)
	})
}

func TestUpdateStepReplacesCycleDependencies(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := helpers.NewSimpleStep("step-a")
		stepA.Attributes = api.AttributeSpecs{
			"foo": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(stepA))

		stepB := helpers.NewSimpleStep("step-b")
		stepB.Attributes = api.AttributeSpecs{
			"foo": {Role: api.RoleRequired, Type: api.TypeString},
			"bar": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(stepB))

		stepC := helpers.NewSimpleStep("step-c")
		stepC.Attributes = api.AttributeSpecs{
			"bar": {Role: api.RoleRequired, Type: api.TypeString},
			"baz": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(stepC))

		updatedA := helpers.NewSimpleStep("step-a")
		updatedA.Attributes = api.AttributeSpecs{
			"baz": {Role: api.RoleRequired, Type: api.TypeString},
			"qux": {Role: api.RoleOutput, Type: api.TypeString},
		}

		assert.NoError(t, eng.UpdateStep(updatedA))
	})
}

func TestRegisterStepValidatesMappings(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewSimpleStep("bad-mapping")
		step.Attributes["in"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
			Mapping: &api.AttributeMapping{
				Script: &api.ScriptConfig{
					Language: api.ScriptLangJPath,
					Script:   "$..[",
				},
			},
		}

		err := eng.RegisterStep(step)
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorContains(t, err, api.ErrInvalidAttributeMapping.Error())
	})
}

func TestRegisterStepJPathInvalid(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewStepWithPredicate(
			"bad-jpath-predicate", api.ScriptLangJPath, "$..[",
		)

		err := eng.RegisterStep(step)
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorContains(t, err, engine.ErrJPathCompile.Error())
	})
}

func TestRegisterStepJPathValid(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := helpers.NewStepWithPredicate(
			"good-jpath-predicate", api.ScriptLangJPath, "$.flag",
		)

		err := eng.RegisterStep(step)
		assert.NoError(t, err)
	})
}

func TestJPathNotValidForScripts(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		step := &api.Step{
			ID:   "jpath-script-step",
			Name: "JPath Script",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangJPath,
				Script:   "$.value",
			},
			Attributes: api.AttributeSpecs{
				"value":  {Role: api.RoleRequired, Type: api.TypeString},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		err := eng.RegisterStep(step)
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorIs(t, err, api.ErrInvalidScriptLanguage)
	})
}
