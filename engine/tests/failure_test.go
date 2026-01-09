package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestPartialWorkflowFailure tests that when one step in a workflow fails,
// independent steps complete successfully while dependent steps fail
func TestPartialWorkflowFailure(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	// Step A: No inputs, produces "valueB" and "valueC"
	stepA := helpers.NewStepWithOutputs("step-a", "valueB", "valueC")

	// Step B: Requires "valueB", will fail
	stepB := helpers.NewTestStepWithArgs([]api.Name{"valueB"}, nil)
	stepB.ID = "step-b"
	stepB.WorkConfig = &api.WorkConfig{MaxRetries: 0}
	stepB.Attributes["outputB"] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: api.TypeString,
	}

	// Step C: Requires "valueC", should complete successfully
	stepC := helpers.NewTestStepWithArgs([]api.Name{"valueC"}, nil)
	stepC.ID = "step-c"
	stepC.Attributes["outputC"] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: api.TypeString,
	}

	// Step D (Goal): Requires both "outputB" and "outputC"
	stepD := helpers.NewTestStepWithArgs([]api.Name{"outputB", "outputC"}, nil)
	stepD.ID = "step-d"
	stepD.Attributes["result"] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: api.TypeString,
	}

	assert.NoError(t, env.Engine.RegisterStep(ctx, stepA))
	assert.NoError(t, env.Engine.RegisterStep(ctx, stepB))
	assert.NoError(t, env.Engine.RegisterStep(ctx, stepC))
	assert.NoError(t, env.Engine.RegisterStep(ctx, stepD))

	// Set mock responses
	env.MockClient.SetResponse("step-a", api.Args{
		"valueB": "b-val",
		"valueC": "c-val",
	})
	env.MockClient.SetError("step-b", api.ErrWorkNotCompleted)
	env.MockClient.SetResponse("step-c", api.Args{"outputC": "C-result"})
	env.MockClient.SetResponse("step-d", api.Args{"result": "done"})

	// Create execution plan
	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-d"},
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
				Providers: []api.StepID{"step-a"},
				Consumers: []api.StepID{"step-c"},
			},
			"outputB": &api.AttributeEdges{
				Providers: []api.StepID{"step-b"},
				Consumers: []api.StepID{"step-d"},
			},
			"outputC": &api.AttributeEdges{
				Providers: []api.StepID{"step-c"},
				Consumers: []api.StepID{"step-d"},
			},
		},
	}

	flowID := api.FlowID("test-partial-failure")
	err := env.Engine.StartFlow(
		ctx, flowID, plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	// Wait for step C to complete (independent branch)
	env.WaitForStepStatus(t, ctx, flowID, "step-c", workflowTimeout)

	// Wait for step B to fail
	env.WaitForStepStatus(t, ctx, flowID, "step-b", workflowTimeout)

	// Now wait for workflow to fail
	flow := env.WaitForFlowStatus(t, ctx, flowID, workflowTimeout)
	assert.Equal(t, api.FlowFailed, flow.Status)

	// Verify step A completed (no dependencies, no errors)
	assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)

	// Verify step B failed (configured to fail)
	assert.Equal(t, api.StepFailed, flow.Executions["step-b"].Status)

	// Verify step C completed (independent of B's failure)
	assert.Equal(t, api.StepCompleted, flow.Executions["step-c"].Status)

	// Verify step D failed (depends on B which failed)
	assert.Equal(t, api.StepFailed, flow.Executions["step-d"].Status)

	// Verify attributes from successful steps were set
	assert.Equal(t, "b-val", flow.Attributes["valueB"].Value)
	assert.Equal(t, "c-val", flow.Attributes["valueC"].Value)
	assert.Equal(t, "C-result", flow.Attributes["outputC"].Value)
	assert.NotContains(t, flow.Attributes, "outputB")
	assert.NotContains(t, flow.Attributes, "result")

	// Verify correct steps were invoked
	invocations := env.MockClient.GetInvocations()
	assert.Contains(t, invocations, api.StepID("step-a"))
	assert.Contains(t, invocations, api.StepID("step-b"))
	assert.Contains(t, invocations, api.StepID("step-c"))
	assert.NotContains(t, invocations, api.StepID("step-d"))
}

func TestUnreachableStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	stepA := helpers.NewStepWithOutputs("provider-step", "value")
	stepA.WorkConfig = &api.WorkConfig{MaxRetries: 0}

	stepB := helpers.NewTestStepWithArgs([]api.Name{"value"}, nil)
	stepB.ID = "consumer-step"
	stepB.Attributes["result"] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: api.TypeString,
	}

	assert.NoError(t, env.Engine.RegisterStep(ctx, stepA))
	assert.NoError(t, env.Engine.RegisterStep(ctx, stepB))

	env.MockClient.SetError(stepA.ID, errors.New("boom"))

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{stepB.ID},
		Steps: api.Steps{
			stepA.ID: stepA,
			stepB.ID: stepB,
		},
		Attributes: api.AttributeGraph{
			"value": &api.AttributeEdges{
				Providers: []api.StepID{stepA.ID},
				Consumers: []api.StepID{stepB.ID},
			},
			"result": &api.AttributeEdges{
				Providers: []api.StepID{stepB.ID},
				Consumers: []api.StepID{},
			},
		},
	}

	err := env.Engine.StartFlow(
		ctx, "wf-unreachable", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	flow := env.WaitForFlowStatus(t, ctx, "wf-unreachable", workflowTimeout)
	assert.Equal(t, api.FlowFailed, flow.Status)

	assert.Equal(t, api.StepFailed, flow.Executions[stepA.ID].Status)
	assert.Equal(t, api.StepFailed, flow.Executions[stepB.ID].Status)
	assert.Equal(
		t,
		"required input no longer available",
		flow.Executions[stepB.ID].Error,
	)
	assert.NotContains(t, env.MockClient.GetInvocations(), stepB.ID)
}
