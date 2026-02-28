package engine_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestOptionalDefaults(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("default-step")
		step.Attributes = api.AttributeSpecs{
			"input": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
			"optional": {
				Role:    api.RoleOptional,
				Type:    api.TypeString,
				Default: "\"fallback\"",
			},
			"result": {
				Role: api.RoleOutput,
				Type: api.TypeString,
			},
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-defaults")
		fl := env.WaitForFlowStatus(flowID, func() {
			env.WaitFor(wait.WorkSucceeded(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			}), func() {
				err := env.Engine.StartFlow(flowID, plan,
					flowopt.WithInit(api.Args{"input": "value"}),
				)
				assert.NoError(t, err)
			})
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, "value", exec.Inputs["input"])
		assert.Equal(t, "fallback", exec.Inputs["optional"])
	})
}

func TestInputMapping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("mapped-input-step")
		step.Attributes = api.AttributeSpecs{
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

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-input-mapping")
		fl := env.WaitForFlowStatus(flowID, func() {
			err := env.Engine.StartFlow(flowID, plan,
				flowopt.WithInit(api.Args{
					"input": map[string]any{"foo": "value"},
				}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, "value", exec.Inputs["input"])
	})
}
func TestInputMappingWithRename(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("rename-input")
		step.Attributes = api.AttributeSpecs{
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

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-input-rename")
		fl := env.WaitForFlowStatus(flowID, func() {
			err := env.Engine.StartFlow(flowID, plan,
				flowopt.WithInit(api.Args{"user_email": "test@example.com"}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}
func TestPredicateFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewStepWithPredicate(
			"pred-fail", api.ScriptLangLua, "error('boom')",
		)

		assert.NoError(t, env.Engine.RegisterStep(step))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		exec := env.WaitForStepStatus("wf-pred-fail", step.ID, func() {
			err := env.Engine.StartFlow("wf-pred-fail", plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.StepFailed, exec.Status)
		assert.True(t, strings.Contains(exec.Error, "predicate"))
	})
}

func TestJPathPredicateMatchOnNull(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewStepWithPredicate(
			"jpath-null", api.ScriptLangJPath, "$.flag", "result",
		)
		step.Attributes["flag"] = &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeAny,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flow := env.WaitForFlowStatus("wf-jpath-null", func() {
			err := env.Engine.StartFlow("wf-jpath-null", plan,
				flowopt.WithInit(api.Args{"flag": nil}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t, api.StepCompleted, flow.Executions[step.ID].Status)

		assert.True(t, env.MockClient.WasInvoked(step.ID))
	})
}
func TestInputMappingWithAle(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("ale-input-map")
		step.Attributes = api.AttributeSpecs{
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

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{"result": float64(10)})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-ale-input")
		fl := env.WaitForFlowStatus(flowID, func() {
			err := env.Engine.StartFlow(flowID, plan,
				flowopt.WithInit(api.Args{"amount": float64(5)}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, float64(10), exec.Inputs["amount"])
	})
}
func TestPredicateExecution(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewStepWithPredicate(
			"predicate-step", api.ScriptLangAle, "true", "output",
		)

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"predicate-step", api.Args{"output": "executed"},
		)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-pred", plan)
		assert.NoError(t, err)
	})
}

func TestPredicateFalse(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewStepWithPredicate(
			"predicate-false-step", api.ScriptLangAle, "false", "output",
		)

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"predicate-false-step", api.Args{"output": "should-not-execute"},
		)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"predicate-false-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-pred-false", plan)
		assert.NoError(t, err)

		assert.False(t, env.MockClient.WasInvoked("predicate-false-step"))
	})
}
func TestLuaPredicate(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewStepWithPredicate(
			"lua-pred-step", api.ScriptLangLua, "return true", "output",
		)

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"lua-pred-step", api.Args{"output": "executed"},
		)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"lua-pred-step"},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("wf-lua-pred", plan)
		assert.NoError(t, err)
	})
}
func TestPredicateError(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewStepWithPredicate(
			"pred-err-step", api.ScriptLangLua,
			"error('predicate failed')", "output",
		)

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(
			"pred-err-step", api.Args{"output": "never"},
		)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"pred-err-step"},
			Steps: api.Steps{step.ID: step},
		}

		finalState := env.WaitForFlowStatus("wf-pred-err", func() {
			err = env.Engine.StartFlow("wf-pred-err", plan)
			assert.NoError(t, err)
		})

		assert.Equal(t, api.FlowFailed, finalState.Status)
		assert.False(t, env.MockClient.WasInvoked("pred-err-step"))
	})
}
