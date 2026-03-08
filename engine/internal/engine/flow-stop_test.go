package engine_test

import (
	"errors"
	"fmt"
	"testing"

	testify "github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestCompleteFlow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		a := assert.New(t)

		testify.NoError(t, env.Engine.Start())

		step := helpers.NewStepWithOutputs("complete-step", "result")

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		// Configure mock to return a result
		env.MockClient.SetResponse("complete-step", api.Args{"result": "final"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"complete-step"},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-complete", func() {
			err = env.Engine.StartFlow("wf-complete", plan)
			testify.NoError(t, err)
		})
		a.FlowStatus(flow, api.FlowCompleted)
	})
}

func TestFailFlow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		a := assert.New(t)

		testify.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("fail-step")

		err := env.Engine.RegisterStep(step)
		testify.NoError(t, err)

		env.MockClient.SetError("fail-step", errors.New("test error"))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"fail-step"},
			Steps: api.Steps{step.ID: step},
		}

		// Wait for flow to fail automatically
		env.WaitFor(wait.FlowFailed("wf-fail"), func() {
			err = env.Engine.StartFlow("wf-fail", plan)
			testify.NoError(t, err)
		})

		flow, err := env.Engine.GetFlowState("wf-fail")
		testify.NoError(t, err)
		a.FlowStatus(flow, api.FlowFailed)
		testify.Contains(t, flow.Error, "test error")
	})
}

func TestFlowStepChildSuccess(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

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

		testify.NoError(t, env.Engine.RegisterStep(child))
		testify.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		parentState := env.WaitForFlowStatus("parent-flow", func() {
			err := env.Engine.StartFlow("parent-flow", plan)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, parentState.Status)

		exec := parentState.Executions[parent.ID]
		if testify.NotNil(t, exec) && testify.NotNil(t, exec.WorkItems) {
			var tkn api.Token
			for t := range exec.WorkItems {
				tkn = t
				break
			}

			childID := api.FlowID(fmt.Sprintf(
				"%s:%s:%s", "parent-flow", parent.ID, tkn,
			))
			childState, err := env.Engine.GetFlowState(childID)
			testify.NoError(t, err)
			testify.Equal(t, api.FlowCompleted, childState.Status)

			testify.Equal(t,
				api.FlowID("parent-flow"), metaFlowID(childState.Metadata),
			)
			testify.Equal(t, parent.ID, metaStepID(childState.Metadata))
			testify.Equal(t, tkn, metaToken(childState.Metadata))
		}
	})
}

func TestFlowStepChildFailureParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

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

		testify.NoError(t, env.Engine.RegisterStep(child))
		testify.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		parentState := env.WaitForFlowStatus("parent-fail", func() {
			err := env.Engine.StartFlow("parent-fail", plan)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowFailed, parentState.Status)
	})
}

func TestFlowStepMissingGoalParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

		parent := &api.Step{
			ID:   "subflow-missing",
			Name: "Subflow Missing",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"missing-goal"},
			},
			Attributes: api.AttributeSpecs{},
		}

		testify.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		parentState := env.WaitForFlowStatus("parent-missing", func() {
			err := env.Engine.StartFlow("parent-missing", plan)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowFailed, parentState.Status)
	})
}

func TestFlowStepMapping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

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

		testify.NoError(t, env.Engine.RegisterStep(child))
		testify.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals:    []api.StepID{parent.ID},
			Steps:    api.Steps{parent.ID: parent},
			Required: []api.Name{"input"},
		}

		parentState := env.WaitForFlowStatus("parent-mapped", func() {
			err := env.Engine.StartFlow("parent-mapped", plan,
				flowopt.WithInit(api.Args{"input": float64(7)}),
			)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowCompleted, parentState.Status)

		exec := parentState.Executions[parent.ID]
		if testify.NotNil(t, exec) {
			testify.Equal(t, float64(7), exec.Outputs["output"])
		}
	})
}

func TestFlowStepMissingOutputParentFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		testify.NoError(t, env.Engine.Start())

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

		testify.NoError(t, env.Engine.RegisterStep(child))
		testify.NoError(t, env.Engine.RegisterStep(parent))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{parent.ID},
			Steps: api.Steps{parent.ID: parent},
		}

		parentState := env.WaitForFlowStatus("parent-missing-output", func() {
			err := env.Engine.StartFlow("parent-missing-output", plan)
			testify.NoError(t, err)
		})
		testify.Equal(t, api.FlowFailed, parentState.Status)
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
