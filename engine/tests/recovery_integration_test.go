package tests

import (
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const recoveryTimeout = 10 * time.Second

func waitForFlowStatusWithTimeoutAfter(
	env *helpers.TestEngineEnv, flowID api.FlowID, timeout time.Duration,
	fn func(),
) *api.FlowState {
	env.T.Helper()

	states := waitForFlowsStatusWithTimeoutAfter(
		env, []api.FlowID{flowID}, timeout, fn,
	)
	return states[flowID]
}

func waitForFlowsStatusWithTimeoutAfter(
	env *helpers.TestEngineEnv, ids []api.FlowID, timeout time.Duration,
	fn func(),
) map[api.FlowID]*api.FlowState {
	env.T.Helper()

	consumers := make([]*timebox.Consumer, len(ids))
	for i := range ids {
		consumer := env.EventHub.NewConsumer()
		consumers[i] = consumer
	}
	defer func() {
		for _, consumer := range consumers {
			consumer.Close()
		}
	}()

	fn()

	res := make(map[api.FlowID]*api.FlowState, len(ids))
	for i, flowID := range ids {
		state, err := env.Engine.GetFlowState(flowID)
		if err != nil {
			env.T.Fatalf("failed to fetch flow %s: %v", flowID, err)
		}
		if state.Status != api.FlowCompleted &&
			state.Status != api.FlowFailed {
			filter := wait.FlowTerminal(flowID)
			timer := time.NewTimer(timeout)
			found := false
			for !found {
				select {
				case ev, ok := <-consumers[i].Receive():
					if !ok {
						timer.Stop()
						env.T.Fatalf(
							"event consumer closed waiting for flow %s",
							flowID,
						)
					}
					if ev != nil && filter(ev) {
						found = true
					}
				case <-timer.C:
					found = true
				}
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			state, err = env.Engine.GetFlowState(flowID)
			if err != nil {
				env.T.Fatalf("failed to fetch flow %s: %v", flowID, err)
			}
		}
		if state.Status != api.FlowCompleted &&
			state.Status != api.FlowFailed {
			env.T.Fatalf(
				"flow %s not terminal after wait; status=%s",
				flowID,
				state.Status,
			)
		}
		res[flowID] = state
	}

	return res
}

// TestBasicFlowRecovery tests that a single flow with pending work recovers
// and completes after engine crash (new engine instance)
func TestBasicFlowRecovery(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("recovery-step")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  20,
			InitBackoff: 200,
			MaxBackoff:  200,
			BackoffType: api.BackoffTypeFixed,
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		// Make step fail initially (will retry)
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("test-recovery")
		env.WaitAfterAll(2, func(waits []*wait.Wait) {
			err = env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
			waits[0].ForEvent(wait.FlowActivated(flowID))
			waits[1].ForEvent(wait.StepStarted(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			}))
		})

		// Verify flow is active with pending work
		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowActive, flow.Status)

		// Stop engine (simulating crash)
		err = env.Engine.Stop()
		assert.NoError(t, err)

		// Change mock to succeed
		env.MockClient.ClearError(step.ID)
		env.MockClient.SetResponse(step.ID, api.Args{})

		// Create new engine instance (simulates process restart)
		env.Engine, err = env.NewEngineInstance()
		assert.NoError(t, err)

		// Re-register step on new engine instance
		err = env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		// Verify flow recovers and completes
		recovered := waitForFlowStatusWithTimeoutAfter(
			env, flowID, recoveryTimeout, func() {
				assert.NoError(t, env.Engine.Start())
			},
		)
		assert.Equal(t, api.FlowCompleted, recovered.Status)
		assert.Equal(t, api.StepCompleted, recovered.Executions[step.ID].Status)
	})
}

// TestMultipleFlowRecovery tests that multiple flows all recover after engine
// restart
func TestMultipleFlowRecovery(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step1 := helpers.NewSimpleStep("step-1")
		step1.WorkConfig = &api.WorkConfig{
			MaxRetries:  20,
			InitBackoff: 200,
			MaxBackoff:  200,
			BackoffType: api.BackoffTypeFixed,
		}

		step2 := helpers.NewSimpleStep("step-2")
		step2.WorkConfig = &api.WorkConfig{
			MaxRetries:  20,
			InitBackoff: 200,
			MaxBackoff:  200,
			BackoffType: api.BackoffTypeFixed,
		}

		step3 := helpers.NewSimpleStep("step-3")
		step3.WorkConfig = &api.WorkConfig{
			MaxRetries:  20,
			InitBackoff: 200,
			MaxBackoff:  200,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(step1))
		assert.NoError(t, env.Engine.RegisterStep(step2))
		assert.NoError(t, env.Engine.RegisterStep(step3))

		// All steps fail initially
		env.MockClient.SetError(step1.ID, api.ErrWorkNotCompleted)
		env.MockClient.SetError(step2.ID, api.ErrWorkNotCompleted)
		env.MockClient.SetError(step3.ID, api.ErrWorkNotCompleted)

		plan1 := &api.ExecutionPlan{
			Goals: []api.StepID{step1.ID},
			Steps: api.Steps{step1.ID: step1},
		}
		plan2 := &api.ExecutionPlan{
			Goals: []api.StepID{step2.ID},
			Steps: api.Steps{step2.ID: step2},
		}
		plan3 := &api.ExecutionPlan{
			Goals: []api.StepID{step3.ID},
			Steps: api.Steps{step3.ID: step3},
		}

		flowID1 := api.FlowID("flow-1")
		flowID2 := api.FlowID("flow-2")
		flowID3 := api.FlowID("flow-3")

		env.WaitAfterAll(4, func(waits []*wait.Wait) {
			assert.NoError(t, env.Engine.StartFlow(flowID1, plan1))
			assert.NoError(t, env.Engine.StartFlow(flowID2, plan2))
			assert.NoError(t, env.Engine.StartFlow(flowID3, plan3))
			waits[0].ForEvents(3, wait.FlowActivated(flowID1, flowID2, flowID3))
			waits[1].ForEvent(wait.StepStarted(api.FlowStep{
				FlowID: flowID1,
				StepID: step1.ID,
			}))
			waits[2].ForEvent(wait.StepStarted(api.FlowStep{
				FlowID: flowID2,
				StepID: step2.ID,
			}))
			waits[3].ForEvent(wait.StepStarted(api.FlowStep{
				FlowID: flowID3,
				StepID: step3.ID,
			}))
		})

		// Verify all flows are active with work in progress
		flow1, err := env.Engine.GetFlowState(flowID1)
		assert.NoError(t, err)
		flow2, err := env.Engine.GetFlowState(flowID2)
		assert.NoError(t, err)
		flow3, err := env.Engine.GetFlowState(flowID3)
		assert.NoError(t, err)

		assert.Equal(t, api.FlowActive, flow1.Status)
		assert.Equal(t, api.FlowActive, flow2.Status)
		assert.Equal(t, api.FlowActive, flow3.Status)

		err = env.Engine.Stop()
		assert.NoError(t, err)

		env.MockClient.ClearError(step1.ID)
		env.MockClient.ClearError(step2.ID)
		env.MockClient.ClearError(step3.ID)
		env.MockClient.SetResponse(step1.ID, api.Args{})
		env.MockClient.SetResponse(step2.ID, api.Args{})
		env.MockClient.SetResponse(step3.ID, api.Args{})

		env.Engine, err = env.NewEngineInstance()
		assert.NoError(t, err)

		// Re-register steps on new engine instance
		assert.NoError(t, env.Engine.RegisterStep(step1))
		assert.NoError(t, env.Engine.RegisterStep(step2))
		assert.NoError(t, env.Engine.RegisterStep(step3))

		recovered := waitForFlowsStatusWithTimeoutAfter(
			env,
			[]api.FlowID{flowID1, flowID2, flowID3},
			recoveryTimeout,
			func() {
				assert.NoError(t, env.Engine.Start())
			},
		)

		assert.Equal(t, api.FlowCompleted, recovered[flowID1].Status)
		assert.Equal(t, api.FlowCompleted, recovered[flowID2].Status)
		assert.Equal(t, api.FlowCompleted, recovered[flowID3].Status)
	})
}

// TestRecoveryWorkStates tests recovery of flows with different work item
// states: Pending, NotCompleted, and Failed
func TestRecoveryWorkStates(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Step 1: Will have Pending work (hasn't started yet)
		pendingStep := helpers.NewSimpleStep("pending-step")
		pendingStep.WorkConfig = &api.WorkConfig{
			MaxRetries:  20,
			InitBackoff: 200,
			MaxBackoff:  200,
			BackoffType: api.BackoffTypeFixed,
		}

		// Step 2: Will have NotCompleted work (failed but retryable)
		notCompletedStep := helpers.NewSimpleStep("not-completed-step")
		notCompletedStep.WorkConfig = &api.WorkConfig{
			MaxRetries:  20,
			InitBackoff: 200,
			MaxBackoff:  200,
			BackoffType: api.BackoffTypeFixed,
		}

		// Step 3: Will have Failed work (perm failure, max retries reached)
		failedStep := helpers.NewSimpleStep("failed-step")
		failedStep.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(pendingStep))
		assert.NoError(t, env.Engine.RegisterStep(notCompletedStep))
		assert.NoError(t, env.Engine.RegisterStep(failedStep))

		// Set up mock responses
		env.MockClient.SetResponse(pendingStep.ID, api.Args{})
		env.MockClient.SetError(notCompletedStep.ID, api.ErrWorkNotCompleted)
		env.MockClient.SetError(failedStep.ID, api.ErrWorkNotCompleted)

		plan1 := &api.ExecutionPlan{
			Goals: []api.StepID{pendingStep.ID},
			Steps: api.Steps{pendingStep.ID: pendingStep},
		}
		plan2 := &api.ExecutionPlan{
			Goals: []api.StepID{notCompletedStep.ID},
			Steps: api.Steps{notCompletedStep.ID: notCompletedStep},
		}
		plan3 := &api.ExecutionPlan{
			Goals: []api.StepID{failedStep.ID},
			Steps: api.Steps{failedStep.ID: failedStep},
		}

		pendingFlowID := api.FlowID("pending-flow")
		notCompletedFlowID := api.FlowID("not-completed-flow")
		failedFlowID := api.FlowID("failed-flow")

		env.WaitAfterAll(2, func(waits []*wait.Wait) {
			assert.NoError(t,
				env.Engine.StartFlow(notCompletedFlowID, plan2),
			)
			waits[0].ForEvent(wait.FlowActivated(notCompletedFlowID))
			waits[1].ForEvent(wait.StepStarted(api.FlowStep{
				FlowID: notCompletedFlowID,
				StepID: notCompletedStep.ID,
			}))
		})

		// Wait for failed flow to fail
		env.WaitForFlowStatus(failedFlowID, func() {
			assert.NoError(t, env.Engine.StartFlow(failedFlowID, plan3))
		})

		// Start pending flow just before shutdown
		env.WaitFor(wait.FlowActivated(pendingFlowID), func() {
			assert.NoError(t, env.Engine.StartFlow(pendingFlowID, plan1))
		})

		err := env.Engine.Stop()
		assert.NoError(t, err)

		env.MockClient.ClearError(notCompletedStep.ID)
		env.MockClient.SetResponse(notCompletedStep.ID, api.Args{})

		env.Engine, err = env.NewEngineInstance()
		assert.NoError(t, err)

		// Re-register steps on new engine instance
		assert.NoError(t, env.Engine.RegisterStep(pendingStep))
		assert.NoError(t, env.Engine.RegisterStep(notCompletedStep))
		assert.NoError(t, env.Engine.RegisterStep(failedStep))

		// Verify recovery behavior:

		// 1. Pending flow should complete (was never started, now executes)
		recovered := waitForFlowsStatusWithTimeoutAfter(
			env,
			[]api.FlowID{pendingFlowID, notCompletedFlowID},
			recoveryTimeout,
			func() {
				assert.NoError(t, env.Engine.Start())
			},
		)
		assert.Equal(t, api.FlowCompleted, recovered[pendingFlowID].Status)

		// 2. NotCompleted flow should complete (recover & retry success)
		assert.Equal(t, api.FlowCompleted, recovered[notCompletedFlowID].Status)

		// 3. Failed flow should still fail (no retry for perm failures)
		failedFlow, err := env.Engine.GetFlowState(failedFlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, failedFlow.Status)
	})
}

// TestRecoveryPreservesState tests that flow state is preserved across
// restarts (recovery picks up where it left off)
func TestRecoveryPreservesState(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("retry-step")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  20,
			InitBackoff: 200,
			MaxBackoff:  200,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))

		// Step fails initially
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("state-preservation-flow")
		env.WaitAfterAll(2, func(waits []*wait.Wait) {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
			waits[0].ForEvent(wait.FlowActivated(flowID))
			waits[1].ForEvent(wait.StepStarted(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			}))
		})

		// Get state before restart
		beforeRestart, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowActive, beforeRestart.Status)

		// Verify step is active (has been attempted)
		exec := beforeRestart.Executions[step.ID]
		assert.Equal(t, api.StepActive, exec.Status)
		assert.NotEmpty(t, exec.WorkItems)

		err = env.Engine.Stop()
		assert.NoError(t, err)

		env.MockClient.ClearError(step.ID)
		env.MockClient.SetResponse(step.ID, api.Args{})

		env.Engine, err = env.NewEngineInstance()
		assert.NoError(t, err)

		// Re-register step on new engine instance
		assert.NoError(t, env.Engine.RegisterStep(step))

		// Wait for completion
		afterRestart := waitForFlowStatusWithTimeoutAfter(
			env, flowID, recoveryTimeout, func() {
				assert.NoError(t, env.Engine.Start())
			},
		)

		// Verify flow completed
		assert.Equal(t, api.FlowCompleted, afterRestart.Status)

		// Verify step completed after recovery
		assert.Equal(t,
			api.StepCompleted, afterRestart.Executions[step.ID].Status,
		)
	})
}
