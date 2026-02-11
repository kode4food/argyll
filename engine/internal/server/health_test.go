package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestStartStop(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		checker := server.NewHealthChecker(env.Engine, env.EventHub)
		assert.NotNil(t, checker)

		checker.Start()
		checker.Stop()
	})
}

func TestGetStepHealth(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := helpers.NewSimpleStep("health-test-step")

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		engineState, err := env.Engine.GetEngineState()
		assert.NoError(t, err)
		health, ok := engineState.Health["health-test-step"]
		assert.True(t, ok)
		assert.NotNil(t, health)
		assert.Equal(t, api.HealthUnknown, health.Status)
	})
}

func TestGetStepHealthNotFound(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		engineState, err := env.Engine.GetEngineState()
		assert.NoError(t, err)
		_, ok := engineState.Health["nonexistent-step"]
		assert.False(t, ok)
	})
}

func TestWithRealHealthCheck(t *testing.T) {
	healthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(
				map[string]string{"status": "healthy"},
			)
		}),
	)
	defer healthServer.Close()

	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := &api.Step{
			ID:   "real-health-step",
			Name: "Real Health Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint:    healthServer.URL + "/execute",
				HealthCheck: healthServer.URL + "/health",
			},
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		checker := server.NewHealthChecker(env.Engine, env.EventHub)
		checker.Start()
		defer checker.Stop()

		engineState, err := env.Engine.GetEngineState()
		assert.NoError(t, err)
		health, ok := engineState.Health["real-health-step"]
		assert.True(t, ok)
		assert.NotNil(t, health)
	})
}

func TestRecentSuccess(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := helpers.NewSimpleStep("recent-success-step")

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		checker := server.NewHealthChecker(env.Engine, env.EventHub)
		checker.Start()
		defer checker.Stop()

		err = env.RaiseFlowEvents("wf-test",
			helpers.FlowEvent{
				Type: api.EventTypeFlowStarted,
				Data: api.FlowStartedEvent{
					FlowID: "wf-test",
					Plan: &api.ExecutionPlan{
						Goals: []api.StepID{step.ID},
						Steps: api.Steps{step.ID: step},
					},
					Init:     api.Args{},
					Metadata: api.Metadata{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeStepCompleted,
				Data: api.StepCompletedEvent{
					StepID: step.ID,
					FlowID: "wf-test",
				},
			},
		)
		assert.NoError(t, err)

		engineState, err := env.Engine.GetEngineState()
		assert.NoError(t, err)
		health, ok := engineState.Health["recent-success-step"]
		assert.True(t, ok)
		assert.NotNil(t, health)
	})
}

func TestHealthCheckMarksUnhealthy(t *testing.T) {
	healthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer healthServer.Close()
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := &api.Step{
			ID:   "unhealthy-step",
			Name: "Unhealthy Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint:    healthServer.URL + "/execute",
				HealthCheck: healthServer.URL + "/health",
			},
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		checker := server.NewHealthChecker(env.Engine, env.EventHub)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(
			"unhealthy-step", api.HealthUnhealthy,
		), func() {
			checker.Start()
		})

		state, err := env.Engine.GetEngineState()
		assert.NoError(t, err)
		health, ok := state.Health["unhealthy-step"]
		assert.True(t, ok)
		assert.Equal(t, api.HealthUnhealthy, health.Status)
	})
}

func TestEventLoopUnmarshalError(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		checker := server.NewHealthChecker(env.Engine, env.EventHub)
		checker.Start()
		defer checker.Stop()

		err := env.RaiseFlowEvents("wf-test", helpers.FlowEvent{
			Type: api.EventTypeStepCompleted,
			Data: map[string]any{"step_id": 123},
		})
		assert.NoError(t, err)
	})
}

func TestCheckMultipleHTTPSteps(t *testing.T) {
	healthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer healthServer.Close()
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		for i := 0; i < 3; i++ {
			step := &api.Step{
				ID:   api.StepID("multi-health-" + string(rune('a'+i))),
				Name: "Multi Health Step",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint:    healthServer.URL + "/execute",
					HealthCheck: healthServer.URL + "/health",
				},
			}

			err := env.Engine.RegisterStep(step)
			assert.NoError(t, err)
		}

		checker := server.NewHealthChecker(env.Engine, env.EventHub)
		checker.Start()
		defer checker.Stop()

		engineState, err := env.Engine.GetEngineState()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(engineState.Health), 3)
	})
}

func TestNonStepCompletedEvent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		checker := server.NewHealthChecker(env.Engine, env.EventHub)
		checker.Start()
		defer checker.Stop()

		err := env.RaiseFlowEvents("wf-test", helpers.FlowEvent{
			Type: api.EventTypeFlowStarted,
			Data: api.FlowStartedEvent{
				FlowID: "wf-test",
				Plan: &api.ExecutionPlan{
					Goals: []api.StepID{},
					Steps: api.Steps{},
				},
				Init:     api.Args{},
				Metadata: api.Metadata{},
			},
		})
		assert.NoError(t, err)
	})
}
