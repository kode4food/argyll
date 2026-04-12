package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/script"
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

func TestRegisterStepIdempotent(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("dup-step")

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		_, beforeSeq, err := eng.GetCatalogStateSeq()
		assert.NoError(t, err)

		err = eng.RegisterStep(st)
		assert.NoError(t, err)

		_, afterSeq, err := eng.GetCatalogStateSeq()
		assert.NoError(t, err)
		assert.Equal(t, beforeSeq, afterSeq)
	})
}

func TestUpdateStep(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("update-step")

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		updated := helpers.NewSimpleStep("update-step")
		updated.Name = "Updated"

		err = eng.UpdateStep(updated)
		assert.NoError(t, err)

		state, err := eng.GetCatalogState()
		assert.NoError(t, err)

		retrievedStep, ok := state.Steps["update-step"]
		assert.True(t, ok)
		assert.Equal(t, api.Name("Updated"), retrievedStep.Name)
	})
}

func TestUpdateStepDefaultIdempotent(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("update-defaulted")

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		_, beforeSeq, err := eng.GetCatalogStateSeq()
		assert.NoError(t, err)

		updated := helpers.NewSimpleStep("update-defaulted")
		err = eng.UpdateStep(updated)
		assert.NoError(t, err)

		_, afterSeq, err := eng.GetCatalogStateSeq()
		assert.NoError(t, err)
		assert.Equal(t, beforeSeq, afterSeq)
	})
}

func TestUpdateStepCycles(t *testing.T) {
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

func TestRegisterStepRejectsFlowGoalCycles(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "flow-a",
			Name: "Flow A",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-b"},
			},
			Attributes: api.AttributeSpecs{},
		}
		stepB := &api.Step{
			ID:   "flow-b",
			Name: "Flow B",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-a"},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, eng.RegisterStep(stepA))

		err := eng.RegisterStep(stepB)
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorIs(t, err, engine.ErrCircularDependency)
	})
}

func TestCatalogTxRegister(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "step-a",
			Name: "Step A",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"foo": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080/a",
			},
		}
		stepB := &api.Step{
			ID:   "step-b",
			Name: "Step B",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"foo": {Role: api.RoleRequired, Type: api.TypeString},
				"bar": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080/b",
			},
		}

		err := eng.CatalogTx(func(tx *engine.CatalogTx) error {
			if err := tx.Register(stepA); err != nil {
				return err
			}
			return tx.Register(stepB)
		})
		assert.NoError(t, err)

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Contains(t, cat.Steps, stepA.ID)
		assert.Contains(t, cat.Steps, stepB.ID)
	})
}

func TestCatalogTxRollback(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "step-a",
			Name: "Step A",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"foo": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080/a",
			},
		}
		stepB := &api.Step{
			ID:   "step-b",
			Name: "Step B",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"foo": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080/b",
			},
		}

		err := eng.CatalogTx(func(tx *engine.CatalogTx) error {
			if err := tx.Register(stepA); err != nil {
				return err
			}
			return tx.Register(stepB)
		})
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorIs(t, err, engine.ErrTypeConflict)

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Empty(t, cat.Steps)
	})
}

func TestCatalogTxRejectsGoalCycle(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "flow-a",
			Name: "Flow A",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-b"},
			},
			Attributes: api.AttributeSpecs{},
		}
		stepB := &api.Step{
			ID:   "flow-b",
			Name: "Flow B",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-a"},
			},
			Attributes: api.AttributeSpecs{},
		}

		err := eng.CatalogTx(func(tx *engine.CatalogTx) error {
			if err := tx.Register(stepA); err != nil {
				return err
			}
			return tx.Register(stepB)
		})
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorIs(t, err, engine.ErrCircularDependency)

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Empty(t, cat.Steps)
	})
}

func TestUpdateStepRejectsGoalCycles(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "flow-a",
			Name: "Flow A",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-b"},
			},
			Attributes: api.AttributeSpecs{},
		}
		stepB := &api.Step{
			ID:   "flow-b",
			Name: "Flow B",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"leaf"},
			},
			Attributes: api.AttributeSpecs{},
		}
		leaf := helpers.NewSimpleStep("leaf")

		assert.NoError(t, eng.RegisterStep(leaf))
		assert.NoError(t, eng.RegisterStep(stepA))
		assert.NoError(t, eng.RegisterStep(stepB))

		updatedLeaf := &api.Step{
			ID:   "leaf",
			Name: "Leaf",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-a"},
			},
			Attributes: api.AttributeSpecs{},
		}

		err := eng.UpdateStep(updatedLeaf)
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorIs(t, err, engine.ErrCircularDependency)
	})
}

func TestRegisterStepValidatesMappings(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("bad-mapping")
		st.Attributes["in"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
			Mapping: &api.AttributeMapping{
				Script: &api.ScriptConfig{
					Language: api.ScriptLangJPath,
					Script:   "$..[",
				},
			},
		}

		err := eng.RegisterStep(st)
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorContains(t, err, api.ErrInvalidAttributeMapping.Error())
	})
}

func TestRegisterStepJPathInvalid(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewStepWithPredicate(
			"bad-jpath-predicate", api.ScriptLangJPath, "$..[",
		)

		err := eng.RegisterStep(st)
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorContains(t, err, script.ErrJPathCompile.Error())
	})
}

func TestRegisterStepJPathValid(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewStepWithPredicate(
			"good-jpath-predicate", api.ScriptLangJPath, "$.flag",
		)

		err := eng.RegisterStep(st)
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

func TestUnregisterStep(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("test-step")

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		err = eng.UnregisterStep("test-step")
		assert.NoError(t, err)

		steps, err := eng.ListSteps()
		assert.NoError(t, err)
		assert.Empty(t, steps)
	})
}

func TestListSteps(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("list-step")

		err := eng.RegisterStep(st)
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
		st := helpers.NewSimpleStep("dup-step")

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		err = eng.RegisterStep(st)
		assert.NoError(t, err)

		steps, err := eng.ListSteps()
		assert.NoError(t, err)
		assert.Len(t, steps, 1)
		assert.Equal(t, api.StepID("dup-step"), steps[0].ID)
	})
}

func TestRegisterConflictingStep(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("dup-step")

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		updatedStep := helpers.NewSimpleStep("dup-step")
		updatedStep.Name = "Updated Name"

		err = eng.RegisterStep(updatedStep)
		assert.ErrorIs(t, err, engine.ErrStepExists)
	})
}

func TestUpdateStepSuccess(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("update-step")
		st.Name = "Original Name"

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		updatedStep := helpers.NewSimpleStep("update-step")
		updatedStep.Name = "Updated Name"
		updatedStep.HTTP.Endpoint = "http://test:8080/v2"

		err = eng.UpdateStep(updatedStep)
		assert.NoError(t, err)

		state, err := eng.GetCatalogState()
		assert.NoError(t, err)

		updated, ok := state.Steps["update-step"]
		assert.True(t, ok)
		assert.Equal(t, api.Name("Updated Name"), updated.Name)
	})
}

func TestUpdateStepNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("nonexistent")

		err := eng.UpdateStep(st)
		assert.ErrorIs(t, err, engine.ErrStepNotFound)
	})
}
