package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestMemoizableStepUsesCache(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewTestStep()
		st.Memoizable = true
		assert.NoError(t, env.Engine.RegisterStep(st))

		env.MockClient.SetResponse(st.ID, api.Args{"output": "cached"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-memo-1", func() {
			assert.NoError(t, env.Engine.StartFlow("wf-memo-1", pl,
				flow.WithInit(api.Args{"input": "value"}),
			))
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
		firstInvocations := env.MockClient.GetInvocations()
		assert.NotEmpty(t, firstInvocations)

		fl = env.WaitForFlowStatus("wf-memo-2", func() {
			assert.NoError(t, env.Engine.StartFlow("wf-memo-2", pl,
				flow.WithInit(api.Args{"input": "value"}),
			))
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, len(firstInvocations))
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

		st := helpers.NewSimpleStep("meta-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-meta", func() {
			err := env.Engine.StartFlow("wf-meta", pl,
				flow.WithMetadata(flowMetadata),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		md := env.MockClient.LastMetadata(st.ID)
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

func TestDispatchOnHealthyPeer(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cfg := *env.Config
		cfg.Raft.LocalID = "node-2"
		cfg.Raft.Servers = append(cfg.Raft.Servers,
			raft.Server{ID: "node-2", Address: "127.0.0.1:9702"},
		)

		peer, unsubscribe, err := env.NewEngineWithConfig(&cfg, env.Dependencies())
		assert.NoError(t, err)
		if !assert.NotNil(t, peer) {
			return
		}
		defer func() {
			unsubscribe()
			assert.NoError(t, peer.Stop())
		}()

		assert.NoError(t, env.Engine.Start())
		assert.NoError(t, peer.Start())

		st := helpers.NewSimpleStep("healthy-peer-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		assert.NoError(t,
			env.Engine.UpdateStepHealth(
				st.ID, api.HealthUnhealthy, "connection refused",
			),
		)
		assert.NoError(t,
			peer.UpdateStepHealth(st.ID, api.HealthHealthy, ""),
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-healthy-peer", func() {
			err := env.Engine.StartFlow("wf-healthy-peer", pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Len(t, ex.WorkItems, 1)
	})
}

func TestAsyncMetadata(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		flowMetadata := api.Metadata{
			"correlation_id": "cid-async-123",
		}

		st := helpers.NewSimpleStep("async-meta")
		st.Type = api.StepTypeAsync

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.WorkStarted(api.FlowStep{
			FlowID: "wf-async-meta",
			StepID: st.ID,
		}), func() {
			err := env.Engine.StartFlow("wf-async-meta", pl,
				flow.WithMetadata(flowMetadata),
			)
			assert.NoError(t, err)
		})

		assert.True(t, env.MockClient.WaitForInvocation(
			st.ID, wait.DefaultTimeout,
		))

		md := env.MockClient.LastMetadata(st.ID)
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

		st := &api.Step{
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

		assert.NoError(t, env.Engine.RegisterStep(st))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-script", func() {
			err := env.Engine.StartFlow("wf-script", pl,
				flow.WithInit(api.Args{"x": float64(2)}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepCompleted, ex.Status)
		assert.Equal(t, 6, ex.Outputs["result"])
	})
}

func TestScriptWorkUsesMappedInputName(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
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

		assert.NoError(t, env.Engine.RegisterStep(st))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-script-mapped", func() {
			err := env.Engine.StartFlow("wf-script-mapped", pl,
				flow.WithInit(api.Args{"amount": float64(2)}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepCompleted, ex.Status)
		_, hasOuter := ex.Inputs["amount"]
		assert.False(t, hasOuter)
		assert.Equal(t, float64(2), ex.Inputs["inner_amount"])
		assert.Equal(t, 6, ex.Outputs["result"])
	})
}

func TestUnsupportedStepTypeFailsFlow(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:         "bad-step-type",
			Name:       "Bad Step Type",
			Type:       api.StepType("bad-type"),
			Attributes: api.AttributeSpecs{},
		}
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}
		id := api.FlowID("wf-bad-step-type")

		env.WaitFor(wait.FlowFailed(id), func() {
			assert.NoError(t, env.Engine.StartFlow(id, pl))
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)
		assert.Contains(t, fl.Error, "unsupported step type")
	})
}

func TestParallelWorkItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		st.ID = "parallel-items"
		st.WorkConfig = &api.WorkConfig{Parallelism: 2}
		st.Attributes["items"].ForEach = true
		st.Attributes["items"].Type = api.TypeArray
		st.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		err := env.Engine.StartFlow("wf-parallel", pl,
			flow.WithInit(api.Args{"items": []any{"a", "b", "c"}}),
		)
		assert.NoError(t, err)

		fl := env.WaitForTerminalFlow("wf-parallel")
		assert.Equal(t, api.FlowCompleted, fl.Status)
		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepCompleted, ex.Status)
		assert.Len(t, ex.WorkItems, 3)
		for _, work := range ex.WorkItems {
			assert.Equal(t, api.WorkSucceeded, work.Status)
		}
	})
}
func TestPredicateFailurePerWorkItem(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		st.ID = "predicate-items"
		st.Predicate = &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script: "if type(items) ~= 'table' then error('boom') end; " +
				"return true",
		}
		st.Attributes["items"].ForEach = true
		st.Attributes["items"].Type = api.TypeArray
		st.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-pred-work-item")
		env.WaitAfterAll(2, func(waits []*wait.Wait) {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.Args{"items": []any{"a", "b"}}),
			)
			assert.NoError(t, err)
			waits[0].ForEvent(wait.StepTerminal(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}))
			waits[1].ForEvent(wait.FlowTerminal(id))
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepFailed, ex.Status)
		assert.Contains(t, ex.Error, "predicate")
		assert.Equal(t, api.FlowFailed, fl.Status)
	})
}
func TestHTTPExecution(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithOutputs("http-step", "output")

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse("http-step", api.Args{"output": "success"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"http-step"},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("wf-http", pl)
		assert.NoError(t, err)
	})
}

func TestScriptExecution(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		st := helpers.NewScriptStep(
			"script-step", api.ScriptLangAle, "{:result 42}", "result",
		)

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"script-step"},
			Steps: api.Steps{st.ID: st},
		}

		err = eng.StartFlow("wf-script", pl)
		assert.NoError(t, err)
	})
}
func TestLuaScriptExecution(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		st := helpers.NewScriptStep(
			"lua-script-step", api.ScriptLangLua, "return {result = 42}",
			"result",
		)

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"lua-script-step"},
			Steps: api.Steps{st.ID: st},
		}

		err = eng.StartFlow("wf-lua-script", pl)
		assert.NoError(t, err)
	})
}

func TestAleScriptWithInputs(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		st := helpers.NewScriptStep(
			"ale-input-step", api.ScriptLangAle, "{:doubled (* x 2)}",
			"doubled",
		)
		st.Attributes["x"] = &api.AttributeSpec{Role: api.RoleRequired}

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals:    []api.StepID{"ale-input-step"},
			Steps:    api.Steps{st.ID: st},
			Required: []api.Name{"x"},
		}

		err = eng.StartFlow("wf-ale-input", pl,
			flow.WithInit(api.Args{"x": float64(21)}),
		)
		assert.NoError(t, err)
	})
}
