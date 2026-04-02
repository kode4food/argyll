package server_test

import (
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
		checker := server.NewHealthChecker(env.Engine)
		assert.NotNil(t, checker)

		checker.Start()
		checker.Stop()
	})
}

func TestRegisterStepMarksUnknown(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("health-test-step")

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[st.ID]; ok {
				assert.NotNil(t, h)
				assert.Equal(t, api.HealthUnknown, h.Status)
				return
			}
		}
		assert.Fail(t, "health-test-step not found in any node")
	})
}

func TestHealthCheckMarksHealthy(t *testing.T) {
	healthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer healthServer.Close()

	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := &api.Step{
			ID:   "healthy-step",
			Name: "Healthy Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint:    healthServer.URL + "/execute",
				HealthCheck: healthServer.URL + "/health",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(step))

		checker := server.NewHealthChecker(env.Engine)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(step.ID, api.HealthHealthy), func() {
			checker.Start()
		})

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[step.ID]; ok {
				assert.Equal(t, api.HealthHealthy, h.Status)
				return
			}
		}
		assert.Fail(t, "healthy-step not found in any node")
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
		assert.NoError(t, env.Engine.RegisterStep(step))

		checker := server.NewHealthChecker(env.Engine)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(step.ID, api.HealthUnhealthy), func() {
			checker.Start()
		})

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[step.ID]; ok {
				assert.Equal(t, api.HealthUnhealthy, h.Status)
				return
			}
		}
		assert.Fail(t, "unhealthy-step not found in any node")
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
			assert.NoError(t, env.Engine.RegisterStep(step))
		}

		checker := server.NewHealthChecker(env.Engine)
		checker.Start()
		defer checker.Stop()

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		n := 0
		for _, node := range cluster.Nodes {
			n += len(node.Health)
		}
		assert.GreaterOrEqual(t, n, 3)
	})
}
