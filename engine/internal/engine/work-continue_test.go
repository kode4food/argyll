package engine_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/flow"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
	"github.com/kode4food/argyll/engine/pkg/util"
)

func TestRetryPendingParallelism(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		st.ID = "retry-parallel"
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 500,
			MaxBackoff:  500,
			BackoffType: api.BackoffTypeFixed,
			Parallelism: 1,
		}
		st.Attributes["items"].Required = &api.RequiredConfig{ForEach: true}
		st.Attributes["items"].Type = api.TypeArray
		st.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetError(st.ID, api.ErrWorkNotCompleted)

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-retry-parallel")
		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			err := env.Engine.StartPlan(id, pl,
				flow.WithInit(api.InitArgs{"items": {[]any{"a", "b"}}}),
			)
			assert.NoError(t, err)
			w.ForEvents(2, wait.WorkRetryScheduledDistinct(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}))
			env.MockClient.ClearError(st.ID)
			env.MockClient.SetResponse(st.ID, api.Args{"result": "ok"})
			w.ForEvent(wait.FlowTerminal(id))
		})
		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepCompleted, ex.Status)
		assert.Len(t, ex.WorkItems, 2)
		for _, item := range ex.WorkItems {
			assert.Equal(t, api.WorkSucceeded, item.Status)
			assert.GreaterOrEqual(t, item.RetryCount, 1)
		}
	})
}

func TestRetryDeferredOnUnhealthyNode(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cfg := util.MutableCopy(env.Config)
		cfg.NodeID = "node-retry-unhealthy"

		peer, unsub, err := env.NewEngineWithConfig(cfg, env.Dependencies())
		assert.NoError(t, err)
		if !assert.NotNil(t, peer) {
			return
		}
		defer func() {
			unsub()
			assert.NoError(t, peer.Stop())
		}()

		st := helpers.NewSimpleStep("retry-unhealthy")
		env.MockClient.SetResponse(st.ID, api.Args{})

		assert.NoError(t, env.Engine.UpdateStepHealth(
			st.ID, api.HealthUnhealthy, "offline",
		))
		assert.NoError(t, peer.UpdateStepHealth(
			st.ID, api.HealthUnhealthy, "offline",
		))

		assert.NoError(t, env.Engine.Start())
		assert.NoError(t, peer.Start())

		id := api.FlowID("wf-retry-unhealthy")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-retry-unhealthy")
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		assert.NoError(t, env.SeedStartedWork(fs, pl, tkn))
		assert.NoError(t, env.RaiseFlowEvents(id,
			helpers.FlowEvent{
				Type: api.EventTypeWorkNotCompleted,
				Data: api.WorkNotCompletedEvent{
					FlowID: id,
					StepID: st.ID,
					Token:  tkn,
					Error:  "transient",
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeWorkRetryScheduled,
				Data: api.WorkRetryScheduledEvent{
					FlowID:      id,
					StepID:      st.ID,
					Token:       tkn,
					RetryCount:  1,
					NextRetryAt: scheduler.Now().Add(20 * time.Millisecond),
					Error:       "transient",
				},
			},
		))
		assert.False(t,
			env.MockClient.WaitForInvocation(st.ID, 100*time.Millisecond),
		)

		assert.NoError(t, peer.UpdateStepHealth(st.ID, api.HealthHealthy, ""))

		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkSucceeded, work.Status)
	})
}

func TestRetryOnHealthyPeer(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cfg := util.MutableCopy(env.Config)
		cfg.NodeID = "node-2"

		peer, unsub, err := env.NewEngineWithConfig(cfg, env.Dependencies())
		assert.NoError(t, err)
		if !assert.NotNil(t, peer) {
			return
		}
		defer func() {
			unsub()
			assert.NoError(t, peer.Stop())
		}()

		assert.NoError(t, env.Engine.Start())
		assert.NoError(t, peer.Start())

		st := helpers.NewSimpleStep("retry-shared")
		env.MockClient.SetResponse(st.ID, api.Args{"output": "ok"})

		assert.NoError(t,
			env.Engine.UpdateStepHealth(
				st.ID, api.HealthUnhealthy, "connection refused",
			),
		)
		assert.NoError(t,
			peer.UpdateStepHealth(st.ID, api.HealthHealthy, ""),
		)

		id := api.FlowID("wf-retry-shared")
		tkn := api.Token("retry-token")
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(
			wait.WorkSucceeded(api.FlowStep{FlowID: id, StepID: st.ID}),
			func() {
				assert.NoError(t, env.RaiseFlowEvents(
					id,
					helpers.FlowEvent{
						Type: api.EventTypeFlowStarted,
						Data: api.FlowStartedEvent{
							FlowID: id,
							Plan:   pl,
							Init:   api.InitArgs{},
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
						Type: api.EventTypeWorkRetryScheduled,
						Data: api.WorkRetryScheduledEvent{
							FlowID:      id,
							StepID:      st.ID,
							Token:       tkn,
							RetryCount:  1,
							NextRetryAt: scheduler.Now(),
							Error:       "retry",
						},
					},
				))
			},
		)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkSucceeded, work.Status)
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
				st := &api.Step{
					ID:         "test-step",
					WorkConfig: sc.config,
				}

				work := api.WorkState{
					RetryCount: sc.retries,
					Error:      sc.error,
				}

				result := eng.ShouldRetry(st, work)
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

		st := helpers.NewSimpleStep("failing-step")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			InitBackoff: 200,
			MaxBackoff:  1000,
			BackoffType: api.BackoffTypeFixed,
		}

		env.MockClient.SetError("failing-step",
			errors.Join(api.ErrWorkNotCompleted, assert.AnError))

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{"failing-step"}},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("exhaustion-flow")
		env.WaitFor(wait.WorkRetryScheduled(api.FlowStep{
			FlowID: id,
			StepID: "failing-step",
		}), func() {
			err = env.Engine.StartPlan(id, pl)
			assert.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		ex := fl.Executions["failing-step"]
		if assert.NotNil(t, ex.WorkItems) {
			found := false
			for _, work := range ex.WorkItems {
				if work.RetryCount >= 1 {
					found = true
					break
				}
			}
			assert.True(t, found)
		}
	})
}

func TestHTTPRetryRecovers(t *testing.T) {
	var calls atomic.Int32
	stepServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) <= 2 {
				w.Header().Set("Content-Type", api.ProblemJSONContentType)
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(api.NewProblem(
					http.StatusServiceUnavailable, "temporary outage",
				))
				return
			}

			w.Header().Set("Content-Type", api.JSONContentType)
			_ = json.NewEncoder(w).Encode(api.Args{"result": "ok"})
		},
	))
	defer stepServer.Close()

	scripts := script.NewRegistry()
	deps := engine.Dependencies{
		Scripts: scripts,
		Steps: step.NewRegistry(builtins.All(
			builtins.NewHTTPClient(5*time.Second), nil,
		)),
	}
	helpers.WithTestEnvDeps(t, deps, func(env *helpers.TestEngineEnv) {
		cfg := util.MutableCopy(env.Config)
		cfg.Work = api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}
		deps := env.Dependencies()
		scripts := script.NewRegistry()
		deps.Scripts = scripts
		deps.Steps = step.NewRegistry(builtins.All(
			builtins.NewHTTPClient(5*time.Second), nil,
		))
		eng, unsubscribe, err := env.NewEngineWithConfig(cfg, deps)
		assert.NoError(t, err)
		if !assert.NotNil(t, eng) {
			return
		}
		defer unsubscribe()
		defer func() { assert.NoError(t, eng.Stop()) }()

		assert.NoError(t, eng.Start())

		st := helpers.NewStepWithOutputs("http-retry", "result")
		st.HTTP.Invoke.Endpoint = stepServer.URL
		assert.NoError(t, eng.RegisterStep(st))

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-http-retry")
		err = eng.StartPlan(id, pl)
		assert.NoError(t, err)
		fl := helpers.WaitForTerminalFlowState(t, eng, id)

		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, int32(3), calls.Load())
		assert.Equal(t, "ok", fl.Attributes["result"][0].Value)
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
