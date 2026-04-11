package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
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

		fl, err := env.Engine.GetFlowState("wf-start")
		assert.NoError(t, err)

		exec := fl.Executions[st.ID]
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

		fl, err := env.Engine.GetFlowState("wf-simple")
		assert.NoError(t, err)
		assert.NotNil(t, fl)
		assert.Equal(t, api.FlowID("wf-simple"), fl.ID)
	})
}

func TestStartChildFlowUsesPlan(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		child := &api.Step{
			ID:   "child-step",
			Name: "Child Step",
			Type: api.StepTypeAsync,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(child))

		parent := &api.Step{
			ID:   "subflow-step",
			Name: "Subflow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{},
		}
		assert.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		assert.NoError(t, err)
		parentPlan, err := plan.Create(cat, []api.StepID{parent.ID}, api.Args{})
		assert.NoError(t, err)
		assert.NoError(t, env.Engine.StartFlow("wf-parent", parentPlan))

		updatedChild := &api.Step{
			ID:   "child-step",
			Name: "Child Step",
			Type: api.StepTypeAsync,
			Attributes: api.AttributeSpecs{
				"new-input": {Role: api.RoleRequired, Type: api.TypeString},
				"result":    {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}
		assert.NoError(t, env.Engine.UpdateStep(updatedChild))

		childID, err := env.Engine.StartChildFlow(
			api.FlowStep{
				FlowID: "wf-parent",
				StepID: parent.ID,
			},
			"token-1",
			parentPlan.Children[parent.ID],
			api.Args{},
			api.Metadata{},
		)
		assert.NoError(t, err)

		childFlow, err := env.Engine.GetFlowState(childID)
		assert.NoError(t, err)
		assert.Empty(t, childFlow.Plan.Required)
		if assert.Contains(t, childFlow.Plan.Steps, child.ID) {
			_, ok := childFlow.Plan.Steps[child.ID].Attributes["new-input"]
			assert.False(t, ok)
		}
	})
}

func TestStartChildFlowSetsParentMetadata(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		child := &api.Step{
			ID:   "child-step",
			Name: "Child Step",
			Type: api.StepTypeAsync,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(child))

		parent := &api.Step{
			ID:   "subflow-step",
			Name: "Subflow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{},
		}
		assert.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		assert.NoError(t, err)
		parentPlan, err := plan.Create(cat, []api.StepID{parent.ID}, api.Args{})
		assert.NoError(t, err)

		parentFS := api.FlowStep{
			FlowID: "wf-parent",
			StepID: parent.ID,
		}
		meta := api.Metadata{"source": "test"}
		childID, err := env.Engine.StartChildFlow(
			parentFS,
			"token-1",
			parentPlan.Children[parent.ID],
			api.Args{},
			meta,
		)
		assert.NoError(t, err)

		childFlow, err := env.Engine.GetFlowState(childID)
		assert.NoError(t, err)
		assert.Equal(t, meta["source"], childFlow.Metadata["source"])
		assert.Equal(t, parentFS.FlowID, childFlow.Metadata[api.MetaParentFlowID])
		assert.Equal(t, parentFS.StepID, childFlow.Metadata[api.MetaParentStepID])
		assert.Equal(t, api.Token("token-1"),
			childFlow.Metadata[api.MetaParentWorkItemToken])
	})
}

func TestStartChildFlowRejectsDuplicateID(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		child := &api.Step{
			ID:   "child-step",
			Name: "Child Step",
			Type: api.StepTypeAsync,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(child))

		parent := &api.Step{
			ID:   "subflow-step",
			Name: "Subflow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{},
		}
		assert.NoError(t, env.Engine.RegisterStep(parent))

		cat, err := env.Engine.GetCatalogState()
		assert.NoError(t, err)
		parentPlan, err := plan.Create(cat, []api.StepID{parent.ID}, api.Args{})
		assert.NoError(t, err)

		parentFS := api.FlowStep{
			FlowID: "wf-parent",
			StepID: parent.ID,
		}
		_, err = env.Engine.StartChildFlow(
			parentFS,
			"token-1",
			parentPlan.Children[parent.ID],
			api.Args{},
			api.Metadata{},
		)
		assert.NoError(t, err)

		_, err = env.Engine.StartChildFlow(
			parentFS,
			"token-1",
			parentPlan.Children[parent.ID],
			api.Args{},
			api.Metadata{},
		)
		assert.ErrorIs(t, err, engine.ErrFlowExists)
	})
}
