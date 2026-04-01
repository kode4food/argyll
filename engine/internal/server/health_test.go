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
		st := helpers.NewSimpleStep("health-test-step")

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health["health-test-step"]; ok {
				assert.NotNil(t, h)
				assert.Equal(t, api.HealthUnknown, h.Status)
				return
			}
		}
		assert.Fail(t, "health-test-step not found in any node")
	})
}

func TestGetStepHealthNotFound(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			_, ok := node.Health["nonexistent-step"]
			assert.False(t, ok)
		}
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

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		found := false
		for _, node := range cluster.Nodes {
			if _, ok := node.Health["real-health-step"]; ok {
				found = true
				break
			}
		}
		assert.True(t, found)
	})
}

func TestRecentSuccess(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("recent-success-step")

		err := env.Engine.RegisterStep(st)
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
						Goals: []api.StepID{st.ID},
						Steps: api.Steps{st.ID: st},
					},
					Init:     api.Args{},
					Metadata: api.Metadata{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeStepCompleted,
				Data: api.StepCompletedEvent{
					StepID: st.ID,
					FlowID: "wf-test",
				},
			},
		)
		assert.NoError(t, err)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		found := false
		for _, node := range cluster.Nodes {
			if _, ok := node.Health["recent-success-step"]; ok {
				found = true
				break
			}
		}
		assert.True(t, found)
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

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		found := false
		for _, node := range cluster.Nodes {
			if h, ok := node.Health["unhealthy-step"]; ok {
				assert.Equal(t, api.HealthUnhealthy, h.Status)
				found = true
				break
			}
		}
		assert.True(t, found)
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
		for _, sfx := range []string{"a", "b", "c"} {
			step := &api.Step{
				ID:   api.StepID("multi-health-" + sfx),
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

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		healthCount := 0
		for _, node := range cluster.Nodes {
			healthCount += len(node.Health)
		}
		assert.GreaterOrEqual(t, healthCount, 3)
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
