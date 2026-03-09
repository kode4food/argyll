package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRetryPendingParallelism(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		st.ID = "retry-parallel"
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 500,
			MaxBackoff:  500,
			BackoffType: api.BackoffTypeFixed,
			Parallelism: 1,
		}
		st.Attributes["items"].ForEach = true
		st.Attributes["items"].Type = api.TypeArray
		st.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetError(st.ID, api.ErrWorkNotCompleted)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-retry-parallel")
		fl := env.WaitForFlowStatus(id, func() {
			env.WaitForCount(2, wait.WorkRetryScheduledAny(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}), func() {
				err := env.Engine.StartFlow(id, pl,
					flow.WithInit(api.Args{
						"items": []any{"a", "b"},
					}),
				)
				assert.NoError(t, err)
			})

			env.MockClient.ClearError(st.ID)
			env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		exec := fl.Executions[st.ID]
		assert.Equal(t, api.StepCompleted, exec.Status)
		assert.Len(t, exec.WorkItems, 2)
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkSucceeded, item.Status)
			assert.GreaterOrEqual(t, item.RetryCount, 1)
		}
	})
}
