package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestOptionalDefaults(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("default-step")
		st.Attributes = api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"optional": {
				Role:    api.RoleOptional,
				Type:    api.TypeString,
				Default: `"fallback"`,
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-defaults")
		fl := env.WaitForFlowStatus(id, func() {
			env.WaitFor(wait.WorkSucceeded(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}), func() {
				err := env.Engine.StartFlow(id, pl,
					flow.WithInit(api.Args{"input": "value"}),
				)
				assert.NoError(t, err)
			})
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[st.ID]
		assert.Equal(t, "value", exec.Inputs["input"])
		assert.Equal(t, "fallback", exec.Inputs["optional"])
	})
}

func TestConstObjectDefault(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("const-object")
		st.Attributes = api.AttributeSpecs{
			"config": {
				Role:    api.RoleConst,
				Type:    api.TypeObject,
				Default: `{"name":"cfg","count":2}`,
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-const-object")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[st.ID]
		cfg, ok := exec.Inputs["config"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, map[string]any{
			"name":  "cfg",
			"count": float64(2),
		}, cfg)
	})
}

func TestInputMapping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("mapped-input-step")
		st.Attributes = api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeObject,
				Mapping: &api.AttributeMapping{
					Script: &api.ScriptConfig{
						Language: api.ScriptLangJPath,
						Script:   "$.foo",
					},
				},
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-input-mapping")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.Args{
					"input": map[string]any{"foo": "value"},
				}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[st.ID]
		assert.Equal(t, "value", exec.Inputs["input"])
	})
}
func TestInputMappingWithRename(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("rename-input")
		st.Attributes = api.AttributeSpecs{
			"user_email": {
				Role: api.RoleRequired,
				Type: api.TypeString,
				Mapping: &api.AttributeMapping{
					Name: "email",
				},
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-input-rename")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.Args{"user_email": "test@example.com"}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}
func TestPredicateFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithPredicate(
			"pred-fail", api.ScriptLangLua, "error('boom')",
		)

		assert.NoError(t, env.Engine.RegisterStep(st))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		exec := env.WaitForStepStatus("wf-pred-fail", st.ID, func() {
			err := env.Engine.StartFlow("wf-pred-fail", pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.StepFailed, exec.Status)
		assert.True(t, strings.Contains(exec.Error, "predicate"))
	})
}

func TestJPathPredicateMatchOnNull(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithPredicate(
			"jpath-null", api.ScriptLangJPath, "$.flag", "result",
		)
		st.Attributes["flag"] = &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeAny,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-jpath-null", func() {
			err := env.Engine.StartFlow("wf-jpath-null", pl,
				flow.WithInit(api.Args{"flag": nil}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, api.StepCompleted, fl.Executions[st.ID].Status)

		assert.True(t, env.MockClient.WasInvoked(st.ID))
	})
}
func TestInputMappingWithAle(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("ale-input-map")
		st.Attributes = api.AttributeSpecs{
			"amount": {
				Role: api.RoleRequired,
				Type: api.TypeNumber,
				Mapping: &api.AttributeMapping{
					Script: &api.ScriptConfig{
						Language: api.ScriptLangAle,
						Script:   "(* amount 2)",
					},
				},
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeNumber,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{"result": float64(10)})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-ale-input")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.Args{"amount": float64(5)}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[st.ID]
		assert.Equal(t, float64(10), exec.Inputs["amount"])
	})
}
func TestPredicateExecution(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewStepWithPredicate(
			"predicate-step", api.ScriptLangAle, "true", "output",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"predicate-step", api.Args{"output": "executed"},
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-step"},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("wf-pred", pl)
		assert.NoError(t, err)
	})
}

func TestPredicateFalse(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewStepWithPredicate(
			"predicate-false-step", api.ScriptLangAle, "false", "output",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"predicate-false-step", api.Args{"output": "should-not-execute"},
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-false-step"},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("wf-pred-false", pl)
		assert.NoError(t, err)

		assert.False(t, env.MockClient.WasInvoked("predicate-false-step"))
	})
}
func TestLuaPredicate(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewStepWithPredicate(
			"lua-pred-step", api.ScriptLangLua, "return true", "output",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"lua-pred-step", api.Args{"output": "executed"},
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"lua-pred-step"},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("wf-lua-pred", pl)
		assert.NoError(t, err)
	})
}
func TestPredicateError(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewStepWithPredicate(
			"pred-err-step", api.ScriptLangLua,
			"error('predicate failed')", "output",
		)

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"pred-err-step", api.Args{"output": "never"},
		)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"pred-err-step"},
			Steps: api.Steps{st.ID: st},
		}

		finalState := env.WaitForFlowStatus("wf-pred-err", func() {
			err = env.Engine.StartFlow("wf-pred-err", pl)
			assert.NoError(t, err)
		})

		assert.Equal(t, api.FlowFailed, finalState.Status)
		assert.False(t, env.MockClient.WasInvoked("pred-err-step"))
	})
}
