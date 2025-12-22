package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const recoveryTimeout = 10 * time.Second

// TestBasicWorkflowRecovery tests that a single workflow with pending work
// recovers and completes after engine crash (new engine instance)
func TestBasicWorkflowRecovery(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep("recovery-step")
	step.WorkConfig = &api.WorkConfig{
		MaxRetries: 20,
		BackoffMs:  200,
	}

	err := env.Engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	// Make step fail initially (will retry)
	env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	flowID := api.FlowID("test-recovery")
	err = env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	// Wait for step to start via event hub
	env.WaitForStepStarted(t, ctx, flowID, step.ID, workflowTimeout)

	// Verify workflow is active with pending work
	flow, err := env.Engine.GetFlowState(ctx, flowID)
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
	err = env.Engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	env.Engine.Start()

	// Verify workflow recovers and completes
	recovered := env.WaitForFlowStatus(t, ctx, flowID, recoveryTimeout)
	assert.Equal(t, api.FlowCompleted, recovered.Status)
	assert.Equal(t, api.StepCompleted, recovered.Executions[step.ID].Status)
}

// TestMultipleWorkflowRecovery tests that multiple workflows all recover
// after engine restart
func TestMultipleWorkflowRecovery(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step1 := helpers.NewSimpleStep("step-1")
	step1.WorkConfig = &api.WorkConfig{MaxRetries: 20, BackoffMs: 200}

	step2 := helpers.NewSimpleStep("step-2")
	step2.WorkConfig = &api.WorkConfig{MaxRetries: 20, BackoffMs: 200}

	step3 := helpers.NewSimpleStep("step-3")
	step3.WorkConfig = &api.WorkConfig{MaxRetries: 20, BackoffMs: 200}

	assert.NoError(t, env.Engine.RegisterStep(ctx, step1))
	assert.NoError(t, env.Engine.RegisterStep(ctx, step2))
	assert.NoError(t, env.Engine.RegisterStep(ctx, step3))

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

	assert.NoError(t, env.Engine.StartFlow(ctx, flowID1, plan1, api.Args{}, api.Metadata{}))
	assert.NoError(t, env.Engine.StartFlow(ctx, flowID2, plan2, api.Args{}, api.Metadata{}))
	assert.NoError(t, env.Engine.StartFlow(ctx, flowID3, plan3, api.Args{}, api.Metadata{}))

	// Wait for all steps to start via event hub
	env.WaitForStepStarted(t, ctx, flowID1, step1.ID, workflowTimeout)
	env.WaitForStepStarted(t, ctx, flowID2, step2.ID, workflowTimeout)
	env.WaitForStepStarted(t, ctx, flowID3, step3.ID, workflowTimeout)

	// Verify all workflows are active with work in progress
	flow1, err := env.Engine.GetFlowState(ctx, flowID1)
	assert.NoError(t, err)
	flow2, err := env.Engine.GetFlowState(ctx, flowID2)
	assert.NoError(t, err)
	flow3, err := env.Engine.GetFlowState(ctx, flowID3)
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
	assert.NoError(t, env.Engine.RegisterStep(ctx, step1))
	assert.NoError(t, env.Engine.RegisterStep(ctx, step2))
	assert.NoError(t, env.Engine.RegisterStep(ctx, step3))

	// Subscribe BEFORE starting engine to avoid race condition
	waiter1 := env.SubscribeToFlowStatus(flowID1)
	waiter2 := env.SubscribeToFlowStatus(flowID2)
	waiter3 := env.SubscribeToFlowStatus(flowID3)

	env.Engine.Start()

	// Wait concurrently for all workflows to recover and complete
	var recovered1, recovered2, recovered3 *api.FlowState
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		recovered1 = waiter1.Wait(t, ctx, recoveryTimeout)
	}()
	go func() {
		defer wg.Done()
		recovered2 = waiter2.Wait(t, ctx, recoveryTimeout)
	}()
	go func() {
		defer wg.Done()
		recovered3 = waiter3.Wait(t, ctx, recoveryTimeout)
	}()

	wg.Wait()

	if recovered1 == nil {
		t.Fatal("flow-1 timed out during recovery")
	}
	if recovered2 == nil {
		t.Fatal("flow-2 timed out during recovery")
	}
	if recovered3 == nil {
		t.Fatal("flow-3 timed out during recovery")
	}

	assert.Equal(t, api.FlowCompleted, recovered1.Status)
	assert.Equal(t, api.FlowCompleted, recovered2.Status)
	assert.Equal(t, api.FlowCompleted, recovered3.Status)
}

// TestRecoveryWithDifferentWorkStates tests recovery of workflows with
// different work item states: Pending, NotCompleted, and Failed
func TestRecoveryWithDifferentWorkStates(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	// Step 1: Will have Pending work (hasn't started yet)
	pendingStep := helpers.NewSimpleStep("pending-step")
	pendingStep.WorkConfig = &api.WorkConfig{MaxRetries: 20, BackoffMs: 200}

	// Step 2: Will have NotCompleted work (failed but retryable)
	notCompletedStep := helpers.NewSimpleStep("not-completed-step")
	notCompletedStep.WorkConfig = &api.WorkConfig{
		MaxRetries: 20, BackoffMs: 200,
	}

	// Step 3: Will have Failed work (permanent failure, max retries reached)
	failedStep := helpers.NewSimpleStep("failed-step")
	failedStep.WorkConfig = &api.WorkConfig{MaxRetries: 0} // No retries

	assert.NoError(t, env.Engine.RegisterStep(ctx, pendingStep))
	assert.NoError(t, env.Engine.RegisterStep(ctx, notCompletedStep))
	assert.NoError(t, env.Engine.RegisterStep(ctx, failedStep))

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
	assert.NoError(t, env.Engine.StartFlow(
		ctx, notCompletedFlowID, plan2, api.Args{}, api.Metadata{},
	))

	// Start failed flow (will fail immediately)
	assert.NoError(t, env.Engine.StartFlow(
		ctx, failedFlowID, plan3, api.Args{}, api.Metadata{},
	))

	// Wait for notCompleted to start via event hub
	env.WaitForStepStarted(
		t, ctx, notCompletedFlowID, notCompletedStep.ID, workflowTimeout,
	)

	// Wait for failed flow to fail
	env.WaitForFlowStatus(t, ctx, failedFlowID, workflowTimeout)

	// Start pending flow just before shutdown
	assert.NoError(t, env.Engine.StartFlow(
		ctx, pendingFlowID, plan1, api.Args{}, api.Metadata{},
	))

	err := env.Engine.Stop()
	assert.NoError(t, err)

	env.MockClient.ClearError(notCompletedStep.ID)
	env.MockClient.SetResponse(notCompletedStep.ID, api.Args{})

	env.Engine = env.NewEngineInstance()

	// Re-register steps on new engine instance
	assert.NoError(t, env.Engine.RegisterStep(ctx, pendingStep))
	assert.NoError(t, env.Engine.RegisterStep(ctx, notCompletedStep))
	assert.NoError(t, env.Engine.RegisterStep(ctx, failedStep))

	env.Engine.Start()

	// Verify recovery behavior:

	// 1. Pending workflow should complete (was never started, now executes)
	pendingFlow := env.WaitForFlowStatus(t, ctx, pendingFlowID, recoveryTimeout)
	assert.Equal(t, api.FlowCompleted, pendingFlow.Status)

	// 2. NotCompleted workflow should complete (recover & retry successfully)
	notCompletedFlow := env.WaitForFlowStatus(
		t, ctx, notCompletedFlowID, recoveryTimeout,
	)
	assert.Equal(t, api.FlowCompleted, notCompletedFlow.Status)

	// 3. Failed workflow should still be failed (no retry for perm failures)
	failedFlow, err := env.Engine.GetFlowState(ctx, failedFlowID)
	assert.NoError(t, err)
	assert.Equal(t, api.FlowFailed, failedFlow.Status)
}

// TestRecoveryPreservesWorkflowState tests that workflow state is preserved
// across restarts (recovery picks up where it left off)
func TestRecoveryPreservesWorkflowState(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep("retry-step")
	step.WorkConfig = &api.WorkConfig{MaxRetries: 20, BackoffMs: 200}

	assert.NoError(t, env.Engine.RegisterStep(ctx, step))

	// Step fails initially
	env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	flowID := api.FlowID("state-preservation-flow")

	err := env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	// Wait for step to start via event hub
	env.WaitForStepStarted(t, ctx, flowID, step.ID, workflowTimeout)

	// Get state before restart
	beforeRestart, err := env.Engine.GetFlowState(ctx, flowID)
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
	assert.NoError(t, env.Engine.RegisterStep(ctx, step))

	env.Engine.Start()

	// Wait for completion
	afterRestart := env.WaitForFlowStatus(t, ctx, flowID, recoveryTimeout)

	// Verify workflow completed
	assert.Equal(t, api.FlowCompleted, afterRestart.Status)

	// Verify step completed after recovery
	assert.Equal(t, api.StepCompleted, afterRestart.Executions[step.ID].Status)
}
