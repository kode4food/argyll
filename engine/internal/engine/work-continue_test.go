package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRetryPendingParallelism(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		step.ID = "retry-parallel"
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 500,
			MaxBackoff:  500,
			BackoffType: api.BackoffTypeFixed,
			Parallelism: 1,
		}
		step.Attributes["items"].ForEach = true
		step.Attributes["items"].Type = api.TypeArray
		step.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-retry-parallel")
		flow := env.WaitForFlowStatus(flowID, func() {
			env.WaitForCount(2, wait.WorkRetryScheduledAny(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			}), func() {
				err := env.Engine.StartFlow(flowID, plan,
					flowopt.WithInit(api.Args{
						"items": []any{"a", "b"},
					}),
				)
				assert.NoError(t, err)
			})

			env.MockClient.ClearError(step.ID)
			env.MockClient.SetResponse(step.ID, api.Args{"result": "ok"})
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)

		exec := flow.Executions[step.ID]
		assert.Equal(t, api.StepCompleted, exec.Status)
		assert.Len(t, exec.WorkItems, 2)
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkSucceeded, item.Status)
			assert.GreaterOrEqual(t, item.RetryCount, 1)
		}
	})
}
