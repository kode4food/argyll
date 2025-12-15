package engine_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRecoveryActivation(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	ctx := context.Background()
	flowID := api.FlowID("test-flow")

	step := helpers.NewSimpleStep("step-1")
	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-1"},
		Steps: api.Steps{step.ID: step},
	}

	err := env.Engine.StartFlow(
		ctx, flowID, plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	// Wait a bit for the event loop to process the flow started event
	time.Sleep(100 * time.Millisecond)

	// Check that flow was created and started
	flow, err := env.Engine.GetFlowState(ctx, flowID)
	assert.NoError(t, err)
	assert.NotNil(t, flow)
	assert.Equal(t, flowID, flow.ID)
	assert.False(t, flow.CreatedAt.IsZero())
}

func TestRecoveryDeactivation(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	ctx := context.Background()
	flowID := api.FlowID("test-flow")

	step := helpers.NewSimpleStep("step-1")

	err := env.Engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-1"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	// Wait for flow to complete naturally
	env.WaitForFlowStatus(t, ctx, flowID, 5*time.Second)

	// Poll for flow to be removed from active flows (deactivation is async)
	assert.Eventually(t, func() bool {
		engineState, err := env.Engine.GetEngineState(ctx)
		if err != nil {
			return false
		}
		_, exists := engineState.ActiveFlows[flowID]
		return !exists
	}, 5*time.Second, 10*time.Millisecond, "flow should be deactivated")
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

func TestCalculateNextRetryDefaultsToFixed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	config := &api.WorkConfig{
		BackoffMs:    750,
		MaxBackoffMs: 1200,
		BackoffType:  "unknown",
	}

	start := time.Now()
	nextRetry := env.Engine.CalculateNextRetry(config, 5)
	delay := nextRetry.Sub(start).Milliseconds()

	assert.GreaterOrEqual(t, delay, int64(740))
	assert.LessOrEqual(t, delay, int64(1210))
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

	env.MockClient.SetError("failing-step",
		fmt.Errorf("%w: %w", api.ErrWorkNotCompleted, assert.AnError))

	err := env.Engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"failing-step"},
		Steps: api.Steps{step.ID: step},
	}

	flowID := api.FlowID("exhaustion-flow")
	err = env.Engine.StartFlow(
		ctx, flowID, plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	time.Sleep(1500 * time.Millisecond)

	flow, err := env.Engine.GetFlowState(ctx, flowID)
	assert.NoError(t, err)

	exec := flow.Executions["failing-step"]
	assert.NotNil(t, exec.WorkItems)
	assert.NotEmpty(t, exec.WorkItems)

	hasRetrying := false
	for _, item := range exec.WorkItems {
		if item.RetryCount >= 1 {
			hasRetrying = true
			t.Logf("Work item has retryCount=%d, status=%s",
				item.RetryCount, item.Status)
			break
		}
	}

	assert.True(t, hasRetrying)
}

func TestFindRetriableSteps(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	state := &api.FlowState{
		Executions: api.Executions{
			"step-1": {
				Status: api.StepPending,
				WorkItems: api.WorkItems{
					"token-1": {
						Status:      api.WorkPending,
						RetryCount:  1,
						NextRetryAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
			"step-2": {
				Status: api.StepActive,
				WorkItems: api.WorkItems{
					"token-2": {
						Status:      api.WorkActive,
						RetryCount:  1,
						NextRetryAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
			"step-3": {
				Status: api.StepPending,
				WorkItems: api.WorkItems{
					"token-3": {
						Status: api.WorkPending,
					},
				},
			},
			"step-4": {
				Status: api.StepPending,
				WorkItems: api.WorkItems{
					"token-4": {
						Status:      api.WorkPending,
						RetryCount:  2,
						NextRetryAt: time.Now().Add(1 * time.Hour),
					},
				},
			},
			"step-5": {
				Status: api.StepCompleted,
				WorkItems: api.WorkItems{
					"token-5": {
						Status:     api.WorkSucceeded,
						RetryCount: 1,
					},
				},
			},
		},
	}

	retryable := env.Engine.FindRetrySteps(state)

	assert.Len(t, retryable, 2)
	assert.Contains(t, retryable, api.StepID("step-1"))
	assert.Contains(t, retryable, api.StepID("step-4"))
}

func TestRecoverActiveFlows(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	ctx := context.Background()

	flowID1 := api.FlowID("flow-1")
	flowID2 := api.FlowID("flow-2")

	step := helpers.NewSimpleStep("step-1")
	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-1"},
		Steps: api.Steps{step.ID: step},
	}

	err := env.Engine.StartFlow(
		ctx, flowID1, plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	err = env.Engine.StartFlow(
		ctx, flowID2, plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Flows created successfully
	flow1, err := env.Engine.GetFlowState(ctx, flowID1)
	assert.NoError(t, err)
	assert.NotNil(t, flow1)

	flow2, err := env.Engine.GetFlowState(ctx, flowID2)
	assert.NoError(t, err)
	assert.NotNil(t, flow2)
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
		Goals: []api.StepID{"step-1"},
		Steps: api.Steps{step.ID: step},
	}

	for i := range count {
		go func(id int) {
			flowID := api.FlowID(fmt.Sprintf("flow-%d", id))
			err := env.Engine.StartFlow(
				ctx, flowID, plan, api.Args{}, api.Metadata{},
			)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	for range count {
		<-done
	}

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify all flows were created
	for i := range count {
		flowID := api.FlowID(fmt.Sprintf("flow-%d", i))
		flow, err := env.Engine.GetFlowState(ctx, flowID)
		assert.NoError(t, err)
		assert.NotNil(t, flow)
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

func TestIsRetryableUtility(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		state  *api.WorkState
		expect bool
	}{
		{
			name:   "zero time is not retryable",
			state:  &api.WorkState{NextRetryAt: time.Time{}},
			expect: false,
		},
		{
			name:   "future retry not ready",
			state:  &api.WorkState{NextRetryAt: now.Add(time.Second)},
			expect: false,
		},
		{
			name:   "past retry is ready",
			state:  &api.WorkState{NextRetryAt: now.Add(-time.Second)},
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, isRetryable(tt.state, now))
		})
	}
}

func TestIsRetryable(t *testing.T) {
	now := time.Now()

	assert.False(t, isRetryable(&api.WorkState{}, now))
	assert.False(t, isRetryable(&api.WorkState{
		NextRetryAt: now.Add(time.Minute),
	}, now))
	assert.True(t, isRetryable(&api.WorkState{
		NextRetryAt: now.Add(-time.Minute),
	}, now))
}

func isRetryable(workItem *api.WorkState, now time.Time) bool {
	return !workItem.NextRetryAt.IsZero() && workItem.NextRetryAt.Before(now)
}
