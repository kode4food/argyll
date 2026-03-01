package engine_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRecoveryActivation(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		flowID := api.FlowID("test-flow")

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitFor(wait.FlowStarted(flowID), func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.NotNil(t, flow)
		assert.Equal(t, flowID, flow.ID)
		assert.False(t, flow.CreatedAt.IsZero())
	})
}

func TestRecoveryDeactivation(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		flowID := api.FlowID("test-flow")

		step := helpers.NewSimpleStep("step-1")

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitFor(wait.FlowDeactivated(flowID), func() {
			err = env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		part, err := env.Engine.GetPartitionState()
		assert.NoError(t, err)
		_, ok := part.Active[flowID]
		assert.False(t, ok)
	})
}

func TestRecoverActiveFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		flowID1 := api.FlowID("flow-1")
		flowID2 := api.FlowID("flow-2")

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitForCount(2, wait.FlowStarted(
			flowID1, flowID2,
		), func() {
			err := env.Engine.StartFlow(flowID1, plan)
			assert.NoError(t, err)

			err = env.Engine.StartFlow(flowID2, plan)
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

		step := helpers.NewSimpleStep("retry-active")
		step.Type = api.StepTypeAsync

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-recover-active")
		env.WaitFor(wait.WorkStarted(api.FlowStep{
			FlowID: flowID,
			StepID: step.ID,
		}), func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		env.WaitFor(wait.WorkStarted(api.FlowStep{
			FlowID: flowID,
			StepID: step.ID,
		}), func() {
			assert.NoError(t, env.Engine.RecoverFlow(flowID))
		})
	})
}

func TestRecoverFlowRejectsPartialParentMetadata(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := helpers.NewSimpleStep("step-partial-parent-meta")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}
		flowID := api.FlowID("wf-recover-partial-parent")

		assert.NoError(t, env.RaiseFlowEvents(flowID, helpers.FlowEvent{
			Type: api.EventTypeFlowStarted,
			Data: api.FlowStartedEvent{
				FlowID: flowID,
				Plan:   plan,
				Metadata: api.Metadata{
					api.MetaParentFlowID: "parent",
				},
			},
		}))

		err := env.Engine.RecoverFlow(flowID)
		assert.ErrorContains(t, err, "partial parent metadata")
	})
}

func TestConcurrentRecoveryState(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		count := 10

		done := make(chan bool, count)

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		flowIDs := make([]api.FlowID, 0, count)
		for i := range count {
			flowIDs = append(flowIDs, api.FlowID(fmt.Sprintf("flow-%d", i)))
		}

		env.WaitForCount(
			len(flowIDs), wait.FlowStarted(flowIDs...),
			func() {
				for i := range count {
					go func(id int) {
						flowID := api.FlowID(fmt.Sprintf("flow-%d", id))
						err := env.Engine.StartFlow(flowID, plan)
						assert.NoError(t, err)
						done <- true
					}(i)
				}

				for range count {
					<-done
				}
			})

		for _, flowID := range flowIDs {
			flow, err := env.Engine.GetFlowState(flowID)
			assert.NoError(t, err)
			assert.NotNil(t, flow)
		}
	})
}

func TestTerminalFlow(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		flowID := api.FlowID("terminal-flow")

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		err := eng.StartFlow(flowID, plan)
		assert.NoError(t, err)

		flow, err := eng.GetFlowState(flowID)
		assert.NoError(t, err)
		flow.Status = api.FlowCompleted

		err = eng.RecoverFlow(flowID)
		assert.NoError(t, err)
	})
}

func TestNoRetryableSteps(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		flowID := api.FlowID("no-retry-flow")

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		err := eng.StartFlow(flowID, plan)
		assert.NoError(t, err)

		err = eng.RecoverFlow(flowID)
		assert.NoError(t, err)
	})
}

func TestWorkActiveItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("step-1")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 100,
			MaxBackoff:  1000,
			BackoffType: api.BackoffTypeFixed,
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("active-work-flow")
		env.WaitFor(wait.StepStarted(api.FlowStep{
			FlowID: flowID,
			StepID: step.ID,
		}), func() {
			err = env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlow(flowID)
		assert.NoError(t, err)

		_, err = env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
	})
}

func TestPendingWorkWithActiveStep(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("step-1")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 100,
			MaxBackoff:  1000,
			BackoffType: api.BackoffTypeFixed,
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("pending-active-flow")
		env.WaitFor(wait.StepStarted(api.FlowStep{
			FlowID: flowID,
			StepID: step.ID,
		}), func() {
			err = env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlow(flowID)
		assert.NoError(t, err)
	})
}

func TestFailedWorkRetryable(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("failing-step")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 100,
			MaxBackoff:  1000,
			BackoffType: api.BackoffTypeFixed,
		}

		env.MockClient.SetError("failing-step",
			fmt.Errorf("%w: test error", api.ErrWorkNotCompleted))

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"failing-step"},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("failed-work-flow")
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
			FlowID: flowID,
			StepID: "failing-step",
		}), func() {
			err = env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlow(flowID)
		assert.NoError(t, err)
	})
}

func TestInvalidFlowID(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		flowID := api.FlowID("nonexistent-flow")

		err := eng.RecoverFlow(flowID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get flow state")
	})
}

func TestMultipleFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		var err error
		flowID1 := api.FlowID("flow-1")
		flowID2 := api.FlowID("flow-2")
		env.WaitForCount(2, wait.FlowStarted(
			flowID1, flowID2,
		), func() {
			err = env.Engine.StartFlow(flowID1, plan)
			assert.NoError(t, err)

			err = env.Engine.StartFlow(flowID2, plan)
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
		flowID := api.FlowID("missing-step-flow")

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		err := eng.StartFlow(flowID, plan)
		assert.NoError(t, err)

		err = eng.RecoverFlow(flowID)
		assert.NoError(t, err)
	})
}

func TestRecoverFlowsWithFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		var err error
		flowID1 := api.FlowID("good-flow")
		flowID2 := api.FlowID("bad-flow")
		env.WaitForCount(2, wait.FlowStarted(
			flowID1, flowID2,
		), func() {
			err = env.Engine.StartFlow(flowID1, plan)
			assert.NoError(t, err)

			err = env.Engine.StartFlow(flowID2, plan)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlows()
		assert.NoError(t, err)
	})
}

func TestRecoverFlowNilWorkItems(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		flowID := api.FlowID("nil-work-flow")

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		var err error
		env.WaitFor(wait.FlowStarted(flowID), func() {
			err = env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlow(flowID)
		assert.NoError(t, err)
	})
}

func TestRecoverFlowsFromAggregateList(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
		flowID := api.FlowID("aggregate-recovery-flow")
		step := helpers.NewSimpleStep("aggregate-recovery-step")
		step.Type = api.StepTypeAsync
		token := api.Token("retry-token")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err := env.RaiseFlowEvents(
			flowID,
			helpers.FlowEvent{
				Type: api.EventTypeFlowStarted,
				Data: api.FlowStartedEvent{
					FlowID: flowID,
					Plan:   plan,
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
						token: {},
					},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeRetryScheduled,
				Data: api.RetryScheduledEvent{
					FlowID:      flowID,
					StepID:      step.ID,
					Token:       token,
					RetryCount:  1,
					NextRetryAt: now.Add(-time.Second),
					Error:       "retry",
				},
			},
		)
		assert.NoError(t, err)

		part, err := env.Engine.GetPartitionState()
		assert.NoError(t, err)
		_, ok := part.Active[flowID]
		assert.False(t, ok)

		env.MockClient.SetResponse(step.ID, api.Args{})
		env.WaitFor(wait.FlowActivated(flowID), func() {
			assert.NoError(t, env.Engine.Start())
		})

		invoked := env.MockClient.WaitForInvocation(step.ID, 2*time.Second)
		assert.True(t, invoked)

		part, err = env.Engine.GetPartitionState()
		assert.NoError(t, err)
		_, ok = part.Active[flowID]
		assert.True(t, ok)
	})
}

func TestRecoverFlowMixedStatuses(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		flowID := api.FlowID("recover-mixed-statuses")
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
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{stepA.ID},
			Steps: api.Steps{
				stepA.ID: stepA,
				stepB.ID: stepB,
			},
		}
		now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
		err := env.RaiseFlowEvents(
			flowID,
			helpers.FlowEvent{
				Type: api.EventTypeFlowStarted,
				Data: api.FlowStartedEvent{
					FlowID: flowID,
					Plan:   plan,
					Init:   api.Args{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeStepStarted,
				Data: api.StepStartedEvent{
					FlowID: flowID,
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
					FlowID: flowID,
					StepID: stepA.ID,
					Token:  tokenActive,
					Inputs: api.Args{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeWorkNotCompleted,
				Data: api.WorkNotCompletedEvent{
					FlowID: flowID,
					StepID: stepA.ID,
					Token:  tokenNotCompleted,
					Error:  "retry",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeRetryScheduled,
				Data: api.RetryScheduledEvent{
					FlowID:      flowID,
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
					FlowID:      flowID,
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
					FlowID: flowID,
					StepID: stepA.ID,
					Token:  tokenFailedRetry,
					Error:  "failed",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeWorkFailed,
				Data: api.WorkFailedEvent{
					FlowID: flowID,
					StepID: stepA.ID,
					Token:  tokenFailedNoRetry,
					Error:  "failed",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeWorkSucceeded,
				Data: api.WorkSucceededEvent{
					FlowID:  flowID,
					StepID:  stepA.ID,
					Token:   tokenSucceeded,
					Outputs: api.Args{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeStepStarted,
				Data: api.StepStartedEvent{
					FlowID: flowID,
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
					FlowID:      flowID,
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
					FlowID: flowID,
					StepID: stepB.ID,
					Error:  "failed step",
				},
			},
		)
		assert.NoError(t, err)

		err = env.Engine.RecoverFlow(flowID)
		assert.NoError(t, err)
	})
}

func TestRecoverFlowsPrunesDeactivatedAndArchiving(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

		activeFlowID := api.FlowID("recover-active")
		deactivatedFlowID := api.FlowID("recover-deactivated")
		archivingFlowID := api.FlowID("recover-archiving")
		active := helpers.NewSimpleStep("recover-step-active")
		deactivated := helpers.NewSimpleStep("recover-step-deactivated")
		archiving := helpers.NewSimpleStep("recover-step-archiving")
		activeToken := api.Token("active-token")
		deactivatedToken := api.Token("deactivated-token")
		archivingToken := api.Token("archiving-token")

		raiseFlow := func(
			flowID api.FlowID, step *api.Step, tkn api.Token,
		) {
			plan := &api.ExecutionPlan{
				Goals: []api.StepID{step.ID},
				Steps: api.Steps{step.ID: step},
			}
			err := env.RaiseFlowEvents(
				flowID,
				helpers.FlowEvent{
					Type: api.EventTypeFlowStarted,
					Data: api.FlowStartedEvent{
						FlowID: flowID,
						Plan:   plan,
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
		raiseFlow(archivingFlowID, archiving, archivingToken)

		env.WaitFor(wait.FlowActivated(activeFlowID), func() {
			env.Engine.EnqueueEvent(api.EventTypeFlowActivated,
				api.FlowActivatedEvent{FlowID: activeFlowID})
		})
		env.WaitFor(wait.FlowDeactivated(deactivatedFlowID), func() {
			env.Engine.EnqueueEvent(api.EventTypeFlowDeactivated,
				api.FlowDeactivatedEvent{FlowID: deactivatedFlowID})
		})
		env.WaitFor(wait.And(
			wait.EngineEvent(api.EventTypeFlowArchiving),
			wait.FlowID(archivingFlowID),
		), func() {
			env.Engine.EnqueueEvent(api.EventTypeFlowArchiving,
				api.FlowArchivingEvent{FlowID: archivingFlowID})
		})

		env.MockClient.SetResponse(active.ID, api.Args{})
		env.MockClient.SetResponse(deactivated.ID, api.Args{})
		env.MockClient.SetResponse(archiving.ID, api.Args{})
		assert.NoError(t, env.Engine.Stop())

		restarted, err := env.NewEngineInstance()
		assert.NoError(t, err)
		assert.NoError(t, restarted.Start())
		defer func() { _ = restarted.Stop() }()

		assert.True(t,
			env.MockClient.WaitForInvocation(active.ID, 2*time.Second))
		assert.False(t,
			env.MockClient.WaitForInvocation(deactivated.ID, 300*time.Millisecond))
		assert.False(t,
			env.MockClient.WaitForInvocation(archiving.ID, 300*time.Millisecond))

		activeFlow, err := restarted.GetFlowState(activeFlowID)
		assert.NoError(t, err)
		deactivatedFlow, err := restarted.GetFlowState(deactivatedFlowID)
		assert.NoError(t, err)
		archivingFlow, err := restarted.GetFlowState(archivingFlowID)
		assert.NoError(t, err)

		assert.NotEqual(t, api.WorkPending,
			activeFlow.Executions[active.ID].WorkItems[activeToken].Status)
		assert.Equal(t, api.WorkPending,
			deactivatedFlow.Executions[deactivated.ID].
				WorkItems[deactivatedToken].Status)
		assert.Equal(t, api.WorkPending,
			archivingFlow.Executions[archiving.ID].
				WorkItems[archivingToken].Status)
	})
}
