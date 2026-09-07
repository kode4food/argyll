package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestStartDuplicate(t *testing.T) {
	t.Run("same goals is idempotent", func(t *testing.T) {
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
			assert.NoError(t, err)
		})
	})

	t.Run("different goals conflicts", func(t *testing.T) {
		helpers.WithStartedEngine(t, func(eng *engine.Engine) {
			st1 := helpers.NewSimpleStep("step-1")
			st2 := helpers.NewSimpleStep("step-2")

			assert.NoError(t, eng.RegisterStep(st1))
			assert.NoError(t, eng.RegisterStep(st2))

			pl1 := &api.ExecutionPlan{
				Goals: []api.StepID{"step-1"},
				Steps: api.Steps{st1.ID: st1},
			}
			pl2 := &api.ExecutionPlan{
				Goals: []api.StepID{"step-2"},
				Steps: api.Steps{st2.ID: st2},
			}

			err := eng.StartFlow("wf-conflict", pl1)
			assert.NoError(t, err)

			err = eng.StartFlow("wf-conflict", pl2)
			assert.ErrorIs(t, err, engine.ErrFlowExists)
		})
	})
}

func TestStartFlowSchedulesWork(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("step-start")
		st.HTTP.Invoke.Mode = api.ActionModeAsync
		st.HTTP.Invoke.Timeout = 30 * api.Second

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("_")
		env.WaitFor(wait.WorkStarted(api.FlowStep{
			FlowID: id,
			StepID: st.ID,
		}), func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepActive, ex.Status)
		assert.Len(t, ex.WorkItems, 1)
		for _, item := range ex.WorkItems {
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

		st := &api.Step{
			ID:   "goal-step",
			Name: "Goal",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
			},
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse("goal-step", api.Args{"result": "success"})

		pl := &api.ExecutionPlan{
			Goals:    []api.StepID{"goal-step"},
			Required: []api.Name{},
			Steps: api.Steps{
				"goal-step": st,
			},
		}

		err = env.Engine.StartFlow("wf-simple", pl)
		assert.NoError(t, err)

		fl, err := env.Engine.GetFlowState("wf-simple")
		assert.NoError(t, err)
		assert.NotNil(t, fl)
		assert.Equal(t, api.FlowID("wf-simple"), fl.ID)
	})
}

func TestStartChildFlowUsesPlan(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		child := helpers.NewSimpleStep("child")
		child.HTTP.Invoke.Mode = api.ActionModeAsync
		parent := &api.Step{
			ID:   "sub",
			Name: "Sub Flow",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{Goals: []api.StepID{child.ID}},
		}
		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))
		cat, err := env.Engine.GetCatalogState()
		assert.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match: env.Engine.Matcher, Children: env.Engine.Children,
			Steps: cat.Steps, Goals: []api.StepID{parent.ID},
		})
		assert.NoError(t, err)
		assert.NoError(t, env.Engine.StartFlow("parent", pl,
			flow.WithMetadata(api.Metadata{"source": "test"}),
		))

		updated := helpers.NewSimpleStep(child.ID)
		updated.HTTP.Invoke.Mode = api.ActionModeAsync
		updated.Attributes["new-input"] = &api.AttributeSpec{
			Role: api.RoleRequired, Type: api.TypeString,
		}
		assert.NoError(t, env.Engine.UpdateStep(updated))
		assert.NoError(t, env.Engine.Start())
		fl := helpers.WaitForFlowState(t, env.Engine, helpers.FlowStateQuery{
			FlowID: "parent", Timeout: wait.DefaultTimeout,
			Accept: func(fl api.FlowState) bool {
				for _, work := range fl.Executions[parent.ID].WorkItems {
					return work.Status == api.WorkActive
				}
				return false
			},
		})
		for tkn := range fl.Executions[parent.ID].WorkItems {
			fid := api.FlowID("parent:sub:" + tkn)
			child, err := env.Engine.GetFlowState(fid)
			assert.NoError(t, err)
			assert.Empty(t, child.Plan.Required)
			assert.NotContains(t, child.Plan.Steps["child"].Attributes,
				api.Name("new-input"))
			assert.Equal(t, "test", child.Metadata["source"])
			assert.Equal(t, api.FlowID("parent"),
				child.Metadata[api.MetaParentFlowID])
			assert.Equal(t, parent.ID, child.Metadata[api.MetaParentStepID])
			assert.Equal(t, tkn, child.Metadata[api.MetaParentWorkItemToken])
		}
	})
}
