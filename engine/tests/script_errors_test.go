package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestScriptStepErrorAle verifies that Ale script steps that throw errors fail
// gracefully and cause the flow to fail
func TestScriptStepErrorAle(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Step A: Produces "valueA"
		stepA := helpers.NewStepWithOutputs("step-a", "valueA")

		// Step B (Script - Ale): Consumes "valueA", causes runtime error
		stepB := helpers.NewScriptStep(
			"step-b",
			api.ScriptLangAle,
			`(/ 1 0)`,
			"valueB",
		)
		stepB.Attributes["valueA"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
		stepB.Attributes["valueB"].Type = api.TypeString

		// Step C: Depends on step B's output (won't execute)
		stepC := helpers.NewTestStepWithArgs(
			[]api.Name{"valueB"},
			nil,
		)
		stepC.ID = "step-c"
		stepC.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))
		assert.NoError(t, env.Engine.RegisterStep(stepC))

		env.MockClient.SetResponse("step-a", api.Args{"valueA": "from-A"})
		env.MockClient.SetResponse("step-c", api.Args{"result": "done"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-c"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
				"step-c": stepC,
			},
			Attributes: api.AttributeGraph{
				"valueA": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-b"},
				},
				"valueB": &api.AttributeEdges{
					Providers: []api.StepID{"step-b"},
					Consumers: []api.StepID{"step-c"},
				},
				"result": &api.AttributeEdges{
					Providers: []api.StepID{"step-c"},
					Consumers: []api.StepID{},
				},
			},
		}

		id := api.FlowID("test-script-error-ale")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		// Verify flow failed
		assert.Equal(t, api.FlowFailed, fl.Status)

		// Verify step A completed
		assert.Equal(t, api.StepCompleted, fl.Executions["step-a"].Status)

		// Verify step B failed
		assert.Equal(t, api.StepFailed, fl.Executions["step-b"].Status)
		assert.NotEmpty(t, fl.Executions["step-b"].Error)

		// Verify step C failed (dependency not satisfied)
		assert.Equal(t, api.StepFailed, fl.Executions["step-c"].Status)

		// Verify only step A was invoked (B is script, C never starts)
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1)
		assert.Contains(t, invocations, api.StepID("step-a"))
	})
}

// TestScriptStepErrorLua verifies that Lua script steps that throw errors fail
// gracefully and cause the flow to fail
func TestScriptStepErrorLua(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Step A: Produces "valueA"
		stepA := helpers.NewStepWithOutputs("step-a", "valueA")

		// Step B (Script - Lua): Consumes "valueA", throws error
		stepB := helpers.NewScriptStep(
			"step-b",
			api.ScriptLangLua,
			`error("Lua error message")`,
			"valueB",
		)
		stepB.Attributes["valueA"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
		stepB.Attributes["valueB"].Type = api.TypeString

		// Step C: Depends on step B's output (won't execute)
		stepC := helpers.NewTestStepWithArgs(
			[]api.Name{"valueB"},
			nil,
		)
		stepC.ID = "step-c"
		stepC.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))
		assert.NoError(t, env.Engine.RegisterStep(stepC))

		env.MockClient.SetResponse("step-a", api.Args{"valueA": "from-A"})
		env.MockClient.SetResponse("step-c", api.Args{"result": "done"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-c"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
				"step-c": stepC,
			},
			Attributes: api.AttributeGraph{
				"valueA": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-b"},
				},
				"valueB": &api.AttributeEdges{
					Providers: []api.StepID{"step-b"},
					Consumers: []api.StepID{"step-c"},
				},
				"result": &api.AttributeEdges{
					Providers: []api.StepID{"step-c"},
					Consumers: []api.StepID{},
				},
			},
		}

		id := api.FlowID("test-script-error-lua")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		// Verify flow failed
		assert.Equal(t, api.FlowFailed, fl.Status)

		// Verify step A completed
		assert.Equal(t, api.StepCompleted, fl.Executions["step-a"].Status)

		// Verify step B failed
		assert.Equal(t, api.StepFailed, fl.Executions["step-b"].Status)
		assert.Contains(t, fl.Executions["step-b"].Error, "error message")

		// Verify step C failed (dependency not satisfied)
		assert.Equal(t, api.StepFailed, fl.Executions["step-c"].Status)

		// Verify only step A was invoked (B is script, C never starts)
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1)
		assert.Contains(t, invocations, api.StepID("step-a"))
	})
}
