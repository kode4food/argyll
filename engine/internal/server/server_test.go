package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/kode4food/timebox/raft"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type testServerEnv struct {
	Server *server.Server
	*helpers.TestEngineEnv
}

func withTestServerEnv(t *testing.T, fn func(*testServerEnv)) {
	t.Helper()
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		fn(&testServerEnv{
			Server:        server.NewServer(env.Engine, env.EventHub),
			TestEngineEnv: env,
		})
	})
}

func withServerEnvConfig(
	t *testing.T, makeConfig func(*config.Config) *config.Config,
	fn func(*testServerEnv),
) {
	t.Helper()
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cfg := makeConfig(env.Config)
		eng, err := engine.New(cfg, env.Dependencies())
		assert.NoError(t, err)
		if eng == nil {
			return
		}
		defer func() { _ = eng.Stop() }()

		next := *env
		next.Engine = eng
		next.Config = cfg
		fn(&testServerEnv{
			Server:        server.NewServer(eng, env.EventHub),
			TestEngineEnv: &next,
		})
	})
}

func clusterConfigWithPeer(base *config.Config) *config.Config {
	cfg := util.MutableCopy(base)
	cfg.Raft.Servers = append([]raft.Server{}, base.Raft.Servers...)
	cfg.Raft.Servers = append(cfg.Raft.Servers,
		raft.Server{ID: "node-2", Address: "127.0.0.1:9702"},
	)
	return cfg
}

func peerOnlyConfig(base *config.Config) *config.Config {
	cfg := util.MutableCopy(base)
	cfg.Raft.LocalID = "node-2"
	cfg.Raft.Servers = []raft.Server{
		{ID: "node-2", Address: "127.0.0.1:9702"},
	}
	return cfg
}

func TestHealthIncludesCustomStatus(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		srv := server.NewServer(
			env.Engine,
			env.EventHub,
			func() map[string]any {
				return map[string]any{
					"backend": map[string]any{
						"kind": "test",
					},
				}
			},
		)
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router := srv.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.HealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		if assert.Contains(t, response.Details, "backend") {
			backend, ok := response.Details["backend"].(map[string]any)
			if assert.True(t, ok) {
				assert.Equal(t, "test", backend["kind"])
			}
		}
	})
}

func TestHealthIncludesBackendStatus(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		srv := server.NewServer(
			env.Engine,
			env.EventHub,
			func() map[string]any {
				return map[string]any{
					"backend": map[string]any{
						"type":      "raft",
						"state":     "leader",
						"leader_id": "node-1",
					},
				}
			},
		)
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router := srv.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.HealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		if assert.Contains(t, response.Details, "backend") {
			backend, ok := response.Details["backend"].(map[string]any)
			if assert.True(t, ok) {
				assert.Equal(t, "raft", backend["type"])
				assert.Equal(t, "leader", backend["state"])
				assert.Equal(t, "node-1", backend["leader_id"])
			}
		}
	})
}

func TestHealthLeader(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		srv := server.NewServer(
			env.Engine,
			env.EventHub,
			func() map[string]any {
				return map[string]any{
					"backend": map[string]any{
						"type":  "raft",
						"state": raft.StateLeader,
					},
				}
			},
		)
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router := srv.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "leader", w.Header().Get("X-Argyll-Raft-State"))
	})
}

func TestHealthFollower(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		srv := server.NewServer(
			env.Engine,
			env.EventHub,
			func() map[string]any {
				return map[string]any{
					"backend": map[string]any{
						"type":  "raft",
						"state": raft.StateFollower,
					},
				}
			},
		)
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router := srv.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "follower", w.Header().Get("X-Argyll-Raft-State"))
	})
}

func TestHealthUnknown(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "unknown", w.Header().Get("X-Argyll-Raft-State"))
	})
}

func TestRaftStatusNil(t *testing.T) {
	assert.Nil(t, server.NewRaftStatusProvider(nil)())
}

func TestRaftStatus(t *testing.T) {
	cfg := config.NewDefaultConfig()
	addr := availableAddress(t)
	cfg.Raft.LocalID = "node-" + strconv.Itoa(availablePort(t))
	cfg.Raft.Address = addr
	cfg.Raft.DataDir = t.TempDir()
	cfg.Raft.Servers = []raft.Server{
		{ID: cfg.Raft.LocalID, Address: addr},
	}

	p, err := raft.NewPersistence(cfg.Raft)
	assert.NoError(t, err)
	if p != nil {
		defer func() { _ = p.Close() }()
	}

	st := server.NewRaftStatusProvider(p)()
	if assert.Contains(t, st, "backend") {
		backend, ok := st["backend"].(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, "raft", backend["type"])
			assert.NotEmpty(t, backend["state"])
			assert.Contains(t, backend, "leader_address")
			assert.Contains(t, backend, "leader_id")
		}
	}
}

func availablePort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	addr, ok := ln.Addr().(*net.TCPAddr)
	assert.True(t, ok)
	return addr.Port
}

func availableAddress(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	return ln.Addr().String()
}

func TestEngineHealth(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		router := testEnv.Server.SetupRoutes()
		req := httptest.NewRequest("GET", "/engine/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestEngineClosed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	srv := server.NewServer(env.Engine, env.EventHub)
	env.Cleanup()

	req := httptest.NewRequest("GET", "/engine", nil)
	w := httptest.NewRecorder()

	router := srv.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEngineHealthClosed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	srv := server.NewServer(env.Engine, env.EventHub)
	env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/health", nil)
	w := httptest.NewRecorder()

	router := srv.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestEngineHealthByIDClosed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	srv := server.NewServer(env.Engine, env.EventHub)
	env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/health/step-1", nil)
	w := httptest.NewRecorder()

	router := srv.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStartFlow(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("wf-step")

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		reqBody := api.CreateFlowRequest{
			ID:    "test-flow",
			Goals: []api.StepID{"wf-step"},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestQueryFlows(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader([]byte("{}")),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestListFlows(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		req := httptest.NewRequest("GET", "/engine/flow", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSuccess(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := &api.Step{
			ID:   "async-step",
			Name: "Async Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
		}

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		// Configure mock to return immediately for async steps
		testEnv.MockClient.SetResponse("async-step", api.Args{})

		testEnv.WaitForStepStarted(
			api.FlowStep{
				FlowID: "webhook-wf",
				StepID: "async-step",
			},
			func() {
				err = testEnv.Engine.StartFlow("webhook-wf", &api.ExecutionPlan{
					Goals: []api.StepID{"async-step"},
					Steps: api.Steps{
						"async-step": st,
					},
				})
				assert.NoError(t, err)
			})

		// Get the actual token from the created work item
		fl, err := testEnv.Engine.GetFlowState("webhook-wf")
		assert.NoError(t, err)

		ex := fl.Executions["async-step"]
		assert.NotNil(t, ex)
		assert.NotNil(t, ex.WorkItems)
		assert.Len(t, ex.WorkItems, 1)

		var tkn api.Token
		for t := range ex.WorkItems {
			tkn = t
			break
		}

		// Now call webhook with the real token
		body, _ := json.Marshal(api.Args{"result": "completed"})
		req := httptest.NewRequest("POST",
			"/webhook/webhook-wf/async-step/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHookFlowNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		body, _ := json.Marshal(api.Args{"result": "completed"})
		req := httptest.NewRequest("POST",
			"/webhook/nonexistent-wf/step-id/token",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookStepNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := &api.Step{
			ID:   "async-step",
			Name: "Async Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": st,
			},
		}

		err = testEnv.Engine.StartFlow("webhook-wf", pl)
		assert.NoError(t, err)

		body, _ := json.Marshal(api.Args{})
		req := httptest.NewRequest("POST",
			"/webhook/webhook-wf/nonexistent-step/token",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookInvalidToken(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := &api.Step{
			ID:   "async-step",
			Name: "Async Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
		}

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		// Configure mock to return immediately for async steps
		testEnv.MockClient.SetResponse("async-step", api.Args{})

		testEnv.WaitForStepStarted(
			api.FlowStep{
				FlowID: "webhook-wf",
				StepID: "async-step",
			},
			func() {
				err = testEnv.Engine.StartFlow("webhook-wf", &api.ExecutionPlan{
					Goals: []api.StepID{"async-step"},
					Steps: api.Steps{
						"async-step": st,
					},
				})
				assert.NoError(t, err)
			})

		// Try with wrong token
		body, _ := json.Marshal(api.Args{})
		req := httptest.NewRequest("POST",
			"/webhook/webhook-wf/async-step/wrong-token",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookInvalidJSONRoute(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := &api.Step{
			ID:   "async-step",
			Name: "Async Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		// Configure mock to return immediately for async steps
		testEnv.MockClient.SetResponse("async-step", api.Args{})

		testEnv.WaitForStepStarted(
			api.FlowStep{
				FlowID: "webhook-wf",
				StepID: "async-step",
			},
			func() {
				err = testEnv.Engine.StartFlow("webhook-wf", &api.ExecutionPlan{
					Goals: []api.StepID{"async-step"},
					Steps: api.Steps{
						"async-step": st,
					},
				})
				assert.NoError(t, err)
			})

		// Get the real token
		fl, err := testEnv.Engine.GetFlowState("webhook-wf")
		assert.NoError(t, err)

		ex := fl.Executions["async-step"]
		assert.NotNil(t, ex)
		assert.NotNil(t, ex.WorkItems)

		var tkn api.Token
		for t := range ex.WorkItems {
			tkn = t
			break
		}

		// Send invalid JSON with real token
		req := httptest.NewRequest("POST",
			"/webhook/webhook-wf/async-step/"+string(tkn),
			bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookFailurePath(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := &api.Step{
			ID:   "async-step",
			Name: "Async Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
		}

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		testEnv.MockClient.SetResponse("async-step", api.Args{})

		testEnv.WaitForStepStarted(
			api.FlowStep{
				FlowID: "wf-fail-path",
				StepID: "async-step",
			},
			func() {
				err = testEnv.Engine.StartFlow(
					"wf-fail-path", &api.ExecutionPlan{
						Goals: []api.StepID{"async-step"},
						Steps: api.Steps{
							"async-step": st,
						},
					})
				assert.NoError(t, err)
			})

		fl, err := testEnv.Engine.GetFlowState("wf-fail-path")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range fl.Executions["async-step"].WorkItems {
			tkn = t
			break
		}

		body, _ := json.Marshal(api.NewProblem(
			http.StatusUnprocessableEntity, "boom",
		))
		req := httptest.NewRequest("POST",
			"/webhook/wf-fail-path/async-step/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.ProblemJSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		updated, err := testEnv.Engine.GetFlowState("wf-fail-path")
		assert.NoError(t, err)
		work := updated.Executions["async-step"].WorkItems[tkn]
		assert.Equal(t, api.WorkFailed, work.Status)
		assert.Equal(t, "boom", work.Error)
	})
}

func TestGetFlow(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := helpers.NewSimpleStep("get-wf-step")

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"get-wf-step"},
			Steps: api.Steps{
				"get-wf-step": st,
			},
		}

		err = testEnv.Engine.StartFlow("test-wf-id", pl)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/engine/flow/test-wf-id", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var wf api.FlowState
		err = json.Unmarshal(w.Body.Bytes(), &wf)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowID("test-wf-id"), wf.ID)
	})
}

func TestGetFlowStatus(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := helpers.NewSimpleStep("status-wf-step")

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"status-wf-step"},
			Steps: api.Steps{
				"status-wf-step": st,
			},
		}

		testEnv.WaitForFlowStatus("status-wf-id", func() {
			err = testEnv.Engine.StartFlow("status-wf-id", pl)
			assert.NoError(t, err)
		})

		req := httptest.NewRequest(
			"GET", "/engine/flow/status-wf-id/status", nil,
		)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp api.FlowStatusResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowID("status-wf-id"), resp.ID)
		assert.Equal(t, api.FlowCompleted, resp.Status)
	})
}

func TestGetFlowNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/flow/nonexistent", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestGetFlowStatusNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest(
			"GET", "/engine/flow/nonexistent/status", nil,
		)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestEngineHealthOK(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/health", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestEngine(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestEngineSlash(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestStartFlowInvalidJSON(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader([]byte("invalid json")),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestEngineHealthByID(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("health-step")

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/engine/health/health-step", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var h api.HealthState
		err = json.Unmarshal(w.Body.Bytes(), &h)
		assert.NoError(t, err)
		assert.Equal(t, api.HealthUnknown, h.Status)
	})
}

func TestEngineHealthIncludesClusterNodes(t *testing.T) {
	withServerEnvConfig(t, clusterConfigWithPeer, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("health-step")
		assert.NoError(t, testEnv.Engine.RegisterStep(st))
		assert.NoError(t,
			testEnv.Engine.UpdateStepHealth(st.ID, api.HealthHealthy, ""),
		)

		peer, err := engine.New(
			peerOnlyConfig(testEnv.Config), testEnv.Dependencies(),
		)
		assert.NoError(t, err)
		if peer != nil {
			defer func() { _ = peer.Stop() }()
		}

		assert.NoError(t,
			peer.UpdateStepHealth(st.ID, api.HealthUnhealthy, "peer down"),
		)

		req := httptest.NewRequest("GET", "/engine/health", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.ClusterState
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.Nodes, 2)
		localID := testEnv.Engine.LocalNodeID()
		if assert.Contains(t, response.Nodes, localID) {
			assert.Equal(t,
				api.HealthHealthy,
				response.Nodes[localID].Health[st.ID].Status,
			)
		}
		if assert.Contains(t, response.Nodes, api.NodeID("node-2")) {
			assert.Equal(t,
				api.HealthUnhealthy,
				response.Nodes["node-2"].Health[st.ID].Status,
			)
		}
	})
}

func TestEngineHealthSilentNodes(t *testing.T) {
	withServerEnvConfig(t, clusterConfigWithPeer, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/health", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.ClusterState
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.Nodes, 2)
		assert.Contains(t,
			response.Nodes, api.NodeID(testEnv.Config.Raft.LocalID),
		)
		if assert.Contains(t, response.Nodes, api.NodeID("node-2")) {
			assert.Empty(t, response.Nodes["node-2"].Health)
		}
	})
}

func TestEngineHealthUnknownSteps(t *testing.T) {
	withServerEnvConfig(t, clusterConfigWithPeer, func(testEnv *testServerEnv) {
		healthServer := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)
		defer healthServer.Close()

		stepA := &api.Step{
			ID:   "step-a",
			Name: "Step A",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint:    healthServer.URL + "/execute",
				HealthCheck: healthServer.URL + "/health",
			},
		}
		stepB := helpers.NewSimpleStep("step-b")
		assert.NoError(t, testEnv.Engine.RegisterStep(stepA))
		assert.NoError(t, testEnv.Engine.RegisterStep(stepB))

		peer, err := engine.New(
			peerOnlyConfig(testEnv.Config), testEnv.Dependencies(),
		)
		assert.NoError(t, err)
		if peer != nil {
			defer func() { _ = peer.Stop() }()
		}
		checker := server.NewHealthChecker(peer)
		defer checker.Stop()

		testEnv.WaitFor(
			wait.StepHealthChanged(stepA.ID, api.HealthHealthy),
			func() { checker.Start() },
		)

		req := httptest.NewRequest("GET", "/engine/health", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.ClusterState
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		if assert.Contains(t, response.Nodes, api.NodeID("node-2")) {
			assert.Equal(t,
				api.HealthHealthy,
				response.Nodes["node-2"].Health[stepA.ID].Status,
			)
			assert.Equal(t,
				api.HealthUnknown,
				response.Nodes["node-2"].Health[stepB.ID].Status,
			)
		}
	})
}

func TestEngineHealthScriptNodes(t *testing.T) {
	withServerEnvConfig(t, clusterConfigWithPeer, func(testEnv *testServerEnv) {
		st := &api.Step{
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
		assert.NoError(t, testEnv.Engine.RegisterStep(st))

		peer, err := engine.New(
			peerOnlyConfig(testEnv.Config), testEnv.Dependencies(),
		)
		assert.NoError(t, err)
		if peer != nil {
			defer func() { _ = peer.Stop() }()
		}

		assert.NoError(t, peer.UpdateStepHealth(st.ID, api.HealthUnknown, ""))

		req := httptest.NewRequest("GET", "/engine/health", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.ClusterState
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		localID := testEnv.Engine.LocalNodeID()
		if assert.Contains(t, response.Nodes, localID) {
			assert.Equal(t,
				api.HealthHealthy,
				response.Nodes[localID].Health[st.ID].Status,
			)
		}
		if assert.Contains(t, response.Nodes, api.NodeID("node-2")) {
			assert.Equal(t,
				api.HealthUnknown,
				response.Nodes["node-2"].Health[st.ID].Status,
			)
		}
	})
}

func TestEngineUnknownSteps(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("step-1")
		assert.NoError(t, testEnv.Engine.RegisterStep(st))

		req := httptest.NewRequest("GET", "/engine", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			Health map[api.NodeID]api.NodeState `json:"health"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		nid := api.NodeID(testEnv.Config.Raft.LocalID)
		if assert.Contains(t, response.Health, nid) {
			if assert.Contains(t, response.Health[nid].Health, st.ID) {
				assert.Equal(t,
					api.HealthUnknown,
					response.Health[nid].Health[st.ID].Status,
				)
			}
		}
	})
}

func TestEngineHealthByIDFlow(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		goalA := helpers.NewSimpleStep("goal-a")
		goalB := helpers.NewSimpleStep("goal-b")
		st := &api.Step{
			ID:   "flow-step",
			Name: "Flow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{goalA.ID, goalB.ID},
			},
			Attributes: api.AttributeSpecs{
				"out": {Role: api.RoleOutput},
			},
		}

		assert.NoError(t, testEnv.Engine.RegisterStep(goalA))
		assert.NoError(t, testEnv.Engine.RegisterStep(goalB))
		assert.NoError(t, testEnv.Engine.RegisterStep(st))
		assert.NoError(t,
			testEnv.Engine.UpdateStepHealth(goalA.ID, api.HealthHealthy, ""),
		)
		assert.NoError(t, testEnv.Engine.UpdateStepHealth(
			goalB.ID, api.HealthUnhealthy, "down",
		))

		req := httptest.NewRequest("GET", "/engine/health/flow-step", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var h api.HealthState
		err := json.Unmarshal(w.Body.Bytes(), &h)
		assert.NoError(t, err)
		assert.Equal(t, api.HealthUnhealthy, h.Status)
		assert.Contains(t, h.Error, "goal-b")
	})
}

func TestEngineHealthNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest(
			"GET", "/engine/health/nonexistent-step", nil,
		)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestStartFlowEmptyID(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("test-step")

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		reqData := map[string]any{
			"id":    "",
			"goals": []string{"test-step"},
			"init":  map[string]any{},
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "flow ID empty")
	})
}

func TestStartFlowNoGoals(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqData := map[string]any{
			"id": "test-wf",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "goal step")
	})
}

func TestStartFlowMissingRequiredInputs(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := &api.Step{
			ID:   "required-input-step",
			Name: "Required Input Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
			Attributes: api.AttributeSpecs{
				"customer_id": {Role: api.RoleRequired, Type: api.TypeString},
				"result":      {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		reqBody := api.CreateFlowRequest{
			ID:    "wf-missing-input",
			Goals: []api.StepID{"required-input-step"},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "required inputs")
	})
}

func TestQueryFlowsEmpty(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader([]byte("{}")),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.QueryFlowsResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 0, response.Count)
	})
}

func TestListFlowsEndpoint(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := helpers.NewSimpleStep("list-step")
		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		testEnv.WaitFor(wait.FlowActivated("wf-list"), func() {
			err = testEnv.Engine.StartFlow("wf-list", pl)
			assert.NoError(t, err)
		})

		req := httptest.NewRequest("GET", "/engine/flow", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []*api.QueryFlowsItem
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response)
	})
}

func TestQueryFlowsInvalidStatuses(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqBody := map[string]any{
			"statuses": []string{"nope"},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid statuses")
	})
}

func TestBasicHealthEndpoint(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.HealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "argyll-engine", response.Service)
		assert.Equal(t, api.HealthHealthy, response.Status)
		if assert.Contains(t, response.Details, "websocket") {
			websocket, ok := response.Details["websocket"].(map[string]any)
			if assert.True(t, ok) {
				assert.Equal(t, float64(0), websocket["clients"])
				assert.NotContains(t, websocket, "subscriptions")
			}
		}
		assert.Equal(t, "unknown", w.Header().Get("X-Argyll-Raft-State"))
	})
}

func TestPlanPreview(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		step1 := &api.Step{
			ID:   "step-a",
			Name: "Step A",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"seed":  {Role: api.RoleRequired, Type: api.TypeString},
				"value": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		step2 := &api.Step{
			ID:   "step-b",
			Name: "Step B",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
				"value":  {Role: api.RoleRequired, Type: api.TypeString},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := testEnv.Engine.RegisterStep(step1)
		assert.NoError(t, err)
		err = testEnv.Engine.RegisterStep(step2)
		assert.NoError(t, err)

		reqData := map[string]any{
			"goals": []string{"step-b"},
			"init":  map[string]any{},
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest(
			"POST", "/engine/plan", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		t.Logf("Response: %s", w.Body.String())

		var response api.ExecutionPlan
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.Goals, 1)
		assert.Equal(t, api.StepID("step-b"), response.Goals[0])
		assert.Contains(t, response.Steps, api.StepID("step-a"))
		assert.Contains(t, response.Steps, api.StepID("step-b"))
		assert.Equal(t, []api.Name{"seed"}, response.Required)
	})
}

func TestPlanPreviewInvalidJSON(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest(
			"POST", "/engine/plan", bytes.NewReader([]byte("invalid")),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPlanPreviewNoGoals(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqData := map[string]any{}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest(
			"POST", "/engine/plan", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "goal step")
	})
}

func TestPlanPreviewStepNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqData := map[string]any{
			"goals": []string{"nonexistent-step"},
			"init":  map[string]any{},
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest(
			"POST", "/engine/plan", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "nonexistent-step")
	})
}

func TestStartFlowDuplicate(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("dup-wf-step")

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"dup-wf-step"},
			Steps: api.Steps{
				"dup-wf-step": st,
			},
		}

		err = testEnv.Engine.StartFlow("duplicate-flow", pl)
		assert.NoError(t, err)

		reqBody := api.CreateFlowRequest{
			ID:    "duplicate-flow",
			Goals: []api.StepID{"dup-wf-step"},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "duplicate-flow")
	})
}

func TestStartFlowStepNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqBody := api.CreateFlowRequest{
			ID:    "wf-no-step",
			Goals: []api.StepID{"nonexistent-step"},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "nonexistent-step")
	})
}

func TestCORSOptions(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("OPTIONS", "/engine/step", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t,
			w.Header().Get("Access-Control-Allow-Methods"), "GET",
		)
		assert.Contains(t,
			w.Header().Get("Access-Control-Allow-Headers"), "Content-Type",
		)
	})
}

func TestSanitizeFlowID(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("test-step")
		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		tests := []struct {
			name           string
			flowID         api.FlowID
			expectedStatus int
			shouldSucceed  bool
		}{
			{
				name:           "uppercase_converted_to_lowercase",
				flowID:         "MyFlow-ABC",
				expectedStatus: http.StatusCreated,
				shouldSucceed:  true,
			},
			{
				name:           "spaces_converted_to_dashes",
				flowID:         "my flow test",
				expectedStatus: http.StatusCreated,
				shouldSucceed:  true,
			},
			{
				name:           "special_chars_removed",
				flowID:         "flow@#$%123",
				expectedStatus: http.StatusCreated,
				shouldSucceed:  true,
			},
			{
				name:           "only_special_chars_results_in_error",
				flowID:         "@#$%^&*()",
				expectedStatus: http.StatusBadRequest,
				shouldSucceed:  false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reqBody := api.CreateFlowRequest{
					ID:    tt.flowID,
					Goals: []api.StepID{"test-step"},
				}

				body, _ := json.Marshal(reqBody)
				req := httptest.NewRequest(
					"POST", "/engine/flow", bytes.NewReader(body),
				)
				req.Header.Set("Content-Type", api.JSONContentType)
				w := httptest.NewRecorder()

				router := testEnv.Server.SetupRoutes()
				router.ServeHTTP(w, req)

				assert.Equal(t, tt.expectedStatus, w.Code)
			})
		}
	})
}

func TestQueryFlowsMultiple(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		var err error
		st := helpers.NewSimpleStep("test-step")
		testEnv.WaitFor(wait.EngineEvent(
			api.EventTypeStepRegistered,
		), func() {
			err = testEnv.Engine.RegisterStep(st)
			assert.NoError(t, err)
		})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"test-step"},
			Steps: api.Steps{"test-step": st},
		}

		testEnv.WaitForCount(2,
			wait.FlowActivated("flow-1", "flow-2"), func() {
				err = testEnv.Engine.StartFlow("flow-1", pl)
				assert.NoError(t, err)

				err = testEnv.Engine.StartFlow("flow-2", pl)
				assert.NoError(t, err)
			})

		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader([]byte("{}")),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.QueryFlowsResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 2, response.Count)
	})
}

func TestGetEngine(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := helpers.NewSimpleStep("test-step")
		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/engine", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var state struct {
			Steps api.Steps `json:"steps"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &state)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(state.Steps))
	})
}

func TestHealthList(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := helpers.NewSimpleStep("health-test-step")
		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/engine/health", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.ClusterState
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response.Nodes)
	})
}

func TestHookSuccessRoute(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		assert.NoError(t, testEnv.Engine.Start())
		defer func() { _ = testEnv.Engine.Stop() }()

		st := &api.Step{
			ID:   "webhook-step",
			Name: "Webhook Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
			Attributes: api.AttributeSpecs{
				"output": {Role: api.RoleOutput},
			},
		}

		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		testEnv.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		testEnv.WaitForStepStarted(
			api.FlowStep{
				FlowID: "webhook-flow",
				StepID: st.ID,
			},
			func() {
				err = testEnv.Engine.StartFlow("webhook-flow", pl)
				assert.NoError(t, err)
			})

		fl, err := testEnv.Engine.GetFlowState("webhook-flow")
		assert.NoError(t, err)

		ex := fl.Executions[st.ID]
		assert.NotNil(t, ex)
		assert.NotEmpty(t, ex.WorkItems)

		var tkn api.Token
		for t := range ex.WorkItems {
			tkn = t
			break
		}

		body, _ := json.Marshal(api.Args{"output": "value"})
		req := httptest.NewRequest("POST",
			"/webhook/webhook-flow/"+string(st.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestSocketEndpoint(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/ws", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestQueryFlowsInvalidJSON(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader([]byte("not json")),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestQueryFlowsLimitTooHigh(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqBody := map[string]any{"limit": server.MaxQueryLimit + 1}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Limit must be between")
	})
}

func TestQueryFlowsNegativeLimit(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqBody := map[string]any{"limit": -1}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Limit must be between")
	})
}

func TestQueryFlowsInvalidSort(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqBody := map[string]any{"sort": "invalid-sort-value"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid sort")
	})
}

func TestQueryFlowsInvalidIDPrefix(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqBody := map[string]any{"id_prefix": "bad!prefix"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid ID prefix")
	})
}

func TestQueryFlowsInvalidLabelEmptyKey(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		reqBody := map[string]any{"labels": map[string]string{"": "value"}}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow/query", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid labels")
	})
}

func TestStartFlowIDTooLong(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("test-step-long-id")
		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		longID := api.FlowID(strings.Repeat("a", api.MaxFlowIDLen+1))
		reqBody := api.CreateFlowRequest{
			ID:    longID,
			Goals: []api.StepID{"test-step-long-id"},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "flow ID too long")
	})
}

func TestStartFlowTooManyGoals(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		goals := make([]api.StepID, api.MaxGoalCount+1)
		for i := range goals {
			goals[i] = api.StepID(fmt.Sprintf("step-%d", i))
		}
		reqBody := api.CreateFlowRequest{
			ID:    "too-many-goals",
			Goals: goals,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "too many goals")
	})
}

func TestStartFlowTooManyInitKeys(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("test-step-init-keys")
		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		init := api.InitArgs{}
		for i := range api.MaxInitKeys + 1 {
			init[api.Name(fmt.Sprintf("key-%d", i))] = []any{"value"}
		}
		reqBody := api.CreateFlowRequest{
			ID:    "too-many-init-keys",
			Goals: []api.StepID{"test-step-init-keys"},
			Init:  init,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "too many init keys")
	})
}

func TestStartFlowTooManyLabels(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		st := helpers.NewSimpleStep("test-step-labels")
		err := testEnv.Engine.RegisterStep(st)
		assert.NoError(t, err)

		labels := api.Labels{}
		for i := range api.MaxLabelCount + 1 {
			labels[fmt.Sprintf("label-%d", i)] = "value"
		}
		reqBody := api.CreateFlowRequest{
			ID:     "too-many-labels",
			Goals:  []api.StepID{"test-step-labels"},
			Labels: labels,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "too many labels")
	})
}
