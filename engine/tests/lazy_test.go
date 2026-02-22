package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestLazyEvaluation tests that only steps required to reach the goal are
// executed, even when many other steps are registered
func TestLazyEvaluation(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Create 10 steps, but only A→B→C form path to goal
		// Step A: No inputs, produces "valueA"
		stepA := helpers.NewStepWithOutputs("step-a", "valueA")

		// Step B: Requires "valueA", produces "valueB"
		stepB := helpers.NewTestStepWithArgs([]api.Name{"valueA"}, nil)
		stepB.ID = "step-b"
		stepB.Attributes["valueB"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Step C (Goal): Requires "valueB", produces "result"
		stepC := helpers.NewTestStepWithArgs([]api.Name{"valueB"}, nil)
		stepC.ID = "step-c"
		stepC.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Create 7 additional unrelated steps (should NOT execute)
		stepD := helpers.NewStepWithOutputs("step-d", "valueD")
		stepE := helpers.NewStepWithOutputs("step-e", "valueE")
		stepF := helpers.NewStepWithOutputs("step-f", "valueF")
		stepG := helpers.NewStepWithOutputs("step-g", "valueG")
		stepH := helpers.NewStepWithOutputs("step-h", "valueH")
		stepI := helpers.NewStepWithOutputs("step-i", "valueI")
		stepJ := helpers.NewStepWithOutputs("step-j", "valueJ")

		// Register all 10 steps in the engine
		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))
		assert.NoError(t, env.Engine.RegisterStep(stepC))
		assert.NoError(t, env.Engine.RegisterStep(stepD))
		assert.NoError(t, env.Engine.RegisterStep(stepE))
		assert.NoError(t, env.Engine.RegisterStep(stepF))
		assert.NoError(t, env.Engine.RegisterStep(stepG))
		assert.NoError(t, env.Engine.RegisterStep(stepH))
		assert.NoError(t, env.Engine.RegisterStep(stepI))
		assert.NoError(t, env.Engine.RegisterStep(stepJ))

		// Set responses for all steps (even though only A,B,C should invoke)
		env.MockClient.SetResponse("step-a", api.Args{"valueA": "from-A"})
		env.MockClient.SetResponse("step-b", api.Args{"valueB": "from-B"})
		env.MockClient.SetResponse("step-c", api.Args{"result": "done"})
		env.MockClient.SetResponse("step-d", api.Args{"valueD": "from-D"})
		env.MockClient.SetResponse("step-e", api.Args{"valueE": "from-E"})
		env.MockClient.SetResponse("step-f", api.Args{"valueF": "from-F"})
		env.MockClient.SetResponse("step-g", api.Args{"valueG": "from-G"})
		env.MockClient.SetResponse("step-h", api.Args{"valueH": "from-H"})
		env.MockClient.SetResponse("step-i", api.Args{"valueI": "from-I"})
		env.MockClient.SetResponse("step-j", api.Args{"valueJ": "from-J"})

		// Create execution plan with ONLY the steps needed to reach goal. This
		// simulates the lazy evaluation - plan only includes A→B→C
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-c"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
				"step-c": stepC,
				// D through J deliberately NOT included in plan
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
			},
		}

		flowID := api.FlowID("test-lazy-eval")
		flow := env.WaitForFlowStatus(flowID, func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		// CRITICAL: Verify only 3 steps exist in executions (lazy evaluation)
		assert.Len(t, flow.Executions, 3)
		assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)
		assert.Equal(t, api.StepCompleted, flow.Executions["step-b"].Status)
		assert.Equal(t, api.StepCompleted, flow.Executions["step-c"].Status)

		// Verify unrelated steps are NOT in executions
		assert.NotContains(t, flow.Executions, api.StepID("step-d"))
		assert.NotContains(t, flow.Executions, api.StepID("step-e"))
		assert.NotContains(t, flow.Executions, api.StepID("step-f"))
		assert.NotContains(t, flow.Executions, api.StepID("step-g"))
		assert.NotContains(t, flow.Executions, api.StepID("step-h"))
		assert.NotContains(t, flow.Executions, api.StepID("step-i"))
		assert.NotContains(t, flow.Executions, api.StepID("step-j"))

		// Verify only required attributes were set
		assert.Len(t, flow.Attributes, 3)
		assert.Equal(t, "from-A", flow.Attributes["valueA"].Value)
		assert.Equal(t, "from-B", flow.Attributes["valueB"].Value)
		assert.Equal(t, "done", flow.Attributes["result"].Value)

		// CRITICAL: Verify only 3 steps were actually invoked (lazy evaluation)
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 3)
		assert.Contains(t, invocations, api.StepID("step-a"))
		assert.Contains(t, invocations, api.StepID("step-b"))
		assert.Contains(t, invocations, api.StepID("step-c"))
		assert.NotContains(t, invocations, api.StepID("step-d"))
		assert.NotContains(t, invocations, api.StepID("step-e"))
		assert.NotContains(t, invocations, api.StepID("step-f"))
		assert.NotContains(t, invocations, api.StepID("step-g"))
		assert.NotContains(t, invocations, api.StepID("step-h"))
		assert.NotContains(t, invocations, api.StepID("step-i"))
		assert.NotContains(t, invocations, api.StepID("step-j"))
	})
}
