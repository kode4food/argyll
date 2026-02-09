package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const recoveryTimeout = 10 * time.Second

// TestBasicFlowRecovery tests that a single flow with pending work recovers
// and completes after engine crash (new engine instance)
func TestBasicFlowRecovery(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		consumer := env.EventHub.NewConsumer()
		defer consumer.Close()

		step := helpers.NewSimpleStep("recovery-step")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries: 20,
			Backoff:    200,
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
		err = env.Engine.StartFlow(flowID, plan)
		assert.NoError(t, err)

		helpers.WaitForFlowActivated(t, consumer, flowTimeout, flowID)

		// Wait for step to start via event hub
		env.WaitForStepStarted(t, flowID, step.ID, flowTimeout)

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
		env.Engine = env.NewEngineInstance()

		// Re-register step on new engine instance
		err = env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.Engine.Start()

		// Verify flow recovers and completes
		recovered := env.WaitForFlowStatus(t, flowID, recoveryTimeout)
		assert.Equal(t, api.FlowCompleted, recovered.Status)
		assert.Equal(t, api.StepCompleted, recovered.Executions[step.ID].Status)
	})
}

// TestMultipleFlowRecovery tests that multiple flows all recover after engine
// restart
func TestMultipleFlowRecovery(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		consumer := env.EventHub.NewConsumer()
		defer consumer.Close()

		step1 := helpers.NewSimpleStep("step-1")
		step1.WorkConfig = &api.WorkConfig{MaxRetries: 20, Backoff: 200}

		step2 := helpers.NewSimpleStep("step-2")
		step2.WorkConfig = &api.WorkConfig{MaxRetries: 20, Backoff: 200}

		step3 := helpers.NewSimpleStep("step-3")
		step3.WorkConfig = &api.WorkConfig{MaxRetries: 20, Backoff: 200}

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

		assert.NoError(t, env.Engine.StartFlow(flowID1, plan1))
		assert.NoError(t, env.Engine.StartFlow(flowID2, plan2))
		assert.NoError(t, env.Engine.StartFlow(flowID3, plan3))

		helpers.WaitForFlowActivated(t,
			consumer, flowTimeout, flowID1, flowID2, flowID3,
		)

		// Wait for all steps to start via event hub
		env.WaitForStepStarted(t, flowID1, step1.ID, flowTimeout)
		env.WaitForStepStarted(t, flowID2, step2.ID, flowTimeout)
		env.WaitForStepStarted(t, flowID3, step3.ID, flowTimeout)

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

		env.Engine = env.NewEngineInstance()

		// Re-register steps on new engine instance
		assert.NoError(t, env.Engine.RegisterStep(step1))
		assert.NoError(t, env.Engine.RegisterStep(step2))
		assert.NoError(t, env.Engine.RegisterStep(step3))

		flowConsumer := env.EventHub.NewConsumer()
		defer flowConsumer.Close()
		env.Engine.Start()
		helpers.WaitForFlowTerminal(t,
			flowConsumer, recoveryTimeout, flowID1, flowID2, flowID3,
		)

		recovered1, err := env.Engine.GetFlowState(flowID1)
		assert.NoError(t, err)
		recovered2, err := env.Engine.GetFlowState(flowID2)
		assert.NoError(t, err)
		recovered3, err := env.Engine.GetFlowState(flowID3)
		assert.NoError(t, err)

		assert.Equal(t, api.FlowCompleted, recovered1.Status)
		assert.Equal(t, api.FlowCompleted, recovered2.Status)
		assert.Equal(t, api.FlowCompleted, recovered3.Status)
	})
}

// TestRecoveryWorkStates tests recovery of flows with different work item
// states: Pending, NotCompleted, and Failed
func TestRecoveryWorkStates(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		consumer := env.EventHub.NewConsumer()
		defer consumer.Close()

		// Step 1: Will have Pending work (hasn't started yet)
		pendingStep := helpers.NewSimpleStep("pending-step")
		pendingStep.WorkConfig = &api.WorkConfig{MaxRetries: 20, Backoff: 200}

		// Step 2: Will have NotCompleted work (failed but retryable)
		notCompletedStep := helpers.NewSimpleStep("not-completed-step")
		notCompletedStep.WorkConfig = &api.WorkConfig{
			MaxRetries: 20, Backoff: 200,
		}

		// Step 3: Will have Failed work (perm failure, max retries reached)
		failedStep := helpers.NewSimpleStep("failed-step")
		failedStep.WorkConfig = &api.WorkConfig{MaxRetries: 0} // No retries

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

		// Start not-completed flow first (it will enter retry state)
		assert.NoError(t, env.Engine.StartFlow(notCompletedFlowID, plan2))

		// Start failed flow (will fail immediately)
		assert.NoError(t, env.Engine.StartFlow(failedFlowID, plan3))

		// Wait for notCompleted to start via event hub
		env.WaitForStepStarted(t,
			notCompletedFlowID, notCompletedStep.ID, flowTimeout,
		)

		// Wait for failed flow to fail
		env.WaitForFlowStatus(t, failedFlowID, flowTimeout)

		// Start pending flow just before shutdown
		assert.NoError(t, env.Engine.StartFlow(pendingFlowID, plan1))

		helpers.WaitForFlowActivated(t,
			consumer, flowTimeout, notCompletedFlowID, pendingFlowID,
		)

		err := env.Engine.Stop()
		assert.NoError(t, err)

		env.MockClient.ClearError(notCompletedStep.ID)
		env.MockClient.SetResponse(notCompletedStep.ID, api.Args{})

		env.Engine = env.NewEngineInstance()

		// Re-register steps on new engine instance
		assert.NoError(t, env.Engine.RegisterStep(pendingStep))
		assert.NoError(t, env.Engine.RegisterStep(notCompletedStep))
		assert.NoError(t, env.Engine.RegisterStep(failedStep))

		env.Engine.Start()

		// Verify recovery behavior:

		// 1. Pending flow should complete (was never started, now executes)
		pendingFlow := env.WaitForFlowStatus(t, pendingFlowID, recoveryTimeout)
		assert.Equal(t, api.FlowCompleted, pendingFlow.Status)

		// 2. NotCompleted flow should complete (recover & retry success)
		notCompletedFlow := env.WaitForFlowStatus(t,
			notCompletedFlowID, recoveryTimeout,
		)
		assert.Equal(t, api.FlowCompleted, notCompletedFlow.Status)

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
		env.Engine.Start()
		consumer := env.EventHub.NewConsumer()
		defer consumer.Close()

		step := helpers.NewSimpleStep("retry-step")
		step.WorkConfig = &api.WorkConfig{MaxRetries: 20, Backoff: 200}

		assert.NoError(t, env.Engine.RegisterStep(step))

		// Step fails initially
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("state-preservation-flow")

		err := env.Engine.StartFlow(flowID, plan)
		assert.NoError(t, err)

		helpers.WaitForFlowActivated(t, consumer, flowTimeout, flowID)

		// Wait for step to start via event hub
		env.WaitForStepStarted(t, flowID, step.ID, flowTimeout)

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

		env.Engine = env.NewEngineInstance()

		// Re-register step on new engine instance
		assert.NoError(t, env.Engine.RegisterStep(step))

		env.Engine.Start()

		// Wait for completion
		afterRestart := env.WaitForFlowStatus(t, flowID, recoveryTimeout)

		// Verify flow completed
		assert.Equal(t, api.FlowCompleted, afterRestart.Status)

		// Verify step completed after recovery
		assert.Equal(t,
			api.StepCompleted, afterRestart.Executions[step.ID].Status,
		)
	})
}
