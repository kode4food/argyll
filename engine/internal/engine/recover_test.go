package engine_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kode4food/timebox/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type earlyDelayedTimer struct {
	scheduler.Timer
	firedEarly bool
}

func TestRecoveryActivation(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		id := api.FlowID("test-flow")

		st := helpers.NewSimpleStep("step-1")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.FlowStarted(id), func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.NotNil(t, fl)
		assert.Equal(t, id, fl.ID)
		assert.False(t, fl.CreatedAt.IsZero())
	})
}

func TestRecoveryDeactivation(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		id := api.FlowID("test-flow")

		st := helpers.NewSimpleStep("step-1")

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.FlowDeactivated(id), func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.False(t, fl.DeactivatedAt.IsZero())
	})
}

func TestRecoverActiveFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		flowID1 := api.FlowID("flow-1")
		flowID2 := api.FlowID("flow-2")

		st := helpers.NewSimpleStep("step-1")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForCount(2, wait.FlowStarted(
			flowID1, flowID2,
		), func() {
			err := env.Engine.StartFlow(flowID1, pl)
			assert.NoError(t, err)

			err = env.Engine.StartFlow(flowID2, pl)
			assert.NoError(t, err)
		})

		flow1, err := env.Engine.GetFlowState(flowID1)
		assert.NoError(t, err)
		assert.NotNil(t, flow1)
		flow2, err := env.Engine.GetFlowState(flowID2)
		assert.NoError(t, err)
		assert.NotNil(t, flow2)
	})
}

func TestRecoverActiveWorkStartsRetry(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("retry-active")
		st.Type = api.StepTypeAsync

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-recover-active")
		env.WaitFor(wait.WorkStarted(api.FlowStep{
			FlowID: id,
			StepID: st.ID,
		}), func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		env.WaitFor(wait.WorkStarted(api.FlowStep{
			FlowID: id,
			StepID: st.ID,
		}), func() {
			assert.NoError(t, env.Engine.RecoverFlow(id))
		})
	})
}

func TestRecoverDispatchPeer(t *testing.T) {
	helpers.WithTestEnvDeps(t, engine.Dependencies{
		TimerConstructor: newEarlyDelayedTimer,
	}, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("dispatch-recovery-peer")
		env.MockClient.SetResponse(st.ID, api.Args{})
		require.NoError(t, env.Engine.RegisterStep(st))

		cfg := *env.Config
		cfg.Raft.LocalID = "node-2"
		cfg.Raft.Servers = []raft.Server{
			{ID: "node-2", Address: "127.0.0.1:9702"},
		}

		deps := env.Dependencies()
		deps.EventHub = event.NewHub()

		peer, unsubscribe, err := env.NewEngineWithConfig(&cfg, deps)
		require.NoError(t, err)
		defer func() {
			unsubscribe()
			_ = peer.Stop()
		}()

		require.NoError(t,
			env.Engine.UpdateStepHealth(
				st.ID, api.HealthUnhealthy, "starter offline",
			),
		)
		require.NoError(t, peer.UpdateStepHealth(st.ID, api.HealthHealthy, ""))
		require.NoError(t, env.Engine.Start())

		id := api.FlowID("dispatch-recovery-peer-flow")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		require.NoError(t, env.Engine.StartFlow(id, pl))
		require.NoError(t, peer.Start())

		fl := helpers.WaitForFlowState(
			t, env.Engine, id, time.Second,
			func(fl api.FlowState) bool {
				ex, ok := fl.Executions[st.ID]
				if !ok {
					return false
				}

				for _, work := range ex.WorkItems {
					return work.Status == api.WorkSucceeded &&
						fl.Status == api.FlowCompleted
				}
				return false
			})

		ex := fl.Executions[st.ID]
		assert.NotEmpty(t, ex.WorkItems)
		for _, work := range ex.WorkItems {
			assert.Equal(t, api.WorkSucceeded, work.Status)
		}
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestRecoverRejectsPartialParent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("step-partial-parent-meta")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}
		id := api.FlowID("wf-recover-partial-parent")

		assert.NoError(t, env.RaiseFlowEvents(id, helpers.FlowEvent{
			Type: api.EventTypeFlowStarted,
			Data: api.FlowStartedEvent{
				FlowID: id,
				Plan:   pl,
				Metadata: api.Metadata{
					api.MetaParentFlowID: "parent",
				},
			},
		}))

		err := env.Engine.RecoverFlow(id)
		assert.ErrorContains(t, err, "partial parent metadata")
	})
}

func TestConcurrentRecoveryState(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		count := 10

		done := make(chan bool, count)

		st := helpers.NewSimpleStep("step-1")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		flowIDs := make([]api.FlowID, 0, count)
		for i := range count {
			flowIDs = append(flowIDs, api.FlowID(fmt.Sprintf("flow-%d", i)))
		}

		env.WaitForCount(
			len(flowIDs), wait.FlowStarted(flowIDs...),
			func() {
				for i := range count {
					go func(idx int) {
						id := api.FlowID(fmt.Sprintf("flow-%d", idx))
						err := env.Engine.StartFlow(id, pl)
						assert.NoError(t, err)
						done <- true
					}(i)
				}

				for range count {
					<-done
				}
			})

		for _, id := range flowIDs {
			fl, err := env.Engine.GetFlowState(id)
			assert.NoError(t, err)
			assert.NotNil(t, fl)
		}
	})
}

func TestTerminalFlow(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		id := api.FlowID("terminal-flow")

		st := helpers.NewSimpleStep("step-1")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		err := eng.StartFlow(id, pl)
		assert.NoError(t, err)

		fl, err := eng.GetFlowState(id)
		assert.NoError(t, err)
		fl.Status = api.FlowCompleted

		err = eng.RecoverFlow(id)
		assert.NoError(t, err)
	})
}

func TestNoRetryableSteps(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		id := api.FlowID("no-retry-flow")

		st := helpers.NewSimpleStep("step-1")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		err := eng.StartFlow(id, pl)
		assert.NoError(t, err)

		err = eng.RecoverFlow(id)
		assert.NoError(t, err)
	})
}

func TestWorkActiveItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("step-1")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 100,
			MaxBackoff:  1000,
			BackoffType: api.BackoffTypeFixed,
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("active-work-flow")
		env.WaitFor(wait.StepStarted(api.FlowStep{
			FlowID: id,
			StepID: st.ID,
		}), func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlow(id)
		assert.NoError(t, err)

		_, err = env.Engine.GetFlowState(id)
		assert.NoError(t, err)
	})
}

func TestPendingWorkWithActiveStep(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("step-1")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 100,
			MaxBackoff:  1000,
			BackoffType: api.BackoffTypeFixed,
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("pending-active-flow")
		env.WaitFor(wait.StepStarted(api.FlowStep{
			FlowID: id,
			StepID: st.ID,
		}), func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlow(id)
		assert.NoError(t, err)
	})
}

func TestFailedWorkRetryable(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("failing-step")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 100,
			MaxBackoff:  1000,
			BackoffType: api.BackoffTypeFixed,
		}

		env.MockClient.SetError("failing-step",
			fmt.Errorf("%w: test error", api.ErrWorkNotCompleted))

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"failing-step"},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("failed-work-flow")
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
			FlowID: id,
			StepID: "failing-step",
		}), func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlow(id)
		assert.NoError(t, err)
	})
}

func TestInvalidFlowID(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		id := api.FlowID("nonexistent-flow")

		err := eng.RecoverFlow(id)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get flow state")
	})
}

func TestMultipleFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("step-1")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		var err error
		flowID1 := api.FlowID("flow-1")
		flowID2 := api.FlowID("flow-2")
		env.WaitForCount(2, wait.FlowStarted(
			flowID1, flowID2,
		), func() {
			err = env.Engine.StartFlow(flowID1, pl)
			assert.NoError(t, err)

			err = env.Engine.StartFlow(flowID2, pl)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlows()
		assert.NoError(t, err)
	})
}

func TestNoActiveFlows(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		err := eng.RecoverFlows()
		assert.NoError(t, err)
	})
}

func TestMissingStepInPlan(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		id := api.FlowID("missing-step-flow")

		st := helpers.NewSimpleStep("step-1")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		err := eng.StartFlow(id, pl)
		assert.NoError(t, err)

		err = eng.RecoverFlow(id)
		assert.NoError(t, err)
	})
}

func TestRecoverFlowsWithFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("step-1")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		var err error
		flowID1 := api.FlowID("good-flow")
		flowID2 := api.FlowID("bad-flow")
		env.WaitForCount(2, wait.FlowStarted(
			flowID1, flowID2,
		), func() {
			err = env.Engine.StartFlow(flowID1, pl)
			assert.NoError(t, err)

			err = env.Engine.StartFlow(flowID2, pl)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlows()
		assert.NoError(t, err)
	})
}

func TestRecoverFlowNilWorkItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		id := api.FlowID("nil-work-flow")

		st := helpers.NewSimpleStep("step-1")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{st.ID: st},
		}

		var err error
		env.WaitFor(wait.FlowStarted(id), func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlow(id)
		assert.NoError(t, err)
	})
}

func TestRecoverFlowsFromIndex(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
		id := api.FlowID("aggregate-recovery-flow")
		st := helpers.NewSimpleStep("aggregate-recovery-step")
		st.Type = api.StepTypeAsync
		tkn := api.Token("retry-token")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		err := env.RaiseFlowEvents(
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
					NextRetryAt: now.Add(-time.Second),
					Error:       "retry",
				},
			},
		)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})
		assert.NoError(t, env.Engine.Start())

		invoked := env.MockClient.WaitForInvocation(st.ID, 2*time.Second)
		assert.True(t, invoked)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.True(t, fl.DeactivatedAt.IsZero())
	})
}

func TestRecoverEarlyRetry(t *testing.T) {
	helpers.WithTestEnvDeps(t, engine.Dependencies{
		TimerConstructor: newEarlyDelayedTimer,
	}, func(env *helpers.TestEngineEnv) {
		id := api.FlowID("recover-early-retry")
		st := helpers.NewSimpleStep("recover-early-retry-step")
		tkn := api.Token("retry-token")
		nextRetryAt := time.Now().UTC().Add(250 * time.Millisecond)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		err := env.RaiseFlowEvents(
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
					NextRetryAt: nextRetryAt,
					Error:       "retry",
				},
			},
		)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		assert.NoError(t, env.Engine.Start())

		invoked := env.MockClient.WaitForInvocation(st.ID, 2*time.Second)
		assert.True(t, invoked)

		fl := env.WaitForFlowStatus(id, func() {})
		assert.Equal(t, api.FlowCompleted, fl.Status)
	})
}

func TestRecoverFlowMixedStatuses(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		id := api.FlowID("recover-mixed-statuses")
		stepA := helpers.NewSimpleStep("mixed-step-a")
		stepB := helpers.NewSimpleStep("mixed-step-b")
		tokenActive := api.Token("active")
		tokenNotCompleted := api.Token("not-completed")
		tokenPendingRetry := api.Token("pending-retry")
		tokenPendingNoRetry := api.Token("pending-no-retry")
		tokenFailedRetry := api.Token("failed-retry")
		tokenFailedNoRetry := api.Token("failed-no-retry")
		tokenSucceeded := api.Token("succeeded")
		tokenBranchReady := api.Token("branch-ready")
		tokenBranchSkip := api.Token("branch-skip")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{stepA.ID},
			Steps: api.Steps{
				stepA.ID: stepA,
				stepB.ID: stepB,
			},
		}
		now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
		err := env.RaiseFlowEvents(
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
					StepID: stepA.ID,
					Inputs: api.Args{},
					WorkItems: map[api.Token]api.Args{
						tokenActive:         {},
						tokenNotCompleted:   {},
						tokenPendingRetry:   {},
						tokenPendingNoRetry: {},
						tokenFailedRetry:    {},
						tokenFailedNoRetry:  {},
						tokenSucceeded:      {},
					},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeWorkStarted,
				Data: api.WorkStartedEvent{
					FlowID: id,
					StepID: stepA.ID,
					Token:  tokenActive,
					Inputs: api.Args{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeWorkNotCompleted,
				Data: api.WorkNotCompletedEvent{
					FlowID: id,
					StepID: stepA.ID,
					Token:  tokenNotCompleted,
					Error:  "retry",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeRetryScheduled,
				Data: api.RetryScheduledEvent{
					FlowID:      id,
					StepID:      stepA.ID,
					Token:       tokenPendingRetry,
					RetryCount:  1,
					NextRetryAt: now.Add(-500 * time.Millisecond),
					Error:       "retry",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeRetryScheduled,
				Data: api.RetryScheduledEvent{
					FlowID:      id,
					StepID:      stepA.ID,
					Token:       tokenFailedRetry,
					RetryCount:  2,
					NextRetryAt: now.Add(-500 * time.Millisecond),
					Error:       "retry",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeWorkFailed,
				Data: api.WorkFailedEvent{
					FlowID: id,
					StepID: stepA.ID,
					Token:  tokenFailedRetry,
					Error:  "failed",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeWorkFailed,
				Data: api.WorkFailedEvent{
					FlowID: id,
					StepID: stepA.ID,
					Token:  tokenFailedNoRetry,
					Error:  "failed",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeWorkSucceeded,
				Data: api.WorkSucceededEvent{
					FlowID:  id,
					StepID:  stepA.ID,
					Token:   tokenSucceeded,
					Outputs: api.Args{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeStepStarted,
				Data: api.StepStartedEvent{
					FlowID: id,
					StepID: stepB.ID,
					Inputs: api.Args{},
					WorkItems: map[api.Token]api.Args{
						tokenBranchReady: {},
						tokenBranchSkip:  {},
					},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeRetryScheduled,
				Data: api.RetryScheduledEvent{
					FlowID:      id,
					StepID:      stepB.ID,
					Token:       tokenBranchReady,
					RetryCount:  1,
					NextRetryAt: now.Add(-500 * time.Millisecond),
					Error:       "retry",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeStepFailed,
				Data: api.StepFailedEvent{
					FlowID: id,
					StepID: stepB.ID,
					Error:  "failed step",
				},
			},
		)
		assert.NoError(t, err)

		err = env.Engine.RecoverFlow(id)
		assert.NoError(t, err)
	})
}

func TestRecoverFlowsSkipsDeactivated(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

		activeFlowID := api.FlowID("recover-active")
		deactivatedFlowID := api.FlowID("recover-deactivated")
		active := helpers.NewSimpleStep("recover-step-active")
		deactivated := helpers.NewSimpleStep("recover-step-deactivated")
		activeToken := api.Token("active-token")
		deactivatedToken := api.Token("deactivated-token")

		raiseFlow := func(
			flowID api.FlowID, step *api.Step, tkn api.Token,
		) {
			pl := &api.ExecutionPlan{
				Goals: []api.StepID{step.ID},
				Steps: api.Steps{step.ID: step},
			}
			err := env.RaiseFlowEvents(
				flowID,
				helpers.FlowEvent{
					Type: api.EventTypeFlowStarted,
					Data: api.FlowStartedEvent{
						FlowID: flowID,
						Plan:   pl,
						Init:   api.Args{},
					},
				},
				helpers.FlowEvent{
					Type: api.EventTypeStepStarted,
					Data: api.StepStartedEvent{
						FlowID: flowID,
						StepID: step.ID,
						Inputs: api.Args{},
						WorkItems: map[api.Token]api.Args{
							tkn: {},
						},
					},
				},
				helpers.FlowEvent{
					Type: api.EventTypeRetryScheduled,
					Data: api.RetryScheduledEvent{
						FlowID:      flowID,
						StepID:      step.ID,
						Token:       tkn,
						RetryCount:  1,
						NextRetryAt: now.Add(-500 * time.Millisecond),
						Error:       "retry",
					},
				},
			)
			assert.NoError(t, err)
		}

		raiseFlow(activeFlowID, active, activeToken)
		raiseFlow(deactivatedFlowID, deactivated, deactivatedToken)
		assert.NoError(t, env.RaiseFlowEvents(
			deactivatedFlowID,
			helpers.FlowEvent{
				Type: api.EventTypeFlowDeactivated,
				Data: api.FlowDeactivatedEvent{
					FlowID: deactivatedFlowID,
					Status: api.FlowCompleted,
				},
			},
		))

		env.MockClient.SetResponse(active.ID, api.Args{})
		env.MockClient.SetResponse(deactivated.ID, api.Args{})
		assert.NoError(t, env.Engine.Stop())

		restarted, err := env.NewEngineInstance()
		assert.NoError(t, err)
		assert.NoError(t, restarted.Start())
		defer func() { _ = restarted.Stop() }()

		assert.True(t,
			env.MockClient.WaitForInvocation(active.ID, 2*time.Second))
		assert.False(t,
			env.MockClient.WaitForInvocation(deactivated.ID, 300*time.Millisecond))

		activeFlow, err := restarted.GetFlowState(activeFlowID)
		assert.NoError(t, err)
		deactivatedFlow, err := restarted.GetFlowState(deactivatedFlowID)
		assert.NoError(t, err)

		assert.NotEqual(t, api.WorkPending,
			activeFlow.Executions[active.ID].WorkItems[activeToken].Status)
		assert.Equal(t, api.WorkPending,
			deactivatedFlow.Executions[deactivated.ID].
				WorkItems[deactivatedToken].Status)
		assert.False(t, deactivatedFlow.DeactivatedAt.IsZero())
	})
}

func newEarlyDelayedTimer(delay time.Duration) scheduler.Timer {
	return &earlyDelayedTimer{
		Timer: scheduler.NewTimer(delay),
	}
}

func (t *earlyDelayedTimer) Reset(delay time.Duration) bool {
	if delay > 0 && !t.firedEarly {
		t.firedEarly = true
		return t.Timer.Reset(0)
	}
	return t.Timer.Reset(delay)
}
