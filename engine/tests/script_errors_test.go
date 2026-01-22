package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestScriptStepErrorAle verifies that Ale script steps that throw errors fail
// gracefully and cause the workflow to fail
func TestScriptStepErrorAle(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

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

		plan := &api.ExecutionPlan{
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

		flowID := api.FlowID("test-script-error-ale")
		err := env.Engine.StartFlow(flowID, plan, api.Args{}, api.Metadata{})
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, flowID, workflowTimeout)

		// Verify workflow failed
		assert.Equal(t, api.FlowFailed, flow.Status)

		// Verify step A completed
		assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)

		// Verify step B failed
		assert.Equal(t, api.StepFailed, flow.Executions["step-b"].Status)
		assert.NotEmpty(t, flow.Executions["step-b"].Error)

		// Verify step C failed (dependency not satisfied)
		assert.Equal(t, api.StepFailed, flow.Executions["step-c"].Status)

		// Verify only step A was invoked (B is script, C never starts)
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1)
		assert.Contains(t, invocations, api.StepID("step-a"))
	})
}

// TestScriptStepErrorLua verifies that Lua script steps that throw errors fail
// gracefully and cause the workflow to fail
func TestScriptStepErrorLua(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

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

		plan := &api.ExecutionPlan{
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

		flowID := api.FlowID("test-script-error-lua")
		err := env.Engine.StartFlow(flowID, plan, api.Args{}, api.Metadata{})
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, flowID, workflowTimeout)

		// Verify workflow failed
		assert.Equal(t, api.FlowFailed, flow.Status)

		// Verify step A completed
		assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)

		// Verify step B failed
		assert.Equal(t, api.StepFailed, flow.Executions["step-b"].Status)
		assert.Contains(t, flow.Executions["step-b"].Error, "error message")

		// Verify step C failed (dependency not satisfied)
		assert.Equal(t, api.StepFailed, flow.Executions["step-c"].Status)

		// Verify only step A was invoked (B is script, C never starts)
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1)
		assert.Contains(t, invocations, api.StepID("step-a"))
	})
}
