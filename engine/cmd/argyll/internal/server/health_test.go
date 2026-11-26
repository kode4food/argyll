package server_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/server"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

const (
	agentStepID       api.StepID = "agent-step"
	secondAgentStepID api.StepID = "agent-step-second"
)

func TestStartStop(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		checker := server.NewHealthChecker(env.Engine, nil)
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
		st := &api.Step{
			ID:   "healthy-step",
			Name: "Healthy Step",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: healthServer.URL + "/execute",
				},
				Health: healthServer.URL + "/health",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		checker := server.NewHealthChecker(env.Engine, nil)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(st.ID, api.HealthHealthy), func() {
			checker.Start()
		})

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[st.ID]; ok {
				assert.Equal(t, api.HealthHealthy, h.Status)
				return
			}
		}
		assert.Fail(t, "healthy-step not found in any node")
	})
}

func TestHealthCheckNetworkFail(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := &api.Step{
			ID:   "network-failure-step",
			Name: "Network Failure Step",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:1/execute",
				},
				Health: "http://localhost:1/health",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		checker := server.NewHealthChecker(env.Engine, nil)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(st.ID, api.HealthUnhealthy),
			func() {
				checker.Start()
			},
		)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[st.ID]; ok {
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
		st := &api.Step{
			ID:   "script-step",
			Name: "Script Step",
			Type: api.StepTypeScript,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Source:   "return {result = 42}",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))
		assert.NoError(t, env.Engine.UpdateStepHealth(st.ID, api.HealthUnknown, ""))

		checker := server.NewHealthChecker(env.Engine, nil)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(st.ID, api.HealthHealthy),
			func() {
				checker.Start()
			},
		)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[st.ID]; ok {
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
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: healthServer.URL + "/execute",
				},
				Health: healthServer.URL + "/health",
			},
		}
		st := &api.Step{
			ID:   "flow-step",
			Name: "Flow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{goal.ID}},
			},
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(goal))
		assert.NoError(t, env.Engine.RegisterStep(st))

		checker := server.NewHealthChecker(env.Engine, nil)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(st.ID, api.HealthHealthy),
			func() {
				checker.Start()
			},
		)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[st.ID]; ok {
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
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: healthServer.URL + "/execute",
				},
				Health: healthServer.URL + "/health",
			},
		}
		st := &api.Step{
			ID:   "bad-flow-step",
			Name: "Bad Flow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{goal.ID}},
			},
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(goal))
		assert.NoError(t, env.Engine.RegisterStep(st))

		checker := server.NewHealthChecker(env.Engine, nil)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(st.ID, api.HealthUnhealthy),
			func() {
				checker.Start()
			},
		)

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[st.ID]; ok {
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
		st := &api.Step{
			ID:   "unhealthy-step",
			Name: "Unhealthy Step",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: healthServer.URL + "/execute",
				},
				Health: healthServer.URL + "/health",
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		checker := server.NewHealthChecker(env.Engine, nil)
		defer checker.Stop()

		env.WaitFor(wait.StepHealthChanged(st.ID, api.HealthUnhealthy), func() {
			checker.Start()
		})

		cluster, err := env.Engine.GetClusterState()
		assert.NoError(t, err)
		for _, node := range cluster.Nodes {
			if h, ok := node.Health[st.ID]; ok {
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
			st := &api.Step{
				ID:   api.StepID("multi-health-" + sfx),
				Name: "Multi Health Step",
				Type: api.StepTypeService,
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{
						Endpoint: healthServer.URL + "/execute",
					},
					Health: healthServer.URL + "/health",
				},
			}
			assert.NoError(t, env.Engine.RegisterStep(st))
		}

		checker := server.NewHealthChecker(env.Engine, nil)
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

func TestAgentHealth(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		want   api.HealthStatus
		reason string
	}{
		{
			name:   "healthy",
			status: http.StatusOK,
			want:   api.HealthHealthy,
		},
		{
			name:   "error status",
			status: http.StatusInternalServerError,
			want:   api.HealthUnhealthy,
			reason: "HTTP 500",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			worker := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					assert.Equal(t, builtins.AgentHealthPath, r.URL.Path)
					w.WriteHeader(tc.status)
				}),
			)
			defer worker.Close()

			h := checkAgentHealth(t, builtins.AgentWorkers{"anthropic": worker.URL}, tc.want)
			assert.Equal(t, tc.want, h.Status)
			assert.Contains(t, h.Error, tc.reason)
			assert.EqualValues(t, 1, calls.Load())
		})
	}
}

func TestAgentHealthNoWorker(t *testing.T) {
	h := checkAgentHealth(t, builtins.AgentWorkers{}, api.HealthUnhealthy)
	assert.Equal(t, api.HealthUnhealthy, h.Status)
	assert.Contains(t, h.Error, builtins.ErrNoAgentWorker.Error())
}

func TestAgentHealthUnreachable(t *testing.T) {
	h := checkAgentHealth(t,
		builtins.AgentWorkers{"anthropic": helpers.AgentWorkerURL}, api.HealthUnhealthy)
	assert.Equal(t, api.HealthUnhealthy, h.Status)
}

func checkAgentHealth(
	t *testing.T, workers builtins.AgentWorkers, want api.HealthStatus,
) api.HealthState {
	t.Helper()
	var res api.HealthState
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := &api.Step{
			ID:   agentStepID,
			Name: "Agent Step",
			Type: api.StepTypeAgent,
			Agent: &api.AgentConfig{
				Provider:  "anthropic",
				Model:     "claude-sonnet-5-5",
				Prologue:  "# Score",
				MCP:       []api.MCPServer{{Server: "crm"}},
				MaxTurns:  1,
				MaxTokens: 1,
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))
		second := st.Copy()
		second.ID = secondAgentStepID
		assert.NoError(t, env.Engine.RegisterStep(second))

		checker := server.NewHealthChecker(env.Engine, workers)
		defer checker.Stop()
		env.WaitForCount(2, wait.And(
			wait.Type(api.EventTypeStepHealthChanged),
			wait.PredicateFilter(func(ev api.StepHealthChangedEvent) bool {
				return ev.Status == want && (ev.StepID == st.ID || ev.StepID == second.ID)
			}),
		), checker.Start)

		h, err := env.Engine.GetStepHealth(st.ID)
		assert.NoError(t, err)
		res = h
	})
	return res
}
