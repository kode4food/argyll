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
				InitBackoff: 1000,
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
				InitBackoff: 1000,
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
				InitBackoff: 1000,
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
				InitBackoff: 1000,
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

				work := &api.WorkState{
					RetryCount: sc.retries,
					Error:      sc.error,
				}

				result := eng.ShouldRetry(step, work)
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

	base := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithEngineDeps(t, engine.Dependencies{
		Clock: func() time.Time { return base },
	}, func(eng *engine.Engine) {
		for _, sc := range scenarios {
			t.Run(sc.name, func(t *testing.T) {
				config := &api.WorkConfig{
					InitBackoff: sc.backoff,
					MaxBackoff:  sc.maxBackoff,
					BackoffType: sc.backoffType,
				}

				nextRetry := eng.CalculateNextRetry(config, sc.retryCount)
				expected := base.Add(
					time.Duration(sc.expected) * time.Millisecond,
				)
				assert.Equal(t, expected, nextRetry)
			})
		}
	})
}

func TestRetryDefaults(t *testing.T) {
	base := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithEngineDeps(t, engine.Dependencies{
		Clock: func() time.Time { return base },
	}, func(eng *engine.Engine) {
		config := &api.WorkConfig{
			InitBackoff: 750,
			MaxBackoff:  1200,
			BackoffType: "unknown",
		}

		nextRetry := eng.CalculateNextRetry(config, 5)
		assert.Equal(t,
			base.Add(750*time.Millisecond),
			nextRetry,
		)
	})
}

func TestRetryExhaustion(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("failing-step")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			InitBackoff: 200,
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
			for _, work := range exec.WorkItems {
				if work.RetryCount >= 1 {
					found = true
					break
				}
			}
			assert.True(t, found)
		}
	})
}

func TestFindRetrySteps(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
		state := &api.FlowState{
			Executions: api.Executions{
				"step-1": {
					Status: api.StepPending,
					WorkItems: api.WorkItems{
						"token-1": {
							Status:      api.WorkPending,
							RetryCount:  1,
							NextRetryAt: now.Add(time.Hour),
						},
					},
				},
				"step-2": {
					Status: api.StepActive,
					WorkItems: api.WorkItems{
						"token-2": {
							Status:      api.WorkActive,
							RetryCount:  1,
							NextRetryAt: now.Add(time.Hour),
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
							NextRetryAt: now.Add(time.Hour),
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
		assert.Len(t, retryable, 3)
		assert.Contains(t, retryable, api.StepID("step-1"))
		assert.Contains(t, retryable, api.StepID("step-2"))
		assert.Contains(t, retryable, api.StepID("step-4"))
	})
}

func TestPendingWorkNotRetryable(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
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
							NextRetryAt: now.Add(time.Hour),
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

func TestFindRetryStepsActivePending(t *testing.T) {
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
					Status: api.StepActive,
					WorkItems: api.WorkItems{
						"token-1": {
							Status: api.WorkPending,
						},
					},
				},
			},
		}

		retryable := eng.FindRetrySteps(state)
		assert.Len(t, retryable, 1)
		assert.Contains(t, retryable, api.StepID("step-1"))
	})
}

func TestNextRetryNilConfig(t *testing.T) {
	base := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithEngineDeps(t, engine.Dependencies{
		Clock: func() time.Time { return base },
	}, func(eng *engine.Engine) {
		nextRetry := eng.CalculateNextRetry(nil, 0)
		assert.Equal(t, base.Add(time.Second), nextRetry)
	})
}

func TestNextRetryParallelismOnlyConfig(t *testing.T) {
	base := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	helpers.WithEngineDeps(t, engine.Dependencies{
		Clock: func() time.Time { return base },
	}, func(eng *engine.Engine) {
		cfg := &api.WorkConfig{
			Parallelism: 2,
		}
		nextRetry := eng.CalculateNextRetry(cfg, 0)
		assert.Equal(t, base.Add(time.Second), nextRetry)
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
