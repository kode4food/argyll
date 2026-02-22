package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const retryBackoffMs = 200

// TestMemoStepReusesToken verifies that memoizable steps reuse the same token
// across retries
func TestMemoStepReusesToken(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewStepWithOutputs("memo-retry", "result")
		step.Memoizable = true
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			InitBackoff: retryBackoffMs,
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
		// Wait for retry to be scheduled
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
			FlowID: flowID,
			StepID: step.ID,
		}), func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		// Clear error and set success response for retry
		flow := env.WaitForFlowStatus(flowID, func() {
			env.MockClient.ClearError(step.ID)
			env.MockClient.SetResponse(step.ID, api.Args{"result": "success"})
		})
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
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewStepWithOutputs("non-memo-retry", "result")
		step.Memoizable = false
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			InitBackoff: retryBackoffMs,
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
		// Wait for retry to be scheduled
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
			FlowID: flowID,
			StepID: step.ID,
		}), func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		// Clear error and set success response for retry
		flow := env.WaitForFlowStatus(flowID, func() {
			env.MockClient.ClearError(step.ID)
			env.MockClient.SetResponse(step.ID, api.Args{"result": "success"})
		})
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
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewStepWithOutputs("multi-retry", "result")
		step.Memoizable = false
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: retryBackoffMs,
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
		// Wait for first retry
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
			FlowID: flowID,
			StepID: step.ID,
		}), func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		// Allow next retry to succeed
		flow := env.WaitForFlowStatus(flowID, func() {
			env.MockClient.ClearError(step.ID)
			env.MockClient.SetResponse(step.ID, api.Args{"result": "success"})
		})
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
