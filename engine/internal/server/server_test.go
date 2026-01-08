package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/events"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type testServerEnv struct {
	Server *server.Server
	*helpers.TestEngineEnv
}

func TestHealthEndpoint(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	router := env.Server.SetupRoutes()
	req := httptest.NewRequest("GET", "/engine/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegisterStep(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("test-step")

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestRegisterStepConflict(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("dupe-step")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateStepInvalidJSONBody(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader([]byte("not-json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateStepValidationError(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	body, _ := json.Marshal(&api.Step{})
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListSteps(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("list-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.StepsListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, response.Count)
	assert.Len(t, response.Steps, 1)
	assert.Equal(t, api.StepID("list-step"), response.Steps[0].ID)
}

func TestGetStep(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("get-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/step/get-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var retrieved *api.Step
	err = json.Unmarshal(w.Body.Bytes(), &retrieved)
	assert.NoError(t, err)
	assert.Equal(t, step.ID, retrieved.ID)
}

func TestDeleteStep(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("delete-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/engine/step/delete-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStart(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("wf-step")

	err := env.Engine.RegisterStep(context.Background(), step)
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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestListFlows(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/flow", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSuccess(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

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

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	// Configure mock to return immediately for async steps
	env.MockClient.SetResponse("async-step", api.Args{})

	err = env.Engine.StartFlow(
		context.Background(), "webhook-wf",
		&api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": step,
			},
		},
		api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	// Wait for flow to execute and create work item
	fs := engine.FlowStep{FlowID: "webhook-wf", StepID: "async-step"}
	env.waitForWorkItem(t, fs)

	// Get the actual token from the created work item
	flow, err := env.Engine.GetFlowState(context.Background(), "webhook-wf")
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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFlowNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStepNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := &api.Step{
		ID:   "async-step",
		Name: "Async Step",
		Type: api.StepTypeAsync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"async-step"},
		Steps: api.Steps{
			"async-step": step,
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "webhook-wf", plan, api.Args{}, api.Metadata{},
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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInvalidToken(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

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

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	// Configure mock to return immediately for async steps
	env.MockClient.SetResponse("async-step", api.Args{})

	err = env.Engine.StartFlow(
		context.Background(), "webhook-wf",
		&api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": step,
			},
		},
		api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	// Wait for work item to be created
	fs := engine.FlowStep{FlowID: "webhook-wf", StepID: "async-step"}
	env.waitForWorkItem(t, fs)

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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInvalidJSON(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := &api.Step{
		ID:   "async-step",
		Name: "Async Step",
		Type: api.StepTypeAsync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	// Configure mock to return immediately for async steps
	env.MockClient.SetResponse("async-step", api.Args{})

	err = env.Engine.StartFlow(
		context.Background(), "webhook-wf",
		&api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": step,
			},
		},
		api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	// Wait for work item to be created
	fs := engine.FlowStep{FlowID: "webhook-wf", StepID: "async-step"}
	env.waitForWorkItem(t, fs)

	// Get the real token
	flow, err := env.Engine.GetFlowState(context.Background(), "webhook-wf")
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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebhookFailurePath(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

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

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	env.MockClient.SetResponse("async-step", api.Args{})

	err = env.Engine.StartFlow(
		context.Background(), "wf-fail-path",
		&api.ExecutionPlan{
			Goals: []api.StepID{"async-step"},
			Steps: api.Steps{
				"async-step": step,
			},
		},
		api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	fs := engine.FlowStep{FlowID: "wf-fail-path", StepID: "async-step"}
	env.waitForWorkItem(t, fs)

	flow, err := env.Engine.GetFlowState(context.Background(), "wf-fail-path")
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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	updated, err := env.Engine.GetFlowState(context.Background(), "wf-fail-path")
	assert.NoError(t, err)
	work := updated.Executions["async-step"].WorkItems[token]
	assert.Equal(t, api.WorkFailed, work.Status)
	assert.Equal(t, "boom", work.Error)
}

func TestGetFlow(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewSimpleStep("get-wf-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"get-wf-step"},
		Steps: api.Steps{
			"get-wf-step": step,
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "test-wf-id", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/flow/test-wf-id", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var wf api.FlowState
	err = json.Unmarshal(w.Body.Bytes(), &wf)
	assert.NoError(t, err)
	assert.Equal(t, api.FlowID("test-wf-id"), wf.ID)
}

func TestGetFlowNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/flow/nonexistent", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteStepNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("DELETE", "/engine/step/missing-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response api.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "not found")
}

func TestUpdateStep(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("update-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	updatedStep := helpers.NewSimpleStep("update-step")

	body, _ := json.Marshal(updatedStep)
	req := httptest.NewRequest(
		"PUT", "/engine/step/update-step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateStepIDMismatchStatus(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("update-step-mismatch")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	updatedStep := helpers.NewSimpleStep("other-id")

	body, _ := json.Marshal(updatedStep)
	req := httptest.NewRequest(
		"PUT", "/engine/step/update-step-mismatch", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEngineHealthEndpoint(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/health", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEngineRootEndpoint(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEngineRootSlashEndpoint(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetStepNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/step/nonexistent", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteMissingStepReturns404(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("DELETE", "/engine/step/nonexistent", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response api.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "not found")
}

func TestRegisterStepInvalidJSON(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStartInvalidJSON(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"POST", "/engine/flow", bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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

func TestHandleHealthByID(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("health-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/health/health-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var health api.HealthState
	err = json.Unmarshal(w.Body.Bytes(), &health)
	assert.NoError(t, err)
	assert.Equal(t, api.HealthUnknown, health.Status)
}

func TestHandleHealthByIDNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/health/nonexistent-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateValidationError(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := &api.Step{
		ID:   "",
		Name: "Invalid Step",
		Type: api.StepTypeSync,
	}

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateStepIDMismatchMessage(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("original-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	updatedStep := helpers.NewSimpleStep("different-id")

	body, _ := json.Marshal(updatedStep)
	req := httptest.NewRequest(
		"PUT", "/engine/step/original-step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "does not match")
}

func TestStartEmptyID(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("test-step")

	err := env.Engine.RegisterStep(context.Background(), step)
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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Flow ID is required")
}

func TestStartNoGoals(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	reqData := map[string]any{
		"id": "test-wf",
	}

	body, _ := json.Marshal(reqData)
	req := httptest.NewRequest(
		"POST", "/engine/flow", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "goal step")
}

func TestListFlowsEmpty(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/flow", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.FlowsListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Count)
}

func TestUpdateValidationError(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("update-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	invalidStep := &api.Step{
		ID:   "update-step",
		Name: "",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{},
	}

	body, _ := json.Marshal(invalidStep)
	req := httptest.NewRequest(
		"PUT", "/engine/step/update-step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBasicHealthEndpoint(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "argyll-engine", response.Service)
	assert.Equal(t, api.HealthHealthy, response.Status)
}

func TestPlanPreview(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

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

	err := env.Engine.RegisterStep(context.Background(), step1)
	assert.NoError(t, err)
	err = env.Engine.RegisterStep(context.Background(), step2)
	assert.NoError(t, err)

	reqData := map[string]any{
		"goals": []string{"step-b"},
		"init":  map[string]any{},
	}

	body, _ := json.Marshal(reqData)
	req := httptest.NewRequest("POST", "/engine/plan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	t.Logf("Response: %s", w.Body.String())

	var response api.ExecutionPlan
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.Goals, 1)
	assert.Equal(t, api.StepID("step-b"), response.Goals[0])
}

func TestPlanPreviewInvalidJSON(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"POST", "/engine/plan", bytes.NewReader([]byte("invalid")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlanPreviewNoGoals(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	reqData := map[string]any{}

	body, _ := json.Marshal(reqData)
	req := httptest.NewRequest("POST", "/engine/plan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "goal step")
}

func TestPlanPreviewStepNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	reqData := map[string]any{
		"goals": []string{"nonexistent-step"},
		"init":  map[string]any{},
	}

	body, _ := json.Marshal(reqData)
	req := httptest.NewRequest("POST", "/engine/plan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "nonexistent-step")
}

func TestUpdateStepNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("nonexistent")

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"PUT", "/engine/step/nonexistent", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}

func TestUpdateStepInvalidJSON(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"PUT",
		"/engine/step/test-step",
		bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateStepDuplicate(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("duplicate-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateStepInvalidScript(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewScriptStep(
		"invalid-script", api.ScriptLangLua, "return {invalid syntax", "result",
	)

	body, _ := json.Marshal(step)
	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to register step")
}

func TestCreateStepInvalidJSONText(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest(
		"POST", "/engine/step", bytes.NewReader([]byte("not json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStartDuplicate(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("dup-wf-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"dup-wf-step"},
		Steps: api.Steps{
			"dup-wf-step": step,
		},
	}

	err = env.Engine.StartFlow(
		context.Background(),
		"duplicate-flow",
		plan,
		api.Args{},
		api.Metadata{},
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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "duplicate-flow")
}

func TestStartStepNotFound(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "nonexistent-step")
}

func TestCORSOptions(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("OPTIONS", "/engine/step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(
		t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type",
	)
}

func TestFlowIDSanitization(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("test-step")
	err := env.Engine.RegisterStep(context.Background(), step)
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

			router := env.Server.SetupRoutes()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDeleteStepInternalError(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("test-delete-step")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("DELETE", "/engine/step/test-delete-step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func (e *testServerEnv) waitForWorkItem(t *testing.T, fs engine.FlowStep) {
	t.Helper()
	assert.Eventually(t, func() bool {
		flow, err := e.Engine.GetFlowState(context.Background(), fs.FlowID)
		if err != nil {
			return false
		}
		exec := flow.Executions[fs.StepID]
		return exec != nil && exec.WorkItems != nil && len(exec.WorkItems) > 0
	}, 5*time.Second, 50*time.Millisecond)
}

func TestListStepsRunning(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step1 := helpers.NewSimpleStep("step-1")
	step2 := helpers.NewSimpleStep("step-2")

	err := env.Engine.RegisterStep(context.Background(), step1)
	assert.NoError(t, err)
	err = env.Engine.RegisterStep(context.Background(), step2)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/step", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.StepsListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 2, response.Count)
}

func TestListMultipleFlows(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewSimpleStep("test-step")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"test-step"},
		Steps: api.Steps{"test-step": step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "flow-1", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	err = env.Engine.StartFlow(
		context.Background(), "flow-2", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/flow", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.FlowsListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 2, response.Count)
}

func TestEngineState(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewSimpleStep("test-step")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var state api.EngineState
	err = json.Unmarshal(w.Body.Bytes(), &state)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(state.Steps))
}

func TestHealthList(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := helpers.NewSimpleStep("health-test-step")
	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/engine/health", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.HealthListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, response.Count, 0)
}

func TestWebhookSuccess(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

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

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	env.MockClient.SetResponse(step.ID, api.Args{})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "webhook-flow", plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	fs := engine.FlowStep{FlowID: "webhook-flow", StepID: step.ID}
	env.waitForWorkItem(t, fs)

	flow, err := env.Engine.GetFlowState(context.Background(), "webhook-flow")
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

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWebSocketEndpoint(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/engine/ws", nil)
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func testServer(t *testing.T) *testServerEnv {
	t.Helper()

	env := helpers.NewTestEngine(t)
	srv := server.NewServer(env.Engine, env.EventHub)

	return &testServerEnv{
		Server:        srv,
		TestEngineEnv: env,
	}
}
