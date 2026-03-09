package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestStartDuplicate(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("step-1")

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		err = eng.StartFlow("wf-dup", pl)
		assert.NoError(t, err)

		err = eng.StartFlow("wf-dup", pl)
		assert.ErrorIs(t, err, engine.ErrFlowExists)
	})
}

func TestStartFlowSchedulesWork(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("step-start")
		st.Type = api.StepTypeAsync
		st.HTTP.Timeout = 30 * api.Second

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("wf-start", pl)
		assert.NoError(t, err)

		flow, err := env.Engine.GetFlowState("wf-start")
		assert.NoError(t, err)

		exec := flow.Executions[st.ID]
		assert.Equal(t, api.StepActive, exec.Status)
		assert.Len(t, exec.WorkItems, 1)
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkActive, item.Status)
		}
	})
}

func TestStartMissingInput(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("step-needs-input")
		st.Attributes["required_value"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}

		pl := &api.ExecutionPlan{
			Goals:    []api.StepID{"step-needs-input"},
			Steps:    api.Steps{st.ID: st},
			Required: []api.Name{"required_value"},
		}

		err := eng.StartFlow("wf-missing", pl)
		assert.Error(t, err)
	})
}

func TestStartRejectsPartialParent(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("step-parent-meta")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		err := eng.StartFlow("wf-partial-parent-meta", pl,
			flow.WithMetadata(api.Metadata{
				api.MetaParentFlowID: "parent",
			}),
		)
		assert.ErrorContains(t, err, "partial parent metadata")
	})
}

func TestStartFlowSimple(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := &api.Step{
			ID:   "goal-step",
			Name: "Goal",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse("goal-step", api.Args{"result": "success"})

		pl := &api.ExecutionPlan{
			Goals:    []api.StepID{"goal-step"},
			Required: []api.Name{},
			Steps: api.Steps{
				"goal-step": step,
			},
		}

		err = env.Engine.StartFlow("wf-simple", pl)
		assert.NoError(t, err)

		flow, err := env.Engine.GetFlowState("wf-simple")
		assert.NoError(t, err)
		assert.NotNil(t, flow)
		assert.Equal(t, api.FlowID("wf-simple"), flow.ID)
	})
}
