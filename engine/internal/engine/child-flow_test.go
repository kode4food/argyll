package engine_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestFlowStepChildSuccess(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		child := &api.Step{
			ID:   "child-step",
			Name: "Child Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-step",
			Name: "Subflow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		parentState := env.WaitForFlowStatus("parent-flow", func() {
			err := env.Engine.StartFlow("parent-flow", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, parentState.Status)

		exec := parentState.Executions[parent.ID]
		if assert.NotNil(t, exec) && assert.NotNil(t, exec.WorkItems) {
			var token api.Token
			for tkn := range exec.WorkItems {
				token = tkn
				break
			}

			childID := api.FlowID(fmt.Sprintf(
				"%s:%s:%s", "parent-flow", parent.ID, token,
			))
			childState, err := env.Engine.GetFlowState(childID)
			assert.NoError(t, err)
			assert.Equal(t, api.FlowCompleted, childState.Status)

			assert.Equal(t,
				api.FlowID("parent-flow"), metaFlowID(childState.Metadata),
			)
			assert.Equal(t, parent.ID, metaStepID(childState.Metadata))
			assert.Equal(t, token, metaToken(childState.Metadata))
		}
	})
}

func TestFlowStepChildFailureParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		child := &api.Step{
			ID:   "child-fail",
			Name: "Child Fail",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "error('boom')",
			},
			Attributes: api.AttributeSpecs{},
		}

		parent := &api.Step{
			ID:   "subflow-fail",
			Name: "Subflow Fail",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		parentState := env.WaitForFlowStatus("parent-fail", func() {
			err := env.Engine.StartFlow("parent-fail", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowFailed, parentState.Status)
	})
}

func TestFlowStepMissingGoalParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		parent := &api.Step{
			ID:   "subflow-missing",
			Name: "Subflow Missing",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"missing-goal"},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		parentState := env.WaitForFlowStatus("parent-missing", func() {
			err := env.Engine.StartFlow("parent-missing", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowFailed, parentState.Status)
	})
}

func TestFlowStepMapping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		child := &api.Step{
			ID:   "child-mapped",
			Name: "Child Mapped",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "{:child_out child_in}",
			},
			Attributes: api.AttributeSpecs{
				"child_in":  {Role: api.RoleRequired},
				"child_out": {Role: api.RoleOutput},
			},
		}

		parent := &api.Step{
			ID:   "subflow-mapped",
			Name: "Subflow Mapped",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{
				"input": {
					Role: api.RoleRequired,
					Mapping: &api.AttributeMapping{
						Name: "child_in",
					},
				},
				"output": {
					Role: api.RoleOutput,
					Mapping: &api.AttributeMapping{
						Name: "child_out",
					},
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals:    []api.StepID{parent.ID},
			Steps:    api.Steps{parent.ID: parent},
			Required: []api.Name{"input"},
		}

		parentState := env.WaitForFlowStatus("parent-mapped", func() {
			err := env.Engine.StartFlow("parent-mapped", plan,
				flowopt.WithInit(api.Args{"input": float64(7)}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, parentState.Status)

		exec := parentState.Executions[parent.ID]
		if assert.NotNil(t, exec) {
			assert.Equal(t, float64(7), exec.Outputs["output"])
		}
	})
}

func TestFlowStepMissingOutputParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		child := &api.Step{
			ID:   "child-empty",
			Name: "Child Empty",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{
				"child_out": {Role: api.RoleOutput},
			},
		}

		parent := &api.Step{
			ID:   "subflow-missing-output",
			Name: "Subflow Missing Output",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{
				"output": {
					Role: api.RoleOutput,
					Mapping: &api.AttributeMapping{
						Name: "child_out",
					},
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(child))
		assert.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		parentState := env.WaitForFlowStatus("parent-missing-output", func() {
			err := env.Engine.StartFlow("parent-missing-output", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowFailed, parentState.Status)
	})
}

func metaFlowID(meta api.Metadata) api.FlowID {
	switch val := meta[api.MetaParentFlowID].(type) {
	case api.FlowID:
		return val
	case string:
		return api.FlowID(val)
	default:
		return ""
	}
}

func metaStepID(meta api.Metadata) api.StepID {
	switch val := meta[api.MetaParentStepID].(type) {
	case api.StepID:
		return val
	case string:
		return api.StepID(val)
	default:
		return ""
	}
}

func metaToken(meta api.Metadata) api.Token {
	switch val := meta[api.MetaParentWorkItemToken].(type) {
	case api.Token:
		return val
	case string:
		return api.Token(val)
	default:
		return ""
	}
}
