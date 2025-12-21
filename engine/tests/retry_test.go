package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestRetryExhaustion tests that steps with MaxRetries eventually fail after
// exhausting all retry attempts
func TestRetryExhaustion(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	// Create a step that will always fail
	step := helpers.NewStepWithOutputs("failing-step", "result")
	step.WorkConfig = &api.WorkConfig{
		MaxRetries:  3,
		BackoffMs:   10,
		BackoffType: api.BackoffTypeFixed,
	}

	assert.NoError(t, env.Engine.RegisterStep(ctx, step))

	// Make the step always fail with a retryable error
	env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	flowID := api.FlowID("test-retry-exhaustion")
	err := env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	// Wait for workflow to fail (step exhausts retries)
	flow := env.WaitForFlowStatus(t, ctx, flowID, workflowTimeout)
	assert.Equal(t, api.FlowFailed, flow.Status)

	// Verify step failed after exhausting retries
	assert.Equal(t, api.StepFailed, flow.Executions[step.ID].Status)

	// Verify the step was invoked initial + MaxRetries times (1 + 3 = 4)
	invocations := env.MockClient.GetInvocations()
	assert.Len(t, invocations, 4)
}
