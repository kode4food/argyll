package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestMemoizableStepUsesCache(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewTestStep()
		step.Memoizable = true
		assert.NoError(t, env.Engine.RegisterStep(step))

		env.MockClient.SetResponse(step.ID, api.Args{"output": "cached"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-memo-1", func() {
			assert.NoError(t, env.Engine.StartFlow("wf-memo-1", plan,
				flowopt.WithInit(api.Args{"input": "value"}),
			))
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		flow = env.WaitForFlowStatus("wf-memo-2", func() {
			assert.NoError(t, env.Engine.StartFlow("wf-memo-2", plan,
				flowopt.WithInit(api.Args{"input": "value"}),
			))
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1)
	})
}

func TestHTTPMetadata(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		flowMetadata := api.Metadata{
			"correlation_id": "cid-123",
			api.MetaFlowID:   "wrong-flow",
			api.MetaStepID:   "wrong-step",
		}

		step := helpers.NewSimpleStep("meta-step")
		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-meta", func() {
			err := env.Engine.StartFlow("wf-meta", plan,
				flowopt.WithMetadata(flowMetadata),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		md := env.MockClient.LastMetadata(step.ID)
		if assert.NotNil(t, md) {
			assert.Equal(t, "cid-123", md["correlation_id"])
			assert.Equal(t, api.FlowID("wf-meta"), md[api.MetaFlowID])
			assert.Equal(t, api.StepID("meta-step"), md[api.MetaStepID])
			assert.NotEmpty(t, md[api.MetaReceiptToken])
			_, hasWebhook := md[api.MetaWebhookURL]
			assert.False(t, hasWebhook)
		}
	})
}

func TestAsyncMetadata(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		flowMetadata := api.Metadata{
			"correlation_id": "cid-async-123",
		}

		step := helpers.NewSimpleStep("async-meta")
		step.Type = api.StepTypeAsync

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitFor(wait.WorkStarted(api.FlowStep{
			FlowID: "wf-async-meta",
			StepID: step.ID,
		}), func() {
			err := env.Engine.StartFlow("wf-async-meta", plan,
				flowopt.WithMetadata(flowMetadata),
			)
			assert.NoError(t, err)
		})

		assert.True(t, env.MockClient.WaitForInvocation(
			step.ID, wait.DefaultTimeout,
		))

		md := env.MockClient.LastMetadata(step.ID)
		assert.NotNil(t, md)
		assert.Equal(t, "cid-async-123", md["correlation_id"])

		webhook, ok := md[api.MetaWebhookURL].(string)
		assert.True(t, ok)
		assert.True(t, strings.Contains(webhook, "wf-async-meta"))
		assert.True(t, strings.Contains(webhook, "async-meta"))
	})
}

func TestScriptWorkExecutes(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := &api.Step{
			ID:   "script-work",
			Name: "Script Work",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return { result = (x or 0) * 3 }",
			},
			Attributes: api.AttributeSpecs{
				"x":      {Role: api.RoleRequired, Type: api.TypeNumber},
				"result": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(step))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-script", func() {
			err := env.Engine.StartFlow("wf-script", plan,
				flowopt.WithInit(api.Args{"x": float64(2)}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		exec := flow.Executions[step.ID]
		assert.Equal(t, api.StepCompleted, exec.Status)
		assert.Equal(t, float64(6), exec.Outputs["result"])
	})
}

func TestScriptWorkUsesMappedInputName(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := &api.Step{
			ID:   "script-mapped-input",
			Name: "Script Mapped Input",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return { result = (inner_amount or 0) * 3 }",
			},
			Attributes: api.AttributeSpecs{
				"amount": {
					Role: api.RoleRequired,
					Type: api.TypeNumber,
					Mapping: &api.AttributeMapping{
						Name: "inner_amount",
					},
				},
				"result": {
					Role: api.RoleOutput,
					Type: api.TypeNumber,
				},
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(step))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-script-mapped", func() {
			err := env.Engine.StartFlow("wf-script-mapped", plan,
				flowopt.WithInit(api.Args{"amount": float64(2)}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		exec := flow.Executions[step.ID]
		assert.Equal(t, api.StepCompleted, exec.Status)
		_, hasOuter := exec.Inputs["amount"]
		assert.False(t, hasOuter)
		assert.Equal(t, float64(2), exec.Inputs["inner_amount"])
		assert.Equal(t, float64(6), exec.Outputs["result"])
	})
}

func TestUnsupportedStepTypeFailsFlow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := &api.Step{
			ID:         "bad-step-type",
			Name:       "Bad Step Type",
			Type:       api.StepType("bad-type"),
			Attributes: api.AttributeSpecs{},
		}
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}
		flowID := api.FlowID("wf-bad-step-type")

		env.WaitFor(wait.FlowFailed(flowID), func() {
			assert.NoError(t, env.Engine.StartFlow(flowID, plan))
		})

		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, flow.Status)
		assert.Contains(t, flow.Error, "unsupported step type")
	})
}

func TestParallelWorkItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		step.ID = "parallel-items"
		step.WorkConfig = &api.WorkConfig{Parallelism: 2}
		step.Attributes["items"].ForEach = true
		step.Attributes["items"].Type = api.TypeArray
		step.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-parallel", func() {
			err := env.Engine.StartFlow("wf-parallel", plan,
				flowopt.WithInit(api.Args{"items": []any{"a", "b", "c"}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t, api.StepCompleted, flow.Executions[step.ID].Status)
	})
}
func TestPredicateFailurePerWorkItem(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		step.ID = "predicate-items"
		step.Predicate = &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script: "if type(items) ~= 'table' then error('boom') end; " +
				"return true",
		}
		step.Attributes["items"].ForEach = true
		step.Attributes["items"].Type = api.TypeArray
		step.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-pred-work-item")
		env.WaitAfterAll(2, func(waits []*wait.Wait) {
			err := env.Engine.StartFlow(flowID, plan,
				flowopt.WithInit(api.Args{"items": []any{"a", "b"}}),
			)
			assert.NoError(t, err)
			waits[0].ForEvent(wait.StepTerminal(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			}))
			waits[1].ForEvent(wait.FlowTerminal(flowID))
		})

		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		exec := flow.Executions[step.ID]
		assert.Equal(t, api.StepFailed, exec.Status)
		assert.Contains(t, exec.Error, "predicate")
		assert.Equal(t, api.FlowFailed, flow.Status)
	})
}
func TestHTTPExecution(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

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
