package engine_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/flow"
	"github.com/kode4food/argyll/engine/pkg/util"
)

func TestMemoizedParallelWork(t *testing.T) {
	for _, tt := range []struct {
		name        string
		items       []any
		invocations int
	}{
		{name: "cached", items: []any{"a", "b"}, invocations: 2},
		{name: "mixed", items: []any{"a", "b", "c", "d"}, invocations: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				assert.NoError(t, env.Engine.Start())
				st := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
				st.Handling = api.HandlingMemoized
				st.WorkConfig = &api.WorkConfig{Parallelism: 2}
				st.Attributes["items"].Required = &api.RequiredConfig{
					ForEach: true,
				}
				st.Attributes["items"].Type = api.TypeArray
				assert.NoError(t, env.Engine.RegisterStep(st))
				env.MockClient.SetResponse(st.ID, api.Args{})
				pl := &api.ExecutionPlan{
					Goals: api.Goals{Steps: []api.StepID{st.ID}},
					Steps: api.Steps{st.ID: st},
				}
				fl := env.WaitForFlowStatus("warm-items", func() {
					err := env.Engine.StartPlan("warm-items", pl,
						flow.WithInit(api.InitArgs{"items": {[]any{"a", "b"}}}),
					)
					assert.NoError(t, err)
				})
				assert.Equal(t, api.FlowCompleted, fl.Status)
				assert.Len(t, env.MockClient.GetInvocations(), 2)

				err := env.Engine.StartPlan("cached-items", pl,
					flow.WithInit(api.InitArgs{"items": {tt.items}}),
				)
				if !assert.NoError(t, err) {
					return
				}
				fl = env.WaitForTerminalFlow("cached-items")
				assert.Equal(t, api.FlowCompleted, fl.Status)
				assert.Len(t, env.MockClient.GetInvocations(), tt.invocations)
			})
		})
	}
}

func TestCachedChildFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		leaf := helpers.NewSimpleStep("cached-leaf")
		leaf.Handling = api.HandlingMemoized
		assert.NoError(t, env.Engine.RegisterStep(leaf))
		env.MockClient.SetResponse(leaf.ID, api.Args{})
		child := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{leaf.ID}},
			Steps: api.Steps{leaf.ID: leaf},
		}
		fl := env.WaitForFlowStatus("warm-leaf", func() {
			assert.NoError(t, env.Engine.StartPlan("warm-leaf", child))
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		sub := &api.Step{
			ID:   "cached-sub",
			Type: api.StepTypeFlow,
			Attributes: api.AttributeSpecs{
				"items": {
					Role:     api.RoleRequired,
					Type:     api.TypeArray,
					Required: &api.RequiredConfig{ForEach: true},
				},
			},
			WorkConfig: &api.WorkConfig{Parallelism: 2},
		}
		pl := &api.ExecutionPlan{
			Goals:    api.Goals{Steps: []api.StepID{sub.ID}},
			Steps:    api.Steps{sub.ID: sub},
			Children: map[api.StepID]*api.ExecutionPlan{sub.ID: child},
		}
		err := env.Engine.StartPlan("cached-parent", pl,
			flow.WithInit(api.InitArgs{"items": {[]any{"a", "b"}}}),
		)
		if !assert.NoError(t, err) {
			return
		}
		fl, err = env.Engine.GetFlowState("cached-parent")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Len(t, env.MockClient.GetInvocations(), 1)
	})
}

func TestDeferredAfterMemoHit(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		st := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		st.Handling = api.HandlingMemoized
		st.WorkConfig = &api.WorkConfig{Parallelism: 2}
		st.Attributes["items"].Required = &api.RequiredConfig{ForEach: true}
		st.Attributes["items"].Type = api.TypeArray
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		fl := env.WaitForFlowStatus("warm-deferred", func() {
			assert.NoError(t, env.Engine.StartPlan("warm-deferred", pl,
				flow.WithInit(api.InitArgs{"items": {[]any{"a"}}}),
			))
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		assert.NoError(t, env.Engine.UpdateStepHealth(
			st.ID, api.HealthUnhealthy, "offline",
		))
		assert.NoError(t, env.Engine.StartPlan("deferred", pl,
			flow.WithInit(api.InitArgs{"items": {[]any{"a", "c"}}}),
		))
		evs, err := env.Engine.GetFlowEvents("deferred")
		assert.NoError(t, err)
		deferred := 0
		for _, ev := range evs {
			if api.EventType(ev.Type) == api.EventTypeDispatchDeferred {
				deferred++
			}
		}
		assert.Equal(t, 1, deferred)
	})
}

func TestMemoizedWork(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewTestStep()
		st.Handling = api.HandlingMemoized
		assert.NoError(t, env.Engine.RegisterStep(st))

		env.MockClient.SetResponse(st.ID, api.Args{"output": "cached"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-memo-1", func() {
			assert.NoError(t, env.Engine.StartPlan("wf-memo-1", pl,
				flow.WithInit(api.InitArgs{"input": {"value"}}),
			))
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
		firstInvocations := env.MockClient.GetInvocations()
		assert.NotEmpty(t, firstInvocations)

		fl = env.WaitForFlowStatus("wf-memo-2", func() {
			assert.NoError(t, env.Engine.StartPlan("wf-memo-2", pl,
				flow.WithInit(api.InitArgs{"input": {"value"}}),
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-meta", func() {
			err := env.Engine.StartPlan("wf-meta", pl,
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
		cfg := util.MutableCopy(env.Config)
		cfg.NodeID = "node-2"

		peer, unsub, err := env.NewEngineWithConfig(cfg, env.Dependencies())
		assert.NoError(t, err)
		if !assert.NotNil(t, peer) {
			return
		}
		defer func() {
			unsub()
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-healthy-peer", func() {
			err := env.Engine.StartPlan("wf-healthy-peer", pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Len(t, ex.WorkItems, 1)
	})
}

func TestDispatchRecovery(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("recovering-health-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		assert.NoError(t, env.Engine.UpdateStepHealth(
			st.ID, api.HealthUnhealthy, "service down",
		))

		id := api.FlowID("wf-health-recovers")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.StepStarted(fs), func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl))
		})
		assert.False(t,
			env.MockClient.WaitForInvocation(st.ID, 120*time.Millisecond),
		)

		fl := env.WaitForFlowStatus(id, func() {
			assert.NoError(t, env.Engine.UpdateStepHealth(
				st.ID, api.HealthHealthy, "",
			))
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.True(t,
			env.MockClient.WaitForInvocation(st.ID, wait.DefaultTimeout),
		)
	})
}

func TestAsyncMetadata(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		flowMetadata := api.Metadata{
			"correlation_id": "cid-async-123",
		}

		st := helpers.NewSimpleStep("async-meta")
		st.HTTP.Invoke.Mode = api.ActionModeAsync

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.WorkStarted(api.FlowStep{
			FlowID: "wf-async-meta",
			StepID: st.ID,
		}), func() {
			err := env.Engine.StartPlan("wf-async-meta", pl,
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
		assert.Contains(t, webhook, "/callbacks/wf-async-meta/async-meta/")
		assert.True(t, strings.HasSuffix(webhook, "/invoke"))
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-script", func() {
			err := env.Engine.StartPlan("wf-script", pl,
				flow.WithInit(api.InitArgs{"x": {float64(2)}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepCompleted, ex.Status)
		assert.EqualValues(t, 6, ex.Outputs["result"])
	})
}

func TestScriptMapping(t *testing.T) {
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
					Required: &api.RequiredConfig{
						Mapping: &api.MappingConfig{Name: "inner_amount"},
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-script-mapped", func() {
			err := env.Engine.StartPlan("wf-script-mapped", pl,
				flow.WithInit(api.InitArgs{"amount": {float64(2)}}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepCompleted, ex.Status)
		_, hasOuter := ex.Inputs["amount"]
		assert.False(t, hasOuter)
		assert.Equal(t, float64(2), ex.Inputs["inner_amount"])
		assert.EqualValues(t, 6, ex.Outputs["result"])
	})
}

func TestUnsupportedStep(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := &api.Step{
			ID:         "bad-step-type",
			Name:       "Bad Step Type",
			Type:       api.StepType("bad-type"),
			Attributes: api.AttributeSpecs{},
		}
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		id := api.FlowID("wf-bad-step-type")

		env.WaitFor(wait.FlowFailed(id), func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl))
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
		st.Attributes["items"].Required = &api.RequiredConfig{ForEach: true}
		st.Attributes["items"].Type = api.TypeArray
		st.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		err := env.Engine.StartPlan("wf-parallel", pl,
			flow.WithInit(api.InitArgs{"items": {[]any{"a", "b", "c"}}}),
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
func TestWorkPredicate(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		st.ID = "predicate-items"
		st.Predicate = &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script: "if type(items) ~= 'table' then error('boom') end; " +
				"return true",
		}
		st.Attributes["items"].Required = &api.RequiredConfig{ForEach: true}
		st.Attributes["items"].Type = api.TypeArray
		st.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-pred-work-item")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"items": {[]any{"a", "b"}}}),
			)
			assert.NoError(t, err)
		})

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepFailed, ex.Status)
		assert.Contains(t, ex.Error, "predicate")
		assert.Empty(t, ex.WorkItems)
		assert.True(t, ex.StartedAt.IsZero())
		assert.Equal(t, api.FlowFailed, fl.Status)
	})
}

func TestWorkPredicateFilters(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    any
		forEach  bool
		accepted int
		status   api.StepStatus
	}{
		{
			name:     "some",
			input:    []any{1, -1},
			forEach:  true,
			accepted: 1,
			status:   api.StepCompleted,
		},
		{
			name:    "none",
			input:   []any{-1, -2},
			forEach: true,
			status:  api.StepSkipped,
		},
		{
			name:    "empty",
			input:   []any{},
			forEach: true,
			status:  api.StepCompleted,
		},
		{
			name:     "single accepted",
			input:    1,
			accepted: 1,
			status:   api.StepCompleted,
		},
		{
			name:   "single rejected",
			input:  -1,
			status: api.StepSkipped,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				st := helpers.NewStepWithOutputs("filtered-items", "result")
				st.Type = api.StepTypeScript
				st.HTTP = nil
				st.Script = &api.ScriptConfig{
					Language: api.ScriptLangLua,
					Script:   `return {result = items}`,
				}
				st.Predicate = &api.ScriptConfig{
					Language: api.ScriptLangLua,
					Script:   `return items > 0`,
				}
				st.Attributes["items"] = &api.AttributeSpec{
					Role:     api.RoleRequired,
					Type:     api.TypeNumber,
					Required: &api.RequiredConfig{ForEach: tt.forEach},
				}
				if tt.forEach {
					st.Attributes["items"].Type = api.TypeArray
				}
				assert.NoError(t, env.Engine.RegisterStep(st))
				pl := &api.ExecutionPlan{
					Goals: api.Goals{Steps: []api.StepID{st.ID}},
					Steps: api.Steps{st.ID: st},
				}
				assert.NoError(t,
					env.Engine.StartPlan("filtered", pl,
						flow.WithInit(api.InitArgs{"items": {tt.input}})))
				fl, err := env.Engine.GetFlowState("filtered")
				assert.NoError(t, err)
				assert.Equal(t, api.FlowCompleted, fl.Status)
				ex := fl.Executions[st.ID]
				assert.Equal(t, tt.status, ex.Status)
				assert.Len(t, ex.WorkItems, tt.accepted)
				for _, work := range ex.WorkItems {
					assert.Equal(t, api.WorkSucceeded, work.Status)
					assert.EqualValues(t, 1, work.Outputs["result"])
				}
			})
		})
	}
}

func TestHTTPExecution(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithOutputs("http-step", "output")

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse("http-step", api.Args{"output": "success"})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{"http-step"}},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartPlan("wf-http", pl)
		assert.NoError(t, err)
	})
}

func TestScriptExecution(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		st := helpers.NewScriptStep(
			"script-step", api.ScriptLangLua, "return {result = 42}", "result",
		)

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{"script-step"}},
			Steps: api.Steps{st.ID: st},
		}

		err = eng.StartPlan("wf-script", pl)
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
			Goals: api.Goals{Steps: []api.StepID{"lua-script-step"}},
			Steps: api.Steps{st.ID: st},
		}

		err = eng.StartPlan("wf-lua-script", pl)
		assert.NoError(t, err)
	})
}

func TestLuaScriptWithInputs(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		st := helpers.NewScriptStep(
			"lua-input-step", api.ScriptLangLua, "return {doubled = x * 2}",
			"doubled",
		)
		st.Attributes["x"] = &api.AttributeSpec{Role: api.RoleRequired}

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals:    api.Goals{Steps: []api.StepID{"lua-input-step"}},
			Steps:    api.Steps{st.ID: st},
			Required: []api.Name{"x"},
		}

		err = eng.StartPlan("wf-lua-input", pl,
			flow.WithInit(api.InitArgs{"x": {float64(21)}}),
		)
		assert.NoError(t, err)
	})
}

func TestCompetingNodesStartOnce(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cfg := util.MutableCopy(env.Config)
		cfg.NodeID = "node-compete"

		peer, unsub, err := env.NewEngineWithConfig(cfg, env.Dependencies())
		assert.NoError(t, err)
		if !assert.NotNil(t, peer) {
			return
		}
		defer func() {
			unsub()
			assert.NoError(t, peer.Stop())
		}()

		assert.NoError(t, env.Engine.Start())
		assert.NoError(t, peer.Start())

		st := helpers.NewSimpleStep("compete-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		// Both nodes can run the step, so both race to claim the work item
		assert.NoError(t,
			env.Engine.UpdateStepHealth(st.ID, api.HealthHealthy, ""),
		)
		assert.NoError(t, peer.UpdateStepHealth(st.ID, api.HealthHealthy, ""))

		id := api.FlowID("wf-compete")
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus(id, func() {
			assert.NoError(t, env.Engine.StartPlan(id, pl))
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, []api.StepID{st.ID}, env.MockClient.GetInvocations())
	})
}
