package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestMultipleGoals verifies that workflows with multiple goal steps execute
// all necessary dependencies (union of paths) and only those steps
func TestMultipleGoals(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		// Step A: Produces both "valueB" and "valueD"
		stepA := helpers.NewStepWithOutputs("step-a", "valueB", "valueD")

		// Step B: Requires "valueB", produces "valueC"
		stepB := helpers.NewTestStepWithArgs([]api.Name{"valueB"}, nil)
		stepB.ID = "step-b"
		stepB.Attributes["valueC"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Step C (Goal 1): Requires "valueC", produces "resultC"
		stepC := helpers.NewTestStepWithArgs([]api.Name{"valueC"}, nil)
		stepC.ID = "step-c"
		stepC.Attributes["resultC"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Step D (Goal 2): Requires "valueD", produces "resultD"
		stepD := helpers.NewTestStepWithArgs([]api.Name{"valueD"}, nil)
		stepD.ID = "step-d"
		stepD.Attributes["resultD"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Step E: Unrelated step (should NOT execute)
		stepE := helpers.NewStepWithOutputs("step-e", "valueE")

		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))
		assert.NoError(t, env.Engine.RegisterStep(stepC))
		assert.NoError(t, env.Engine.RegisterStep(stepD))
		assert.NoError(t, env.Engine.RegisterStep(stepE))

		env.MockClient.SetResponse("step-a", api.Args{
			"valueB": "from-A-B",
			"valueD": "from-A-D",
		})
		env.MockClient.SetResponse("step-b", api.Args{"valueC": "from-B"})
		env.MockClient.SetResponse("step-c", api.Args{"resultC": "done-C"})
		env.MockClient.SetResponse("step-d", api.Args{"resultD": "done-D"})
		env.MockClient.SetResponse("step-e", api.Args{"valueE": "from-E"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-c", "step-d"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
				"step-c": stepC,
				"step-d": stepD,
			},
			Attributes: api.AttributeGraph{
				"valueB": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-b"},
				},
				"valueC": &api.AttributeEdges{
					Providers: []api.StepID{"step-b"},
					Consumers: []api.StepID{"step-c"},
				},
				"valueD": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-d"},
				},
				"resultC": &api.AttributeEdges{
					Providers: []api.StepID{"step-c"},
					Consumers: []api.StepID{},
				},
				"resultD": &api.AttributeEdges{
					Providers: []api.StepID{"step-d"},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("test-multiple-goals")
		err := env.Engine.StartFlow(flowID, plan, api.Args{}, api.Metadata{})
		assert.NoError(t, err)

		flow := env.WaitForFlowStatus(t, flowID, workflowTimeout)

		// Verify workflow completed successfully
		assert.Equal(t, api.FlowCompleted, flow.Status)

		// Verify both goals (C and D) completed
		assert.Equal(t, api.StepCompleted, flow.Executions["step-c"].Status)
		assert.Equal(t, api.StepCompleted, flow.Executions["step-d"].Status)

		// Verify all steps in execution plan completed
		assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)
		assert.Equal(t, api.StepCompleted, flow.Executions["step-b"].Status)

		// Verify step E does NOT appear in executions
		assert.NotContains(t, flow.Executions, api.StepID("step-e"))

		// Verify all attributes from both paths are set
		assert.Equal(t, "from-A-B", flow.Attributes["valueB"].Value)
		assert.Equal(t, "from-B", flow.Attributes["valueC"].Value)
		assert.Equal(t, "from-A-D", flow.Attributes["valueD"].Value)
		assert.Equal(t, "done-C", flow.Attributes["resultC"].Value)
		assert.Equal(t, "done-D", flow.Attributes["resultD"].Value)

		// Verify step E was not invoked
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 4)
		assert.NotContains(t, invocations, api.StepID("step-e"))
	})
}
