package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
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

func TestEngineHealth(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		router := testEnv.Server.SetupRoutes()
		req := httptest.NewRequest("GET", "/engine/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestStartFlow(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		step := helpers.NewSimpleStep("wf-step")

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		reqBody := api.CreateFlowRequest{
			ID:    "test-flow",
			Goals: []api.StepID{"wf-step"},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})
}

func TestListFlows(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
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
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := &api.Step{
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

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		// Configure mock to return immediately for async steps
		testEnv.MockClient.SetResponse("async-step", api.Args{})

		err = testEnv.Engine.StartFlow("webhook-wf", &api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": step,
			},
		}, api.Args{}, api.Metadata{})
		assert.NoError(t, err)

		// Wait for flow to execute and create work item
		fs := engine.FlowStep{FlowID: "webhook-wf", StepID: "async-step"}
		testEnv.waitForWorkItem(t, fs)

		// Get the actual token from the created work item
		flow, err := testEnv.Engine.GetFlowState("webhook-wf")
		assert.NoError(t, err)

		exec := flow.Executions["async-step"]
		assert.NotNil(t, exec)
		assert.NotNil(t, exec.WorkItems)
		assert.Len(t, exec.WorkItems, 1)

		var token api.Token
		for t := range exec.WorkItems {
			token = t
			break
		}

		// Now call webhook with the real token
		result := api.StepResult{
			Success: true,
			Outputs: api.Args{"result": "completed"},
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/webhook-wf/async-step/"+string(token),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHookFlowNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		result := api.StepResult{
			Success: true,
			Outputs: api.Args{"result": "completed"},
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/nonexistent-wf/step-id/token",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookStepNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := &api.Step{
			ID:   "async-step",
			Name: "Async Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": step,
			},
		}

		err = testEnv.Engine.StartFlow(
			"webhook-wf", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		result := api.StepResult{
			Success: true,
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/webhook-wf/nonexistent-step/token",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookInvalidToken(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := &api.Step{
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

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		// Configure mock to return immediately for async steps
		testEnv.MockClient.SetResponse("async-step", api.Args{})

		err = testEnv.Engine.StartFlow("webhook-wf", &api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": step,
			},
		}, api.Args{}, api.Metadata{})
		assert.NoError(t, err)

		// Wait for work item to be created
		fs := engine.FlowStep{FlowID: "webhook-wf", StepID: "async-step"}
		testEnv.waitForWorkItem(t, fs)

		// Try with wrong token
		result := api.StepResult{
			Success: true,
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/webhook-wf/async-step/wrong-token",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookInvalidJSONRoute(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := &api.Step{
			ID:   "async-step",
			Name: "Async Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		// Configure mock to return immediately for async steps
		testEnv.MockClient.SetResponse("async-step", api.Args{})

		err = testEnv.Engine.StartFlow("webhook-wf", &api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": step,
			},
		}, api.Args{}, api.Metadata{})
		assert.NoError(t, err)

		// Wait for work item to be created
		fs := engine.FlowStep{FlowID: "webhook-wf", StepID: "async-step"}
		testEnv.waitForWorkItem(t, fs)

		// Get the real token
		flow, err := testEnv.Engine.GetFlowState("webhook-wf")
		assert.NoError(t, err)

		exec := flow.Executions["async-step"]
		assert.NotNil(t, exec)
		assert.NotNil(t, exec.WorkItems)

		var token api.Token
		for t := range exec.WorkItems {
			token = t
			break
		}

		// Send invalid JSON with real token
		req := httptest.NewRequest("POST",
			"/webhook/webhook-wf/async-step/"+string(token),
			bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookFailurePath(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := &api.Step{
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

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		testEnv.MockClient.SetResponse("async-step", api.Args{})

		err = testEnv.Engine.StartFlow("wf-fail-path", &api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": step,
			},
		}, api.Args{}, api.Metadata{})
		assert.NoError(t, err)

		fs := engine.FlowStep{FlowID: "wf-fail-path", StepID: "async-step"}
		testEnv.waitForWorkItem(t, fs)

		flow, err := testEnv.Engine.GetFlowState("wf-fail-path")
		assert.NoError(t, err)

		var token api.Token
		for tkn := range flow.Executions["async-step"].WorkItems {
			token = tkn
			break
		}

		result := api.StepResult{
			Success: false,
			Error:   "boom",
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/wf-fail-path/async-step/"+string(token),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		updated, err := testEnv.Engine.GetFlowState("wf-fail-path")
		assert.NoError(t, err)
		work := updated.Executions["async-step"].WorkItems[token]
		assert.Equal(t, api.WorkFailed, work.Status)
		assert.Equal(t, "boom", work.Error)
	})
}

func TestGetFlow(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := helpers.NewSimpleStep("get-wf-step")

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"get-wf-step"},
			Steps: api.Steps{
				"get-wf-step": step,
			},
		}

		err = testEnv.Engine.StartFlow(
			"test-wf-id", plan, api.Args{}, api.Metadata{},
		)
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

func TestGetFlowNotFound(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/flow/nonexistent", nil)
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
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestFilterEngineEvents(t *testing.T) {
	sub := &api.ClientSubscription{
		AggregateID: []string{"engine"},
	}

	filter := server.BuildFilter(sub)
	assert.NotNil(t, filter)

	engineEvent := &timebox.Event{
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeStepRegistered),
	}
	assert.True(t, filter(engineEvent))

	flowEvent := &timebox.Event{
		AggregateID: timebox.AggregateID{
			timebox.ID("flow"), timebox.ID("flow-123"),
		},
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}
	assert.False(t, filter(flowEvent))
}

func TestFilterEventTypes(t *testing.T) {
	sub := &api.ClientSubscription{
		EventTypes: []api.EventType{
			api.EventTypeStepRegistered,
			api.EventTypeFlowStarted,
		},
	}

	filter := server.BuildFilter(sub)

	event1 := &timebox.Event{
		Type: timebox.EventType(api.EventTypeStepRegistered),
	}
	assert.True(t, filter(event1))

	event2 := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}
	assert.True(t, filter(event2))

	event3 := &timebox.Event{
		Type: timebox.EventType(api.EventTypeStepCompleted),
	}
	assert.False(t, filter(event3))
}

func TestFilterFlowID(t *testing.T) {
	sub := &api.ClientSubscription{
		AggregateID: []string{"flow", "test-flow"},
	}

	filter := server.BuildFilter(sub)

	event1 := &timebox.Event{
		AggregateID: timebox.AggregateID{
			timebox.ID("flow"), timebox.ID("test-flow"),
		},
	}
	assert.True(t, filter(event1))

	event2 := &timebox.Event{
		AggregateID: timebox.AggregateID{
			timebox.ID("flow"), timebox.ID("other-flow"),
		},
	}
	assert.False(t, filter(event2))

	event3 := &timebox.Event{AggregateID: events.EngineID}
	assert.False(t, filter(event3))
}

func TestFilterEmpty(t *testing.T) {
	sub := &api.ClientSubscription{}

	filter := server.BuildFilter(sub)

	event := &timebox.Event{
		Type: timebox.EventType(api.EventTypeStepRegistered),
	}
	assert.False(t, filter(event))
}

func TestFilterCombinedUsesAndLogic(t *testing.T) {
	sub := &api.ClientSubscription{
		AggregateID: []string{"engine"},
		EventTypes: []api.EventType{
			api.EventTypeStepRegistered,
		},
	}

	filter := server.BuildFilter(sub)

	matchingEvent := &timebox.Event{
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeStepRegistered),
	}
	assert.True(t, filter(matchingEvent))

	wrongTypeEvent := &timebox.Event{
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeFlowStarted),
	}
	assert.False(t, filter(wrongTypeEvent))

	wrongAggregateEvent := &timebox.Event{
		AggregateID: timebox.AggregateID{
			timebox.ID("flow"), timebox.ID("flow-123"),
		},
		Type: timebox.EventType(api.EventTypeStepRegistered),
	}
	assert.False(t, filter(wrongAggregateEvent))
}

func TestEngineHealthByID(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		step := helpers.NewSimpleStep("health-step")

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/engine/health/health-step", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var health api.HealthState
		err = json.Unmarshal(w.Body.Bytes(), &health)
		assert.NoError(t, err)
		assert.Equal(t, api.HealthUnknown, health.Status)
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
		step := helpers.NewSimpleStep("test-step")

		err := testEnv.Engine.RegisterStep(step)
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
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Flow ID is required")
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
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "goal step")
	})
}

func TestListFlowsEmpty(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest("GET", "/engine/flow", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.FlowsListResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 0, response.Count)
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
	})
}

func TestPlanPreview(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		step1 := &api.Step{
			ID:   "step-a",
			Name: "Step A",
			Type: api.StepTypeSync,
			Attributes: api.AttributeSpecs{
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
				"value":  {Role: api.RoleOutput, Type: api.TypeString},
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
		req.Header.Set("Content-Type", "application/json")
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
	})
}

func TestPlanPreviewInvalidJSON(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		req := httptest.NewRequest(
			"POST", "/engine/plan", bytes.NewReader([]byte("invalid")),
		)
		req.Header.Set("Content-Type", "application/json")
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
		req.Header.Set("Content-Type", "application/json")
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
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "nonexistent-step")
	})
}

func TestStartFlowDuplicate(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		step := helpers.NewSimpleStep("dup-wf-step")

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"dup-wf-step"},
			Steps: api.Steps{
				"dup-wf-step": step,
			},
		}

		err = testEnv.Engine.StartFlow(
			"duplicate-flow", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		reqBody := api.CreateFlowRequest{
			ID:    "duplicate-flow",
			Goals: []api.StepID{"dup-wf-step"},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(
			"POST", "/engine/flow", bytes.NewReader(body),
		)
		req.Header.Set("Content-Type", "application/json")
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
		req.Header.Set("Content-Type", "application/json")
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
		step := helpers.NewSimpleStep("test-step")
		err := testEnv.Engine.RegisterStep(step)
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
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				router := testEnv.Server.SetupRoutes()
				router.ServeHTTP(w, req)

				assert.Equal(t, tt.expectedStatus, w.Code)
			})
		}
	})
}

func (e *testServerEnv) waitForWorkItem(t *testing.T, fs engine.FlowStep) {
	t.Helper()
	assert.Eventually(t, func() bool {
		flow, err := e.Engine.GetFlowState(fs.FlowID)
		if err != nil {
			return false
		}
		exec := flow.Executions[fs.StepID]
		return exec != nil && exec.WorkItems != nil && len(exec.WorkItems) > 0
	}, 5*time.Second, 50*time.Millisecond)
}

func TestListFlowsMultiple(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := helpers.NewSimpleStep("test-step")
		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"test-step"},
			Steps: api.Steps{"test-step": step},
		}

		err = testEnv.Engine.StartFlow(
			"flow-1", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		err = testEnv.Engine.StartFlow(
			"flow-2", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		helpers.WaitForFlowActivated(
			t, testEnv.EventHub, []api.FlowID{"flow-1", "flow-2"},
			5*time.Second,
		)

		req := httptest.NewRequest("GET", "/engine/flow", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.FlowsListResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, 2, response.Count)
	})
}

func TestEngineState(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := helpers.NewSimpleStep("test-step")
		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/engine", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var state api.EngineState
		err = json.Unmarshal(w.Body.Bytes(), &state)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(state.Steps))
	})
}

func TestHealthList(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := helpers.NewSimpleStep("health-test-step")
		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		req := httptest.NewRequest("GET", "/engine/health", nil)
		w := httptest.NewRecorder()

		router := testEnv.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.HealthListResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, response.Count, 0)
	})
}

func TestHookSuccessRoute(t *testing.T) {
	withTestServerEnv(t, func(testEnv *testServerEnv) {
		testEnv.Engine.Start()
		defer func() { _ = testEnv.Engine.Stop() }()

		step := &api.Step{
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

		err := testEnv.Engine.RegisterStep(step)
		assert.NoError(t, err)

		testEnv.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err = testEnv.Engine.StartFlow(
			"webhook-flow", plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		fs := engine.FlowStep{FlowID: "webhook-flow", StepID: step.ID}
		testEnv.waitForWorkItem(t, fs)

		flow, err := testEnv.Engine.GetFlowState("webhook-flow")
		assert.NoError(t, err)

		exec := flow.Executions[step.ID]
		assert.NotNil(t, exec)
		assert.NotEmpty(t, exec.WorkItems)

		var token api.Token
		for t := range exec.WorkItems {
			token = t
			break
		}

		result := api.StepResult{
			Success: true,
			Outputs: api.Args{"output": "value"},
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/webhook-flow/"+string(step.ID)+"/"+string(token),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
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
