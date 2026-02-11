package engine_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRecoveryActivation(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		flowID := api.FlowID("test-flow")

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitFor(helpers.FlowStarted(flowID), func() {
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
		env.Engine.Start()

		flowID := api.FlowID("test-flow")

		step := helpers.NewSimpleStep("step-1")

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitFor(helpers.FlowDeactivated(flowID), func() {
			err = env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		engineState, err := env.Engine.GetEngineState()
		assert.NoError(t, err)
		_, exists := engineState.Active[flowID]
		assert.False(t, exists)
	})
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
				MaxRetries:  0,
				Backoff:     1000,
				MaxBackoff:  10000,
				BackoffType: api.BackoffTypeFixed,
			},
			retries:  0,
			error:    "network timeout",
			expected: false,
		},
		{
			name: "within limit",
			config: &api.WorkConfig{
				MaxRetries:  3,
				Backoff:     1000,
				MaxBackoff:  10000,
				BackoffType: api.BackoffTypeFixed,
			},
			retries:  2,
			error:    "network timeout",
			expected: true,
		},
		{
			name: "at limit",
			config: &api.WorkConfig{
				MaxRetries:  3,
				Backoff:     1000,
				MaxBackoff:  10000,
				BackoffType: api.BackoffTypeFixed,
			},
			retries:  3,
			error:    "network timeout",
			expected: false,
		},
		{
			name: "unlimited retries",
			config: &api.WorkConfig{
				MaxRetries:  -1,
				Backoff:     1000,
				MaxBackoff:  10000,
				BackoffType: api.BackoffTypeFixed,
			},
			retries:  100,
			error:    "network timeout",
			expected: true,
		},
	}

	helpers.WithEngine(t, func(eng *engine.Engine) {
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

				result := eng.ShouldRetry(step, workItem)
				assert.Equal(t, sc.expected, result)
			})
		}
	})
}

func TestCalculateNextRetry(t *testing.T) {
	scenarios := []struct {
		name        string
		backoffType string
		backoff     int64
		maxBackoff  int64
		retryCount  int
		expected    int64
	}{
		{
			name:        "fixed backoff",
			backoffType: api.BackoffTypeFixed,
			backoff:     1000,
			maxBackoff:  10000,
			retryCount:  0,
			expected:    1000,
		},
		{
			name:        "fixed backoff retry 5",
			backoffType: api.BackoffTypeFixed,
			backoff:     1000,
			maxBackoff:  10000,
			retryCount:  5,
			expected:    1000,
		},
		{
			name:        "linear backoff retry 0",
			backoffType: api.BackoffTypeLinear,
			backoff:     1000,
			maxBackoff:  10000,
			retryCount:  0,
			expected:    1000,
		},
		{
			name:        "linear backoff retry 3",
			backoffType: api.BackoffTypeLinear,
			backoff:     1000,
			maxBackoff:  10000,
			retryCount:  3,
			expected:    4000,
		},
		{
			name:        "exponential backoff retry 0",
			backoffType: api.BackoffTypeExponential,
			backoff:     1000,
			maxBackoff:  10000,
			retryCount:  0,
			expected:    1000,
		},
		{
			name:        "exponential backoff retry 3",
			backoffType: api.BackoffTypeExponential,
			backoff:     1000,
			maxBackoff:  10000,
			retryCount:  3,
			expected:    8000,
		},
		{
			name:        "exponential backoff capped",
			backoffType: api.BackoffTypeExponential,
			backoff:     1000,
			maxBackoff:  10000,
			retryCount:  10,
			expected:    10000,
		},
	}

	helpers.WithEngine(t, func(eng *engine.Engine) {
		for _, sc := range scenarios {
			t.Run(sc.name, func(t *testing.T) {
				config := &api.WorkConfig{
					Backoff:     sc.backoff,
					MaxBackoff:  sc.maxBackoff,
					BackoffType: sc.backoffType,
				}

				before := time.Now()
				nextRetry := eng.CalculateNextRetry(config, sc.retryCount)
				after := time.Now()

				delay := nextRetry.Sub(before).Milliseconds()
				maxDelay := nextRetry.Sub(after).Milliseconds()

				assert.GreaterOrEqual(t, delay, sc.expected-10)
				assert.LessOrEqual(t, maxDelay, sc.expected+10)
			})
		}
	})
}

func TestRetryDefaults(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		config := &api.WorkConfig{
			Backoff:     750,
			MaxBackoff:  1200,
			BackoffType: "unknown",
		}

		start := time.Now()
		nextRetry := eng.CalculateNextRetry(config, 5)
		delay := nextRetry.Sub(start).Milliseconds()

		assert.GreaterOrEqual(t, delay, int64(740))
		assert.LessOrEqual(t, delay, int64(1210))
	})
}

func TestRetryExhaustion(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewSimpleStep("failing-step")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			Backoff:     200,
			MaxBackoff:  1000,
			BackoffType: api.BackoffTypeFixed,
		}

		env.MockClient.SetError("failing-step",
			fmt.Errorf("%w: %w", api.ErrWorkNotCompleted, assert.AnError))

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"failing-step"},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("exhaustion-flow")
		env.WaitFor(helpers.WorkRetryScheduled(api.FlowStep{
			FlowID: flowID,
			StepID: "failing-step",
		}), func() {
			err = env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		flow, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		exec := flow.Executions["failing-step"]
		if assert.NotNil(t, exec) && assert.NotNil(t, exec.WorkItems) {
			found := false
			for _, item := range exec.WorkItems {
				if item.RetryCount >= 1 {
					found = true
					break
				}
			}
			assert.True(t, found)
		}
	})
}

func TestFindRetriableSteps(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
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

		retryable := eng.FindRetrySteps(state)

		// Should include:
		// - step-1: WorkPending with NextRetryAt
		// - step-2: WorkActive (always retryable during recovery)
		// - step-4: WorkPending with NextRetryAt
		assert.Len(t, retryable, 3)
		assert.Contains(t, retryable, api.StepID("step-1"))
		assert.Contains(t, retryable, api.StepID("step-2"))
		assert.Contains(t, retryable, api.StepID("step-4"))
	})
}

func TestRecoverActiveFlows(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		flowID1 := api.FlowID("flow-1")
		flowID2 := api.FlowID("flow-2")

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitForCount(2, helpers.FlowStarted(
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
		env.Engine.Start()

		step := helpers.NewSimpleStep("retry-active")
		step.Type = api.StepTypeAsync

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-recover-active")
		env.WaitFor(helpers.WorkStarted(api.FlowStep{
			FlowID: flowID,
			StepID: step.ID,
		}), func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		env.WaitFor(helpers.WorkStarted(api.FlowStep{
			FlowID: flowID,
			StepID: step.ID,
		}), func() {
			assert.NoError(t, env.Engine.RecoverFlow(flowID))
		})
	})
}

func TestConcurrentRecoveryState(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

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
			len(flowIDs), helpers.FlowStarted(flowIDs...),
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
				MaxRetries:  3,
				Backoff:     1000,
				MaxBackoff:  10000,
				BackoffType: api.BackoffTypeFixed,
			},
			expectErr: false,
		},
		{
			name: "negative backoff invalid",
			config: &api.WorkConfig{
				MaxRetries:  3,
				Backoff:     -1000,
				MaxBackoff:  10000,
				BackoffType: api.BackoffTypeFixed,
			},
			expectErr: true,
		},
		{
			name: "max less than backoff invalid",
			config: &api.WorkConfig{
				MaxRetries:  3,
				Backoff:     10000,
				MaxBackoff:  1000,
				BackoffType: api.BackoffTypeFixed,
			},
			expectErr: true,
		},
		{
			name: "invalid backoff type",
			config: &api.WorkConfig{
				MaxRetries:  3,
				Backoff:     1000,
				MaxBackoff:  10000,
				BackoffType: "invalid",
			},
			expectErr: true,
		},
		{
			name: "empty backoff type",
			config: &api.WorkConfig{
				MaxRetries:  3,
				Backoff:     1000,
				MaxBackoff:  10000,
				BackoffType: "",
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

		flowState, err := eng.GetFlowState(flowID)
		assert.NoError(t, err)
		flowState.Status = api.FlowCompleted

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
		env.Engine.Start()

		step := helpers.NewSimpleStep("step-1")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			Backoff:     100,
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
		env.WaitFor(helpers.StepStarted(api.FlowStep{
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
		env.Engine.Start()

		step := helpers.NewSimpleStep("step-1")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			Backoff:     100,
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
		env.WaitFor(helpers.StepStarted(api.FlowStep{
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
		env.Engine.Start()

		step := helpers.NewSimpleStep("failing-step")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			Backoff:     100,
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
		env.WaitFor(helpers.WorkRetryScheduled(api.FlowStep{
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
		env.Engine.Start()

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		var err error
		flowID1 := api.FlowID("flow-1")
		flowID2 := api.FlowID("flow-2")
		env.WaitForCount(2, helpers.FlowStarted(
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
		env.Engine.Start()

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		var err error
		flowID1 := api.FlowID("good-flow")
		flowID2 := api.FlowID("bad-flow")
		env.WaitForCount(2, helpers.FlowStarted(
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
		env.Engine.Start()

		flowID := api.FlowID("nil-work-flow")

		step := helpers.NewSimpleStep("step-1")
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-1"},
			Steps: api.Steps{step.ID: step},
		}

		var err error
		env.WaitFor(helpers.FlowStarted(flowID), func() {
			err = env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		err = env.Engine.RecoverFlow(flowID)
		assert.NoError(t, err)
	})
}

func TestPendingWorkNotRetryable(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		state := &api.FlowState{
			ID:     "test-flow",
			Status: api.FlowActive,
			Plan: &api.ExecutionPlan{
				Goals: []api.StepID{"step-1"},
				Steps: api.Steps{
					"step-1": helpers.NewSimpleStep("step-1"),
				},
			},
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
			},
		}

		retryable := eng.FindRetrySteps(state)
		assert.Len(t, retryable, 1)
	})
}

func TestWorkItemNoNextRetry(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		state := &api.FlowState{
			ID:     "test-flow",
			Status: api.FlowActive,
			Plan: &api.ExecutionPlan{
				Goals: []api.StepID{"step-1"},
				Steps: api.Steps{
					"step-1": helpers.NewSimpleStep("step-1"),
				},
			},
			Executions: api.Executions{
				"step-1": {
					Status: api.StepPending,
					WorkItems: api.WorkItems{
						"token-1": {
							Status:      api.WorkPending,
							RetryCount:  0,
							NextRetryAt: time.Time{},
						},
					},
				},
			},
		}

		retryable := eng.FindRetrySteps(state)
		assert.Empty(t, retryable)
	})
}

func TestNextRetryNilConfig(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		nextRetry := eng.CalculateNextRetry(nil, 0)
		assert.False(t, nextRetry.IsZero())
	})
}

func TestFindRetryEmptyWorkItems(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		state := &api.FlowState{
			Executions: api.Executions{
				"step-1": {
					WorkItems: nil,
				},
				"step-2": {
					WorkItems: api.WorkItems{},
				},
			},
		}

		retryable := eng.FindRetrySteps(state)
		assert.Empty(t, retryable)
	})
}
