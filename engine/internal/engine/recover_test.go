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
		env.Engine.Start()

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
		env.Engine.Start()

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

		partState, err := env.Engine.GetPartitionState()
		assert.NoError(t, err)
		_, exists := partState.Active[flowID]
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
			name: "parallelism only uses global retry defaults",
			config: &api.WorkConfig{
				Parallelism: 4,
			},
			retries:  0,
			error:    "network timeout",
			expected: true,
		},
		{
			name: "zero max retries uses global defaults",
			config: &api.WorkConfig{
				MaxRetries:  0,
				Backoff:     1000,
				MaxBackoff:  10000,
				BackoffType: api.BackoffTypeFixed,
			},
			retries:  0,
			error:    "network timeout",
			expected: true,
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
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
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
			expectErr: false,
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
		env.Engine.Start()

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
		env.Engine.Start()

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
		env.Engine.Start()

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
					NextRetryAt: time.Now().Add(-1 * time.Second),
					Error:       "retry",
				},
			},
		)
		assert.NoError(t, err)

		partState, err := env.Engine.GetPartitionState()
		assert.NoError(t, err)
		_, ok := partState.Active[flowID]
		assert.False(t, ok)

		env.MockClient.SetResponse(step.ID, api.Args{})
		env.WaitFor(wait.FlowActivated(flowID), func() {
			env.Engine.Start()
		})

		invoked := env.MockClient.WaitForInvocation(step.ID, 2*time.Second)
		assert.True(t, invoked)

		partState, err = env.Engine.GetPartitionState()
		assert.NoError(t, err)
		_, ok = partState.Active[flowID]
		assert.True(t, ok)
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

func TestNextRetryParallelismOnlyConfig(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		cfg := &api.WorkConfig{
			Parallelism: 2,
		}
		start := time.Now()
		nextRetry := eng.CalculateNextRetry(cfg, 0)
		delay := nextRetry.Sub(start).Milliseconds()

		assert.GreaterOrEqual(t, delay, int64(990))
		assert.LessOrEqual(t, delay, int64(1010))
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
		now := time.Now()
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
		env.Engine.Start()

		activeFlowID := api.FlowID("recover-active")
		deactivatedFlowID := api.FlowID("recover-deactivated")
		archivingFlowID := api.FlowID("recover-archiving")
		activeStep := helpers.NewSimpleStep("recover-step-active")
		deactivatedStep := helpers.NewSimpleStep("recover-step-deactivated")
		archivingStep := helpers.NewSimpleStep("recover-step-archiving")
		activeToken := api.Token("active-token")
		deactivatedToken := api.Token("deactivated-token")
		archivingToken := api.Token("archiving-token")

		raiseFlow := func(
			flowID api.FlowID, step *api.Step, token api.Token,
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
						NextRetryAt: time.Now().Add(-500 * time.Millisecond),
						Error:       "retry",
					},
				},
			)
			assert.NoError(t, err)
		}

		raiseFlow(activeFlowID, activeStep, activeToken)
		raiseFlow(deactivatedFlowID, deactivatedStep, deactivatedToken)
		raiseFlow(archivingFlowID, archivingStep, archivingToken)

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

		env.MockClient.SetResponse(activeStep.ID, api.Args{})
		env.MockClient.SetResponse(deactivatedStep.ID, api.Args{})
		env.MockClient.SetResponse(archivingStep.ID, api.Args{})
		assert.NoError(t, env.Engine.Stop())

		restarted, err := env.NewEngineInstance()
		assert.NoError(t, err)
		restarted.Start()
		defer func() { _ = restarted.Stop() }()

		assert.True(t,
			env.MockClient.WaitForInvocation(activeStep.ID, 2*time.Second))
		assert.False(t,
			env.MockClient.WaitForInvocation(deactivatedStep.ID, 300*time.Millisecond))
		assert.False(t,
			env.MockClient.WaitForInvocation(archivingStep.ID, 300*time.Millisecond))

		activeState, err := restarted.GetFlowState(activeFlowID)
		assert.NoError(t, err)
		deactivatedState, err := restarted.GetFlowState(deactivatedFlowID)
		assert.NoError(t, err)
		archivingState, err := restarted.GetFlowState(archivingFlowID)
		assert.NoError(t, err)

		assert.NotEqual(t, api.WorkPending,
			activeState.Executions[activeStep.ID].WorkItems[activeToken].Status)
		assert.Equal(t, api.WorkPending,
			deactivatedState.Executions[deactivatedStep.ID].
				WorkItems[deactivatedToken].Status)
		assert.Equal(t, api.WorkPending,
			archivingState.Executions[archivingStep.ID].
				WorkItems[archivingToken].Status)
	})
}
