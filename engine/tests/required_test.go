package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestForEachWorkItems verifies that a required input with for_each and
// collect:all fans out one work item per collected provider value and
// aggregates outputs correctly
func TestForEachWorkItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Steps A1-A3: each produces one "items" value
		stepA1 := helpers.NewStepWithOutputs("step-a1", "items")
		stepA2 := helpers.NewStepWithOutputs("step-a2", "items")
		stepA3 := helpers.NewStepWithOutputs("step-a3", "items")

		// Step B: collects all "items" values and runs once per value
		stepB := helpers.NewTestStepWithArgs(
			[]api.Name{"items"},
			nil,
		)
		stepB.ID = "step-b"
		stepB.Attributes["items"].Required = &api.RequiredConfig{
			ForEach: true,
			Collect: api.InputCollectAll,
		}
		stepB.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(stepA1))
		assert.NoError(t, env.Engine.RegisterStep(stepA2))
		assert.NoError(t, env.Engine.RegisterStep(stepA3))
		assert.NoError(t, env.Engine.RegisterStep(stepB))

		env.MockClient.SetResponse("step-a1", api.Args{"items": "apple"})
		env.MockClient.SetResponse("step-a2", api.Args{"items": "banana"})
		env.MockClient.SetResponse("step-a3", api.Args{"items": "cherry"})

		// Step B will be invoked once per collected value
		env.MockClient.SetResponse("step-b", api.Args{
			"result": "processed",
		})

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{"step-b"}},
			Steps: api.Steps{
				"step-a1": stepA1,
				"step-a2": stepA2,
				"step-a3": stepA3,
				"step-b":  stepB,
			},
			Attributes: api.AttributeGraph{
				"items": &api.AttributeEdges{
					Providers: []api.StepID{"step-a1", "step-a2", "step-a3"},
					Consumers: []api.StepID{"step-b"},
				},
				"result": &api.AttributeEdges{
					Providers: []api.StepID{"step-b"},
					Consumers: []api.StepID{},
				},
			},
		}

		id := api.FlowID("test-foreach")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartPlan(id, pl)
			assert.NoError(t, err)
		})

		// Verify flow completed successfully
		assert.Equal(t, api.FlowCompleted, fl.Status)

		// Verify all steps completed
		assert.Equal(t, api.StepCompleted, fl.Executions["step-a1"].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions["step-a2"].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions["step-a3"].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions["step-b"].Status)

		// Verify step B was invoked 3 times (once per collected value)
		invocations := env.MockClient.GetInvocations()
		stepBInvocations := 0
		for _, inv := range invocations {
			if inv == "step-b" {
				stepBInvocations++
			}
		}
		assert.Equal(t, 3, stepBInvocations)

		// Verify aggregated output contains results
		result := fl.Attributes["result"][0].Value
		assert.NotNil(t, result)
	})
}
