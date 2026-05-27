package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestGetCompiledPredicate(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithPredicate(
			"predicate-step", api.ScriptLangLua, "return true",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-step"},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.FlowStarted("wf-predicate"), func() {
			err = env.Engine.StartFlow("wf-predicate", pl)
			assert.NoError(t, err)
		})

		fs := api.FlowStep{
			FlowID: "wf-predicate", StepID: "predicate-step",
		}
		comp, err := env.Engine.GetCompiledPredicate(fs)
		assert.NoError(t, err)
		assert.NotNil(t, comp)
	})
}

func TestGetCompiledPredicateFlowNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		fs := api.FlowStep{FlowID: "nonexistent-flow", StepID: "step-id"}
		_, err := eng.GetCompiledPredicate(fs)
		assert.ErrorIs(t, err, engine.ErrFlowNotFound)
	})
}

func TestCreatePlanEmbedsChildPlans(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		child := &api.Step{
			ID:   "child",
			Name: "Child",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test",
				Timeout:  30 * api.Second,
			},
		}
		assert.NoError(t, eng.RegisterStep(child))

		parent := &api.Step{
			ID:   "parent",
			Name: "Parent",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{
				"wrapped": {
					Role: api.RoleOutput,
					Type: api.TypeString,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{Name: "result"},
					},
				},
			},
		}
		assert.NoError(t, eng.RegisterStep(parent))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)

		pl, err := plan.Create(
			eng.Matcher, eng.Children, cat,
			[]api.StepID{parent.ID}, api.InitArgs{},
		)
		assert.NoError(t, err)

		if assert.Contains(t, pl.Children, parent.ID) {
			childPlan := pl.Children[parent.ID]
			assert.NotNil(t, childPlan)
			assert.Contains(t, childPlan.Steps, child.ID)
		}
	})
}
