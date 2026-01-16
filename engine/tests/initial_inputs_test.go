package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestInitialWorkflowInputs verifies that workflows can start with
// pre-populated attributes that aren't produced by any step. These initial
// inputs are provided when the workflow starts and are available to all steps
func TestInitialWorkflowInputs(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		ctx := context.Background()

		// Step A (Goal): Requires "initialValue" and "configValue", produces
		// "result". Neither initialValue nor configValue are produced by any
		// step, they must be provided as initial workflow inputs
		stepA := helpers.NewTestStepWithArgs(
			[]api.Name{"initialValue", "configValue"},
			nil,
		)
		stepA.ID = "step-a"
		stepA.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(ctx, stepA))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-a"},
			Steps: api.Steps{
				"step-a": stepA,
			},
			Attributes: api.AttributeGraph{
				"initialValue": &api.AttributeEdges{
					Providers: []api.StepID{}, // No provider
					Consumers: []api.StepID{"step-a"},
				},
				"configValue": &api.AttributeEdges{
					Providers: []api.StepID{}, // No provider
					Consumers: []api.StepID{"step-a"},
				},
				"result": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{},
				},
			},
		}

		// Set up mock response for step A
		env.MockClient.SetResponse(stepA.ID, api.Args{
			"result": "computed from initial inputs",
		})

		// Start workflow with initial inputs
		initialInputs := api.Args{
			"initialValue": "user-provided",
			"configValue":  42,
		}

		flowID := api.FlowID("test-initial-inputs")
		err := env.Engine.StartFlow(
			ctx, flowID, plan, initialInputs, api.Metadata{},
		)
		assert.NoError(t, err)

		// Wait for workflow completion
		flow := env.WaitForFlowStatus(t, ctx, flowID, workflowTimeout)

		// Verify workflow completed successfully
		assert.Equal(t, api.FlowCompleted, flow.Status)

		// Verify step A completed
		execA := flow.Executions["step-a"]
		assert.NotNil(t, execA)
		assert.Equal(t, api.StepCompleted, execA.Status)

		// Verify step A was invoked
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1)
		assert.Contains(t, invocations, api.StepID("step-a"))

		// Verify final attributes contain initial values plus step A's output
		assert.Equal(t, "user-provided", flow.Attributes["initialValue"].Value)
		assert.Equal(t, float64(42), flow.Attributes["configValue"].Value)
		assert.Equal(
			t, "computed from initial inputs", flow.Attributes["result"].Value,
		)

		// Verify initial attributes have no producing step (provenance = empty)
		assert.Equal(t, api.StepID(""), flow.Attributes["initialValue"].Step)
		assert.Equal(t, api.StepID(""), flow.Attributes["configValue"].Step)

		// Verify step A's output has correct provenance
		assert.Equal(t, api.StepID("step-a"), flow.Attributes["result"].Step)
	})
}

func TestRequiredInputsMissing(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		ctx := context.Background()

		step := helpers.NewTestStepWithArgs([]api.Name{"customer_id"}, nil)
		step.ID = "requires-input"
		step.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, eng.RegisterStep(ctx, step))

		plan := &api.ExecutionPlan{
			Goals:    []api.StepID{step.ID},
			Steps:    api.Steps{step.ID: step},
			Required: []api.Name{"customer_id"},
		}

		err := eng.StartFlow(
			ctx, "wf-missing-required", plan, api.Args{}, api.Metadata{},
		)
		assert.ErrorIs(t, err, api.ErrRequiredInputs)
	})
}
