package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestOptionalInputsWithDefaults verifies that steps with optional inputs use
// default values when those inputs aren't provided by upstream steps
func TestOptionalInputsWithDefaults(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Step A: No inputs, produces "valueA"
		stepA := helpers.NewStepWithOutputs("step-a", "valueA")

		// Step B: Requires "valueA", has optional "config" input with default,
		// produces "result"
		stepB := helpers.NewTestStepWithArgs(
			[]api.Name{"valueA"},
			[]api.Name{"config"},
		)
		stepB.ID = "step-b"
		stepB.Attributes["config"].Default = `{"key": "default"}`
		stepB.Attributes["config"].Type = api.TypeObject
		stepB.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))

		// Set mock responses - Step A produces valueA
		// Step B will receive valueA and default config value
		env.MockClient.SetResponse("step-a", api.Args{"valueA": "from-A"})
		env.MockClient.SetResponse("step-b", api.Args{"result": "done"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-b"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
			},
			Attributes: api.AttributeGraph{
				"valueA": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-b"},
				},
				"config": &api.AttributeEdges{
					Providers: []api.StepID{}, // No provider
					Consumers: []api.StepID{"step-b"},
				},
				"result": &api.AttributeEdges{
					Providers: []api.StepID{"step-b"},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("test-optional-defaults")
		flow := env.WaitForFlowStatus(flowID, func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		// Verify flow completed successfully
		assert.Equal(t, api.FlowCompleted, flow.Status)

		// Verify both steps completed
		assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)
		assert.Equal(t, api.StepCompleted, flow.Executions["step-b"].Status)

		// Verify step A produced valueA
		assert.Equal(t, "from-A", flow.Attributes["valueA"].Value)

		// Verify step B produced result
		assert.Equal(t, "done", flow.Attributes["result"].Value)

		// Verify both steps were invoked
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 2)
		assert.Contains(t, invocations, api.StepID("step-a"))
		assert.Contains(t, invocations, api.StepID("step-b"))
	})
}
