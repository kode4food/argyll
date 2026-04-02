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

func TestRegisterStepUnknown(t *testing.T) {
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

func TestHealthCheckHealthy(t *testing.T) {
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

func TestHealthCheckNetworkFail(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := &api.Step{
			ID:   "network-failure-step",
			Name: "Network Failure Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint:    "http://127.0.0.1:1/execute",
				HealthCheck: "http://127.0.0.1:1/health",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(step))

		checker := server.NewHealthChecker(env.Engine)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(step.ID, api.HealthUnhealthy),
			func() {
				checker.Start()
			},
		)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[step.ID]; ok {
				assert.Equal(t, api.HealthUnhealthy, h.Status)
				assert.NotEmpty(t, h.Error)
				return
			}
		}
		assert.Fail(t, "network-failure-step not found in any node")
	})
}

func TestHealthCheckScript(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		step := &api.Step{
			ID:   "script-step",
			Name: "Script Step",
			Type: api.StepTypeScript,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "{:result 42}",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(step))
		assert.NoError(t,
			env.Engine.UpdateStepHealth(step.ID, api.HealthUnknown, ""),
		)

		checker := server.NewHealthChecker(env.Engine)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(step.ID, api.HealthHealthy),
			func() {
				checker.Start()
			},
		)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[step.ID]; ok {
				assert.Equal(t, api.HealthHealthy, h.Status)
				assert.Empty(t, h.Error)
				return
			}
		}
		assert.Fail(t, "script-step not found in any node")
	})
}

func TestHealthCheckFlowHealthy(t *testing.T) {
	healthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer healthServer.Close()

	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		goal := &api.Step{
			ID:   "goal-step",
			Name: "Goal Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint:    healthServer.URL + "/execute",
				HealthCheck: healthServer.URL + "/health",
			},
		}
		fl := &api.Step{
			ID:   "flow-step",
			Name: "Flow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{goal.ID},
			},
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(goal))
		assert.NoError(t, env.Engine.RegisterStep(fl))

		checker := server.NewHealthChecker(env.Engine)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(fl.ID, api.HealthHealthy),
			func() {
				checker.Start()
			},
		)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[fl.ID]; ok {
				assert.Equal(t, api.HealthHealthy, h.Status)
				return
			}
		}
		assert.Fail(t, "flow-step not found in any node")
	})
}

func TestHealthCheckFlowUnhealthy(t *testing.T) {
	healthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer healthServer.Close()

	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		goal := &api.Step{
			ID:   "bad-goal-step",
			Name: "Bad Goal Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint:    healthServer.URL + "/execute",
				HealthCheck: healthServer.URL + "/health",
			},
		}
		fl := &api.Step{
			ID:   "bad-flow-step",
			Name: "Bad Flow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{goal.ID},
			},
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(goal))
		assert.NoError(t, env.Engine.RegisterStep(fl))

		checker := server.NewHealthChecker(env.Engine)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(fl.ID, api.HealthUnhealthy),
			func() {
				checker.Start()
			},
		)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[fl.ID]; ok {
				assert.Equal(t, api.HealthUnhealthy, h.Status)
				assert.Contains(t, h.Error, goal.ID)
				return
			}
		}
		assert.Fail(t, "bad-flow-step not found in any node")
	})
}

func TestHealthCheckUnhealthy(t *testing.T) {
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
