package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestForEachWorkItems verifies that steps with ForEach inputs create multiple
// work items (one per array element) and aggregate outputs correctly
func TestForEachWorkItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Step A: Produces array "items"
		stepA := helpers.NewStepWithOutputs("step-a", "items")
		stepA.Attributes["items"].Type = api.TypeArray

		// Step B: Has ForEach on "items" input, processes each item
		stepB := helpers.NewTestStepWithArgs(
			[]api.Name{"items"},
			nil,
		)
		stepB.ID = "step-b"
		stepB.Attributes["items"].ForEach = true
		stepB.Attributes["items"].Type = api.TypeArray
		stepB.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))

		// Step A produces array of items
		env.MockClient.SetResponse("step-a", api.Args{
			"items": []any{"apple", "banana", "cherry"},
		})

		// Step B will be invoked once per item
		env.MockClient.SetResponse("step-b", api.Args{
			"result": "processed",
		})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-b"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
			},
			Attributes: api.AttributeGraph{
				"items": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-b"},
				},
				"result": &api.AttributeEdges{
					Providers: []api.StepID{"step-b"},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("test-foreach")
		flow := env.WaitForFlowStatus(flowID, func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		// Verify flow completed successfully
		assert.Equal(t, api.FlowCompleted, flow.Status)

		// Verify both steps completed
		assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)
		assert.Equal(t, api.StepCompleted, flow.Executions["step-b"].Status)

		// Verify step B was invoked 3 times (once per array element)
		invocations := env.MockClient.GetInvocations()
		stepBInvocations := 0
		for _, id := range invocations {
			if id == "step-b" {
				stepBInvocations++
			}
		}
		assert.Equal(t, 3, stepBInvocations)

		// Verify aggregated output contains results
		result := flow.Attributes["result"].Value
		assert.NotNil(t, result)
	})
}
