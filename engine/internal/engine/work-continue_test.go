package engine_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/timebox/raft"

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

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepCompleted, ex.Status)
		assert.Len(t, ex.WorkItems, 2)
		for _, item := range ex.WorkItems {
			assert.Equal(t, api.WorkSucceeded, item.Status)
			assert.GreaterOrEqual(t, item.RetryCount, 1)
		}
	})
}

func TestRetryOnHealthyPeer(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cfg := *env.Config
		cfg.Raft.LocalID = "node-2"
		cfg.Raft.Servers = append(cfg.Raft.Servers,
			raft.Server{ID: "node-2", Address: "127.0.0.1:9702"},
		)

		peer, unsubscribe, err := env.NewEngineWithConfig(
			&cfg, env.Dependencies(),
		)
		assert.NoError(t, err)
		if !assert.NotNil(t, peer) {
			return
		}
		defer func() {
			unsubscribe()
			assert.NoError(t, peer.Stop())
		}()

		assert.NoError(t, env.Engine.Start())
		assert.NoError(t, peer.Start())

		st := helpers.NewSimpleStep("retry-shared")
		env.MockClient.SetResponse(st.ID, api.Args{"output": "ok"})

		assert.NoError(t,
			env.Engine.UpdateStepHealth(
				st.ID, api.HealthUnhealthy, "connection refused",
			),
		)
		assert.NoError(t,
			peer.UpdateStepHealth(st.ID, api.HealthHealthy, ""),
		)

		id := api.FlowID("wf-retry-shared")
		tkn := api.Token("retry-token")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(
			wait.WorkStarted(api.FlowStep{FlowID: id, StepID: st.ID}),
			func() {
				assert.NoError(t, env.RaiseFlowEvents(
					id,
					helpers.FlowEvent{
						Type: api.EventTypeFlowStarted,
						Data: api.FlowStartedEvent{
							FlowID: id,
							Plan:   pl,
							Init:   api.Args{},
						},
					},
					helpers.FlowEvent{
						Type: api.EventTypeStepStarted,
						Data: api.StepStartedEvent{
							FlowID: id,
							StepID: st.ID,
							Inputs: api.Args{},
							WorkItems: map[api.Token]api.Args{
								tkn: {},
							},
						},
					},
					helpers.FlowEvent{
						Type: api.EventTypeRetryScheduled,
						Data: api.RetryScheduledEvent{
							FlowID:      id,
							StepID:      st.ID,
							Token:       tkn,
							RetryCount:  1,
							NextRetryAt: time.Now(),
							Error:       "retry",
						},
					},
				))
			},
		)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkSucceeded, work.Status)
	})
}
