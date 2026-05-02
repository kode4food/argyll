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

		ex := fl.Executions[st.ID]
		assertRetrySucceeded(t, ex.WorkItems)
	})
}

// TestNonMemoStepReusesToken verifies that non-memoizable steps reuse the same
// token across retries
func TestNonMemoStepReusesToken(t *testing.T) {
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

		id := api.FlowID("test-non-memo-token-reuse")
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

		ex := fl.Executions[st.ID]
		assertRetrySucceeded(t, ex.WorkItems)
	})
}

// TestRetriesReuseToken verifies that retries preserve the work item token
func TestRetriesReuseToken(t *testing.T) {
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

		id := api.FlowID("test-multi-retry-token-reuse")
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

		ex := fl.Executions[st.ID]
		assertRetrySucceeded(t, ex.WorkItems)
	})
}

func assertRetrySucceeded(t *testing.T, items api.WorkItems) {
	t.Helper()

	assert.Len(t, items, 1)
	for _, item := range items {
		assert.Equal(t, api.WorkSucceeded, item.Status)
		assert.GreaterOrEqual(t, item.RetryCount, 1)
	}
}
