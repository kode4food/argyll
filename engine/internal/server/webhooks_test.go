package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestWebhookWithInvalidWorkItem(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := &api.Step{
		ID:   "invalid-work-step",
		Name: "Invalid Work Step",
		Type: api.StepTypeAsync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	assert.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "invalid-work-flow", plan, api.Args{},
		api.Metadata{},
	)
	assert.NoError(t, err)

	result := api.StepResult{Success: true}
	body, _ := json.Marshal(result)
	req := httptest.NewRequest("POST",
		"/webhook/invalid-work-flow/"+string(step.ID)+"/fake-token",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestWebhookCompleteTwice(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := &api.Step{
		ID:   "double-complete",
		Name: "Double Complete",
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
		context.Background(), "double-complete-flow", plan, api.Args{},
		api.Metadata{},
	)
	assert.NoError(t, err)

	fs := engine.FlowStep{FlowID: "double-complete-flow", StepID: step.ID}
	env.waitForWorkItem(fs)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "double-complete-flow",
	)
	assert.NoError(t, err)

	var token api.Token
	for t := range flow.Executions[step.ID].WorkItems {
		token = t
		break
	}

	result := api.StepResult{
		Success: true,
		Outputs: api.Args{"output": "value1"},
	}

	body, _ := json.Marshal(result)
	req := httptest.NewRequest("POST",
		"/webhook/double-complete-flow/"+string(step.ID)+"/"+string(token),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	time.Sleep(100 * time.Millisecond)

	result = api.StepResult{
		Success: true,
		Outputs: api.Args{"output": "value2"},
	}

	body, _ = json.Marshal(result)
	req = httptest.NewRequest("POST",
		"/webhook/double-complete-flow/"+string(step.ID)+"/"+string(token),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)
}

func TestWebhookSuccessPath(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := &api.Step{
		ID:   "webhook-success",
		Name: "Webhook Success",
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

	env.MockClient.SetResponse(step.ID, api.Args{})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "webhook-success-flow", plan, api.Args{},
		api.Metadata{},
	)
	assert.NoError(t, err)

	fs := engine.FlowStep{FlowID: "webhook-success-flow", StepID: step.ID}
	env.waitForWorkItem(fs)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "webhook-success-flow",
	)
	assert.NoError(t, err)

	var token api.Token
	for t := range flow.Executions[step.ID].WorkItems {
		token = t
		break
	}

	result := api.StepResult{
		Success: true,
		Outputs: api.Args{"result": "success"},
	}

	body, _ := json.Marshal(result)
	req := httptest.NewRequest("POST",
		"/webhook/webhook-success-flow/"+string(step.ID)+"/"+string(token),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestWebhookWorkFailure(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := &api.Step{
		ID:   "webhook-fail",
		Name: "Webhook Fail",
		Type: api.StepTypeAsync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
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
		context.Background(), "webhook-fail-flow", plan, api.Args{},
		api.Metadata{},
	)
	assert.NoError(t, err)

	fs := engine.FlowStep{FlowID: "webhook-fail-flow", StepID: step.ID}
	env.waitForWorkItem(fs)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "webhook-fail-flow",
	)
	assert.NoError(t, err)

	var token api.Token
	for t := range flow.Executions[step.ID].WorkItems {
		token = t
		break
	}

	result := api.StepResult{
		Success: false,
		Error:   "step failed",
	}

	body, _ := json.Marshal(result)
	req := httptest.NewRequest("POST",
		"/webhook/webhook-fail-flow/"+string(step.ID)+"/"+string(token),
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestWebhookInvalidJSON(t *testing.T) {
	env := testServer(t)
	defer env.Cleanup()

	env.Engine.Start()
	defer func() { _ = env.Engine.Stop() }()

	step := &api.Step{
		ID:   "webhook-badjson",
		Name: "Webhook Bad JSON",
		Type: api.StepTypeAsync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
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
		context.Background(), "webhook-badjson-flow", plan, api.Args{},
		api.Metadata{},
	)
	assert.NoError(t, err)

	fs := engine.FlowStep{FlowID: "webhook-badjson-flow", StepID: step.ID}
	env.waitForWorkItem(fs)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "webhook-badjson-flow",
	)
	assert.NoError(t, err)

	var token api.Token
	for t := range flow.Executions[step.ID].WorkItems {
		token = t
		break
	}

	req := httptest.NewRequest("POST",
		"/webhook/webhook-badjson-flow/"+string(step.ID)+"/"+string(token),
		bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router := env.Server.SetupRoutes()
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}
