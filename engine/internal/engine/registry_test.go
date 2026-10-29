package engine_test

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type clusterWriteBackend struct {
	timebox.Backend
	beforeAppend func(timebox.AppendRequest) error
}

var ErrClusterWrite = errors.New("cluster write unavailable")

func TestRegisterEmptyScripts(t *testing.T) {
	tests := []struct {
		language string
		kind     string
	}{
		{language: api.ScriptLangLua, kind: "predicate"},
		{language: api.ScriptLangLua, kind: "match"},
		{language: api.ScriptLangArgyll, kind: "predicate"},
		{language: api.ScriptLangArgyll, kind: "match"},
	}
	for _, tt := range tests {
		t.Run(tt.language+"/"+tt.kind, func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				st := helpers.NewStepWithOutputs("invalid-script", "result")
				cfg := &api.ScriptConfig{Language: tt.language}
				if tt.kind == "predicate" {
					st.Predicate = cfg
				} else {
					st.Attributes["input"] = &api.AttributeSpec{
						Role:     api.RoleRequired,
						Required: &api.RequiredConfig{Match: cfg},
					}
				}
				err := env.Engine.RegisterStep(st)
				assert.ErrorIs(t, err, api.ErrScriptEmpty)
				cat, err := env.Engine.GetCatalogState()
				assert.NoError(t, err)
				assert.NotContains(t, cat.Steps, st.ID)

				valid := helpers.NewStepWithOutputs(st.ID, "result")
				assert.NoError(t, env.Engine.RegisterStep(valid))
				err = env.Engine.UpdateStep(st)
				assert.ErrorIs(t, err, api.ErrScriptEmpty)
				cat, err = env.Engine.GetCatalogState()
				assert.NoError(t, err)
				assert.Nil(t, cat.Steps[st.ID].Predicate)
				assert.NotContains(t,
					cat.Steps[st.ID].Attributes, api.Name("input"))
			})
		})
	}
}

func TestRegisterNonBooleanPredicates(t *testing.T) {
	tests := []struct {
		kind    string
		message string
	}{
		{
			kind:    "predicate",
			message: "invalid step: predicate: must return boolean, not number",
		},
		{
			kind: "match",
			message: `invalid step: attribute "input" match: ` +
				"must return boolean, not number",
		},
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				st := helpers.NewStepWithOutputs("non-boolean", "result")
				cfg := &api.ScriptConfig{
					Language: api.ScriptLangArgyll, Script: "1 + 2",
				}
				if tt.kind == "predicate" {
					st.Predicate = cfg
				} else {
					st.Attributes["input"] = &api.AttributeSpec{
						Role:     api.RoleRequired,
						Required: &api.RequiredConfig{Match: cfg},
					}
				}
				err := env.Engine.RegisterStep(st)
				assert.ErrorIs(t, err, script.ErrNotBoolean)
				assert.EqualError(t, err, tt.message)
				cat, err := env.Engine.GetCatalogState()
				assert.NoError(t, err)
				assert.NotContains(t, cat.Steps, st.ID)
			})
		})
	}
}

func TestRegisterStep(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := &api.Step{
			ID:   "test-step",
			Name: "Test Step",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"input":  {Role: api.RoleRequired, Type: api.TypeString},
				"output": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080/execute"},
			},
		}

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		steps, err := eng.ListSteps()
		assert.NoError(t, err)
		assert.Len(t, steps, 1)
		assert.Equal(t, api.StepID("test-step"), steps[0].ID)
	})
}

func TestRegisterStepRejectsUnconfiguredType(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		err := eng.RegisterStep(&api.Step{
			ID: "custom-step", Name: "Custom Step", Type: "custom",
		})
		assert.ErrorIs(t, err, api.ErrInvalidStep)
		assert.ErrorIs(t, err, api.ErrInvalidStepType)
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

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)

		retrievedStep, ok := cat.Steps["update-step"]
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

func TestRegisterSteps(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "step-a",
			Name: "Step A",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"foo": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080/a"},
			},
		}
		stepB := &api.Step{
			ID:   "step-b",
			Name: "Step B",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"foo": {Role: api.RoleRequired, Type: api.TypeString},
				"bar": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080/b"},
			},
		}

		assert.NoError(t, eng.RegisterSteps(stepA, stepB))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Contains(t, cat.Steps, stepA.ID)
		assert.Contains(t, cat.Steps, stepB.ID)
	})
}

func TestRegisterStepsRollback(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "step-a",
			Name: "Step A",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"foo": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080/a"},
			},
		}
		stepB := &api.Step{
			ID:   "step-b",
			Name: "Step B",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"foo": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080/b"},
			},
		}

		err := eng.RegisterSteps(stepA, stepB)
		assert.ErrorIs(t, err, api.ErrInvalidStep)
		assert.ErrorIs(t, err, engine.ErrTypeConflict)

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Empty(t, cat.Steps)
	})
}

func TestRegisterStepValidatesMappings(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("bad-mapping")
		st.Attributes["in"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
			Required: &api.RequiredConfig{
				Mapping: &api.MappingConfig{
					Script: &api.ScriptConfig{
						Language: api.ScriptLangArgyll,
						Script:   "value[",
					},
				},
			},
		}

		err := eng.RegisterStep(st)
		assert.ErrorIs(t, err, api.ErrInvalidStep)
		assert.ErrorContains(t, err, api.ErrInvalidMappingConfig.Error())
	})
}

func TestRegisterStepArgExInvalid(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewStepWithPredicate(
			"bad-argex-predicate", api.ScriptLangArgyll, "value[",
		)

		err := eng.RegisterStep(st)
		assert.ErrorIs(t, err, api.ErrInvalidStep)
		assert.ErrorContains(t, err, script.ErrArgExCompile.Error())
	})
}

func TestRegisterStepArgExValid(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewStepWithPredicate(
			"good-argex-predicate", api.ScriptLangArgyll, "flag ?? false",
		)

		err := eng.RegisterStep(st)
		assert.NoError(t, err)
	})
}

func TestArgExValidForScripts(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := &api.Step{
			ID:   "argex-script-step",
			Name: "ArgEx Script",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangArgyll,
				Script:   "value",
			},
			Attributes: api.AttributeSpecs{
				"value":  {Role: api.RoleRequired, Type: api.TypeString},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		err := eng.RegisterStep(st)
		assert.NoError(t, err)
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
		assert.ErrorIs(t, err, api.ErrStepExists)
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
		updatedStep.HTTP.Invoke.Endpoint = "http://test:8080/v2"

		err = eng.UpdateStep(updatedStep)
		assert.NoError(t, err)

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)

		updated, ok := cat.Steps["update-step"]
		assert.True(t, ok)
		assert.Equal(t, api.Name("Updated Name"), updated.Name)
	})
}

func TestUpdateStepNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("nonexistent")

		err := eng.UpdateStep(st)
		assert.ErrorIs(t, err, api.ErrStepNotFound)
	})
}

// TestStepHealthSettledAtomically proves registration and the health it implies
// commit together, so a failing health write aborts the registration
func TestStepHealthSettledAtomically(t *testing.T) {
	var fail atomic.Bool
	backend := &clusterWriteBackend{
		Backend: memory.Open(),
		beforeAppend: func(req timebox.AppendRequest) error {
			if fail.Load() && req.ID == events.ClusterKey &&
				len(req.Events) != 0 {
				return ErrClusterWrite
			}
			return nil
		},
	}
	helpers.WithTestBackend(t, backend,
		func(env *helpers.TestEngineEnv) {
			st := helpers.NewSimpleStep("atomic-health")

			fail.Store(true)
			assert.ErrorIs(t, env.Engine.RegisterStep(st), ErrClusterWrite)

			cat, err := env.Engine.GetCatalogState()
			assert.NoError(t, err)
			assert.NotContains(t, cat.Steps, st.ID,
				"step must not register when its health write fails")

			fail.Store(false)
			assert.NoError(t, env.Engine.RegisterStep(st))

			cat, err = env.Engine.GetCatalogState()
			assert.NoError(t, err)
			assert.Contains(t, cat.Steps, st.ID)
		})
}

func (b *clusterWriteBackend) Append(reqs ...timebox.AppendRequest) error {
	for _, req := range reqs {
		if err := b.beforeAppend(req); err != nil {
			return err
		}
	}
	return b.Backend.Append(reqs...)
}

func (b *clusterWriteBackend) NewStore(
	cfg timebox.Config,
) (*timebox.Store, error) {
	return timebox.NewStore(b, cfg)
}
