package engine_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestRecoveryActivation(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	ctx := context.Background()
	flowID := timebox.ID("test-workflow")

	step := helpers.NewSimpleStep("step-1")
	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-1"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err := env.Engine.StartWorkflow(
		ctx, flowID, plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	// Wait a bit for the event loop to process the workflow started event
	time.Sleep(100 * time.Millisecond)

	// Check that workflow was created and started
	workflow, err := env.Engine.GetWorkflowState(ctx, flowID)
	require.NoError(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, flowID, workflow.ID)
	assert.False(t, workflow.CreatedAt.IsZero())
}

func TestRecoveryDeactivation(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	ctx := context.Background()
	flowID := timebox.ID("test-workflow")

	step := &api.Step{ID: "step-1"}
	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-1"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err := env.Engine.StartWorkflow(
		ctx, flowID, plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.CompleteWorkflow(ctx, flowID, api.Args{})
	require.NoError(t, err)

	// Wait a bit for the event loop to process the workflow completed event
	time.Sleep(100 * time.Millisecond)

	engineState, err := env.Engine.GetEngineState(ctx)
	require.NoError(t, err)

	_, exists := engineState.ActiveWorkflows[flowID]
	assert.False(t, exists, "workflow should not be in active workflows")
}

func TestShouldRetryStep(t *testing.T) {
	scenarios := []struct {
		name     string
		config   *api.WorkConfig
		retries  int
		error    string
		expected bool
	}{
		{
			name:     "no config",
			config:   nil,
			retries:  0,
			error:    "network timeout",
			expected: true,
		},
		{
			name: "zero retries",
			config: &api.WorkConfig{
				MaxRetries:   0,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  api.BackoffTypeFixed,
			},
			retries:  0,
			error:    "network timeout",
			expected: false,
		},
		{
			name: "within limit",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  api.BackoffTypeFixed,
			},
			retries:  2,
			error:    "network timeout",
			expected: true,
		},
		{
			name: "at limit",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  api.BackoffTypeFixed,
			},
			retries:  3,
			error:    "network timeout",
			expected: false,
		},
		{
			name: "unlimited retries",
			config: &api.WorkConfig{
				MaxRetries:   -1,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  api.BackoffTypeFixed,
			},
			retries:  100,
			error:    "network timeout",
			expected: true,
		},
		{
			name: "permanent failure - success false",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  api.BackoffTypeFixed,
			},
			retries:  0,
			error:    "step returned success=false",
			expected: false,
		},
		{
			name: "permanent failure - success false with message",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  api.BackoffTypeFixed,
			},
			retries:  0,
			error:    "step returned success=false: payment denied",
			expected: false,
		},
		{
			name: "permanent failure - 4xx error",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  api.BackoffTypeFixed,
			},
			retries:  0,
			error:    "step returned HTTP error: HTTP 400",
			expected: false,
		},
	}

	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			step := &api.Step{
				ID:         "test-step",
				WorkConfig: sc.config,
			}

			workItem := &api.WorkState{
				RetryCount: sc.retries,
				Error:      sc.error,
			}

			result := env.Engine.ShouldRetry(step, workItem)
			assert.Equal(t, sc.expected, result)
		})
	}
}

func TestCalculateNextRetry(t *testing.T) {
	scenarios := []struct {
		name        string
		backoffType string
		backoffMs   int64
		maxBackoff  int64
		retryCount  int
		expectedMs  int64
	}{
		{
			name:        "fixed backoff",
			backoffType: api.BackoffTypeFixed,
			backoffMs:   1000,
			maxBackoff:  10000,
			retryCount:  0,
			expectedMs:  1000,
		},
		{
			name:        "fixed backoff retry 5",
			backoffType: api.BackoffTypeFixed,
			backoffMs:   1000,
			maxBackoff:  10000,
			retryCount:  5,
			expectedMs:  1000,
		},
		{
			name:        "linear backoff retry 0",
			backoffType: api.BackoffTypeLinear,
			backoffMs:   1000,
			maxBackoff:  10000,
			retryCount:  0,
			expectedMs:  1000,
		},
		{
			name:        "linear backoff retry 3",
			backoffType: api.BackoffTypeLinear,
			backoffMs:   1000,
			maxBackoff:  10000,
			retryCount:  3,
			expectedMs:  4000,
		},
		{
			name:        "exponential backoff retry 0",
			backoffType: api.BackoffTypeExponential,
			backoffMs:   1000,
			maxBackoff:  10000,
			retryCount:  0,
			expectedMs:  1000,
		},
		{
			name:        "exponential backoff retry 3",
			backoffType: api.BackoffTypeExponential,
			backoffMs:   1000,
			maxBackoff:  10000,
			retryCount:  3,
			expectedMs:  8000,
		},
		{
			name:        "exponential backoff capped",
			backoffType: api.BackoffTypeExponential,
			backoffMs:   1000,
			maxBackoff:  10000,
			retryCount:  10,
			expectedMs:  10000,
		},
	}

	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			config := &api.WorkConfig{
				BackoffMs:    sc.backoffMs,
				MaxBackoffMs: sc.maxBackoff,
				BackoffType:  sc.backoffType,
			}

			before := time.Now()
			nextRetry := env.Engine.CalculateNextRetry(config, sc.retryCount)
			after := time.Now()

			delay := nextRetry.Sub(before).Milliseconds()
			maxDelay := nextRetry.Sub(after).Milliseconds()

			assert.GreaterOrEqual(t, delay, sc.expectedMs-10)
			assert.LessOrEqual(t, maxDelay, sc.expectedMs+10)
		})
	}
}

func TestScheduleRetry(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep("test-step")
	step.WorkConfig = &api.WorkConfig{
		MaxRetries:   3,
		BackoffMs:    100,
		MaxBackoffMs: 1000,
		BackoffType:  api.BackoffTypeFixed,
	}

	err := env.Engine.RegisterStep(ctx, step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"test-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	flowID := timebox.ID("retry-workflow")
	err = env.Engine.StartWorkflow(
		ctx, flowID, plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	token := api.Token("test-token")
	err = env.Engine.StartWork(ctx, flowID, "test-step", token, api.Args{})
	require.NoError(t, err)

	err = env.Engine.FailWork(ctx, flowID, "test-step", token, "test error")
	require.NoError(t, err)

	err = env.Engine.ScheduleRetry(ctx, flowID, "test-step", token, "test error")
	require.NoError(t, err)

	flow, err := env.Engine.GetWorkflowState(ctx, flowID)
	require.NoError(t, err)

	exec := flow.Executions["test-step"]
	workItem := exec.WorkItems[token]
	assert.Equal(t, 1, workItem.RetryCount)
	assert.False(t, workItem.NextRetryAt.IsZero())
	assert.Equal(t, "test error", workItem.LastError)
	assert.Equal(t, api.WorkPending, workItem.Status)
}

func TestRetryExhaustion(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep("failing-step")
	step.WorkConfig = &api.WorkConfig{
		MaxRetries:   2,
		BackoffMs:    200,
		MaxBackoffMs: 1000,
		BackoffType:  api.BackoffTypeFixed,
	}

	env.MockClient.SetError("failing-step", assert.AnError)

	err := env.Engine.RegisterStep(ctx, step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"failing-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	flowID := timebox.ID("exhaustion-workflow")
	err = env.Engine.StartWorkflow(
		ctx, flowID, plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	time.Sleep(1500 * time.Millisecond)

	flow, err := env.Engine.GetWorkflowState(ctx, flowID)
	require.NoError(t, err)

	exec := flow.Executions["failing-step"]
	require.NotNil(t, exec.WorkItems)
	require.NotEmpty(t, exec.WorkItems)

	hasRetrying := false
	for _, item := range exec.WorkItems {
		if item.RetryCount >= 1 {
			hasRetrying = true
			t.Logf("Work item has retryCount=%d, status=%s", item.RetryCount, item.Status)
			break
		}
	}

	assert.True(t, hasRetrying, "at least one work item should have retried")
}

func TestFindRetriableSteps(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	state := &api.WorkflowState{
		Executions: map[timebox.ID]*api.ExecutionState{
			"step-1": {
				Status: api.StepPending,
				WorkItems: map[api.Token]*api.WorkState{
					"token-1": {
						Status:      api.WorkPending,
						RetryCount:  1,
						NextRetryAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
			"step-2": {
				Status: api.StepActive,
				WorkItems: map[api.Token]*api.WorkState{
					"token-2": {
						Status:      api.WorkActive,
						RetryCount:  1,
						NextRetryAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
			"step-3": {
				Status: api.StepPending,
				WorkItems: map[api.Token]*api.WorkState{
					"token-3": {
						Status:      api.WorkPending,
						RetryCount:  0,
						NextRetryAt: time.Time{},
					},
				},
			},
			"step-4": {
				Status: api.StepPending,
				WorkItems: map[api.Token]*api.WorkState{
					"token-4": {
						Status:      api.WorkPending,
						RetryCount:  2,
						NextRetryAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
			"step-5": {
				Status: api.StepCompleted,
				WorkItems: map[api.Token]*api.WorkState{
					"token-5": {
						Status:     api.WorkCompleted,
						RetryCount: 1,
					},
				},
			},
		},
	}

	retriable := env.Engine.FindRetrySteps(state)

	assert.Len(t, retriable, 2, "should find exactly 2 retriable steps")
	assert.Contains(t, retriable, timebox.ID("step-1"))
	assert.Contains(t, retriable, timebox.ID("step-4"))
}

func TestRecoverActiveWorkflows(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	ctx := context.Background()

	flowID1 := timebox.ID("workflow-1")
	flowID2 := timebox.ID("workflow-2")

	step := helpers.NewSimpleStep("step-1")
	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-1"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err := env.Engine.StartWorkflow(
		ctx, flowID1, plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.StartWorkflow(
		ctx, flowID2, plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Workflows created successfully
	workflow1, err := env.Engine.GetWorkflowState(ctx, flowID1)
	require.NoError(t, err)
	assert.NotNil(t, workflow1)

	workflow2, err := env.Engine.GetWorkflowState(ctx, flowID2)
	require.NoError(t, err)
	assert.NotNil(t, workflow2)
}

func TestConcurrentRecoveryState(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	ctx := context.Background()
	count := 10

	done := make(chan bool, count)

	step := helpers.NewSimpleStep("step-1")
	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-1"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	for i := 0; i < count; i++ {
		go func(id int) {
			flowID := timebox.ID(fmt.Sprintf("workflow-%d", id))
			err := env.Engine.StartWorkflow(
				ctx, flowID, plan, api.Args{}, api.Metadata{},
			)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	for i := 0; i < count; i++ {
		<-done
	}

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify all workflows were created
	for i := 0; i < count; i++ {
		flowID := timebox.ID(fmt.Sprintf("workflow-%d", i))
		workflow, err := env.Engine.GetWorkflowState(ctx, flowID)
		assert.NoError(t, err)
		assert.NotNil(t, workflow)
	}
}

func TestWorkConfigValidation(t *testing.T) {
	scenarios := []struct {
		name      string
		config    *api.WorkConfig
		expectErr bool
	}{
		{
			name:      "nil config valid",
			config:    nil,
			expectErr: false,
		},
		{
			name: "valid fixed config",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  api.BackoffTypeFixed,
			},
			expectErr: false,
		},
		{
			name: "negative backoff invalid",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    -1000,
				MaxBackoffMs: 10000,
				BackoffType:  api.BackoffTypeFixed,
			},
			expectErr: true,
		},
		{
			name: "max less than backoff invalid",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    10000,
				MaxBackoffMs: 1000,
				BackoffType:  api.BackoffTypeFixed,
			},
			expectErr: true,
		},
		{
			name: "invalid backoff type",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  "invalid",
			},
			expectErr: true,
		},
		{
			name: "empty backoff type",
			config: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    1000,
				MaxBackoffMs: 10000,
				BackoffType:  "",
			},
			expectErr: true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			step := &api.Step{
				ID:         "test-step",
				Name:       "Test Step",
				Type:       api.StepTypeSync,
				Version:    "1.0.0",
				HTTP:       &api.HTTPConfig{Endpoint: "http://test"},
				WorkConfig: sc.config,
			}

			err := step.Validate()
			if sc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
