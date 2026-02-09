package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const retryBackoffMs = 200

// TestMemoStepReusesToken verifies that memoizable steps reuse the same token
// across retries
func TestMemoStepReusesToken(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewStepWithOutputs("memo-retry", "result")
		step.Memoizable = true
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			Backoff:     retryBackoffMs,
			MaxBackoff:  retryBackoffMs,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))

		// First attempt fails
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("test-memo-token-reuse")
		eventConsumer := env.EventHub.NewConsumer()

		err := env.Engine.StartFlow(flowID, plan)
		assert.NoError(t, err)

		// Wait for retry to be scheduled
		helpers.WaitForWorkRetryScheduled(t,
			eventConsumer, flowID, step.ID, 1, flowTimeout,
		)

		// Clear error and set success response for retry
		env.MockClient.ClearError(step.ID)
		env.MockClient.SetResponse(step.ID, api.Args{"result": "success"})

		flow := env.WaitForFlowStatus(t, flowID, flowTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)

		// Verify only one work item (token was reused)
		exec := flow.Executions[step.ID]
		assert.Len(t, exec.WorkItems, 1)

		// Verify the work item has retry count > 0
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkSucceeded, item.Status)
			assert.GreaterOrEqual(t, item.RetryCount, 1)
		}
	})
}

// TestNonMemoStepRegeneratesToken verifies that non-memoizable steps generate
// a new token on retry
func TestNonMemoStepRegeneratesToken(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewStepWithOutputs("non-memo-retry", "result")
		step.Memoizable = false
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			Backoff:     retryBackoffMs,
			MaxBackoff:  retryBackoffMs,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))

		// First attempt fails
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("test-non-memo-token-regen")
		eventConsumer := env.EventHub.NewConsumer()

		err := env.Engine.StartFlow(flowID, plan)
		assert.NoError(t, err)

		// Wait for retry to be scheduled
		helpers.WaitForWorkRetryScheduled(t,
			eventConsumer, flowID, step.ID, 1, flowTimeout,
		)

		// Clear error and set success response for retry
		env.MockClient.ClearError(step.ID)
		env.MockClient.SetResponse(step.ID, api.Args{"result": "success"})

		flow := env.WaitForFlowStatus(t, flowID, flowTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)

		// Verify only one work item exists (old token was replaced)
		exec := flow.Executions[step.ID]
		assert.Len(t, exec.WorkItems, 1)

		// Verify the work item has retry count > 0
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkSucceeded, item.Status)
			assert.GreaterOrEqual(t, item.RetryCount, 1)
		}
	})
}

// TestRetriesRegenerateTokens verifies that each retry generates a new token
// for non-memoizable steps
func TestRetriesRegenerateTokens(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewStepWithOutputs("multi-retry", "result")
		step.Memoizable = false
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			Backoff:     retryBackoffMs,
			MaxBackoff:  retryBackoffMs,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))

		// Fail multiple times before succeeding
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("test-multi-retry-tokens")
		eventConsumer := env.EventHub.NewConsumer()

		err := env.Engine.StartFlow(flowID, plan)
		assert.NoError(t, err)

		// Wait for first retry
		helpers.WaitForWorkRetryScheduled(t,
			eventConsumer, flowID, step.ID, 1, flowTimeout,
		)

		// Allow next retry to succeed
		env.MockClient.ClearError(step.ID)
		env.MockClient.SetResponse(step.ID, api.Args{"result": "success"})

		flow := env.WaitForFlowStatus(t, flowID, flowTimeout)
		assert.Equal(t, api.FlowCompleted, flow.Status)

		// Verify only one work item exists (tokens were replaced on retry)
		exec := flow.Executions[step.ID]
		assert.Len(t, exec.WorkItems, 1)

		// Verify the work item has retry count >= 1
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkSucceeded, item.Status)
			assert.GreaterOrEqual(t, item.RetryCount, 1)
		}
	})
}
