package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestRetryExhaustion tests that steps with MaxRetries eventually fail after
// exhausting all retry attempts
func TestRetryExhaustion(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Create a step that will always fail
		st := helpers.NewStepWithOutputs("failing-step", "result")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 10,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))

		// Make the step always fail with a retryable error
		env.MockClient.SetError(st.ID, api.ErrWorkNotCompleted)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-retry-exhaustion")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowFailed, fl.Status)

		// Verify step failed after exhausting retries
		assert.Equal(t, api.StepFailed, fl.Executions[st.ID].Status)

		// Verify the step was invoked initial + MaxRetries times (1 + 3 = 4)
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 4)
	})
}
