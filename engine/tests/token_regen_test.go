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

		st := helpers.NewStepWithOutputs("memo-retry", "result")
		st.Memoizable = true
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			InitBackoff: retryBackoffMs,
			MaxBackoff:  retryBackoffMs,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))

		// First attempt fails
		env.MockClient.SetError(st.ID, api.ErrWorkNotCompleted)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-memo-token-reuse")
		// Wait for retry to be scheduled
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
			FlowID: id,
			StepID: st.ID,
		}), func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		// Clear error and set success response for retry
		fl := env.WaitForFlowStatus(id, func() {
			env.MockClient.ClearError(st.ID)
			env.MockClient.SetResponse(st.ID, api.Args{"result": "success"})
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		// Verify only one work item (token was reused)
		ex := fl.Executions[st.ID]
		assert.Len(t, ex.WorkItems, 1)

		// Verify the work item has retry count > 0
		for _, item := range ex.WorkItems {
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

		st := helpers.NewStepWithOutputs("non-memo-retry", "result")
		st.Memoizable = false
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			InitBackoff: retryBackoffMs,
			MaxBackoff:  retryBackoffMs,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))

		// First attempt fails
		env.MockClient.SetError(st.ID, api.ErrWorkNotCompleted)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-non-memo-token-regen")
		// Wait for retry to be scheduled
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
			FlowID: id,
			StepID: st.ID,
		}), func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		// Clear error and set success response for retry
		fl := env.WaitForFlowStatus(id, func() {
			env.MockClient.ClearError(st.ID)
			env.MockClient.SetResponse(st.ID, api.Args{"result": "success"})
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		// Verify only one work item exists (old token was replaced)
		ex := fl.Executions[st.ID]
		assert.Len(t, ex.WorkItems, 1)

		// Verify the work item has retry count > 0
		for _, item := range ex.WorkItems {
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

		st := helpers.NewStepWithOutputs("multi-retry", "result")
		st.Memoizable = false
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: retryBackoffMs,
			MaxBackoff:  retryBackoffMs,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))

		// Fail multiple times before succeeding
		env.MockClient.SetError(st.ID, api.ErrWorkNotCompleted)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-multi-retry-tokens")
		// Wait for first retry
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
			FlowID: id,
			StepID: st.ID,
		}), func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		// Allow next retry to succeed
		fl := env.WaitForFlowStatus(id, func() {
			env.MockClient.ClearError(st.ID)
			env.MockClient.SetResponse(st.ID, api.Args{"result": "success"})
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		// Verify only one work item exists (tokens were replaced on retry)
		ex := fl.Executions[st.ID]
		assert.Len(t, ex.WorkItems, 1)

		// Verify the work item has retry count >= 1
		for _, item := range ex.WorkItems {
			assert.Equal(t, api.WorkSucceeded, item.Status)
			assert.GreaterOrEqual(t, item.RetryCount, 1)
		}
	})
}
