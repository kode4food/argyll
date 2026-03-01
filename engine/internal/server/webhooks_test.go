package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestHookInvalidWorkItem(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		step := &api.Step{
			ID:   "invalid-work-step",
			Name: "Invalid Work Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("invalid-work-flow", plan)
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
	})
}

func TestHookFlowMissing(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		req := httptest.NewRequest("POST",
			"/webhook/missing-flow/step/token",
			bytes.NewReader([]byte(`{"success":true}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookExecutionMissing(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		step := &api.Step{
			ID:   "known-step",
			Name: "Known Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		err = env.Engine.StartFlow("missing-exec-flow", plan)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST",
			"/webhook/missing-exec-flow/unknown-step/token",
			bytes.NewReader([]byte(`{"success":true}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHookCompleteTwice(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		assert.NoError(t, env.Engine.Start())
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

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "double-complete-flow", StepID: step.ID},
			func() {
				err = env.Engine.StartFlow("double-complete-flow", plan)
				assert.NoError(t, err)
			},
		)

		flow, err := env.Engine.GetFlowState("double-complete-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range flow.Executions[step.ID].WorkItems {
			tkn = t
			break
		}

		result := api.StepResult{
			Success: true,
			Outputs: api.Args{"output": "value1"},
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/double-complete-flow/"+string(step.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		router := env.Server.SetupRoutes()
		var w *httptest.ResponseRecorder
		exec := env.WaitForStepStatus("double-complete-flow", step.ID, func() {
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
		})

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, api.StepCompleted, exec.Status)

		// Second webhook call with same token should be rejected (400) due to
		// invalid work state transition
		result = api.StepResult{
			Success: true,
			Outputs: api.Args{"output": "value2"},
		}

		body, _ = json.Marshal(result)
		req = httptest.NewRequest("POST",
			"/webhook/double-complete-flow/"+string(step.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify duplicate call is rejected
		assert.Equal(t, 400, w.Code, "duplicate webhook call should return 400")

		var respErr api.ErrorResponse
		err = json.NewDecoder(w.Body).Decode(&respErr)
		assert.NoError(t, err)
		assert.Equal(t, "Work item already completed", respErr.Error)
	})
}

func TestHookFailTwice(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		step := &api.Step{
			ID:   "double-fail",
			Name: "Double Fail",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "double-fail-flow", StepID: step.ID},
			func() {
				err = env.Engine.StartFlow("double-fail-flow", plan)
				assert.NoError(t, err)
			},
		)

		flow, err := env.Engine.GetFlowState("double-fail-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range flow.Executions[step.ID].WorkItems {
			tkn = t
			break
		}

		// First FailWork should succeed
		result := api.StepResult{
			Success: false,
			Error:   "error1",
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/double-fail-flow/"+string(step.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		// Second FailWork with same token should be rejected (400)
		result = api.StepResult{
			Success: false,
			Error:   "error2",
		}

		body, _ = json.Marshal(result)
		req = httptest.NewRequest("POST",
			"/webhook/double-fail-flow/"+string(step.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code, "duplicate fail webhook should return 400")

		var respErr api.ErrorResponse
		_ = json.NewDecoder(w.Body).Decode(&respErr)
		assert.Equal(t, "Work item already completed", respErr.Error)
	})
}

func TestHookSuccess(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		assert.NoError(t, env.Engine.Start())
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

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "webhook-success-flow", StepID: step.ID},
			func() {
				err = env.Engine.StartFlow("webhook-success-flow", plan)
				assert.NoError(t, err)
			},
		)

		flow, err := env.Engine.GetFlowState("webhook-success-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range flow.Executions[step.ID].WorkItems {
			tkn = t
			break
		}

		result := api.StepResult{
			Success: true,
			Outputs: api.Args{"result": "success"},
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/webhook-success-flow/"+string(step.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

func TestHookWorkFailure(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		step := &api.Step{
			ID:   "webhook-fail",
			Name: "Webhook Fail",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "webhook-fail-flow", StepID: step.ID},
			func() {
				err = env.Engine.StartFlow("webhook-fail-flow", plan)
				assert.NoError(t, err)
			},
		)

		flow, err := env.Engine.GetFlowState("webhook-fail-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range flow.Executions[step.ID].WorkItems {
			tkn = t
			break
		}

		result := api.StepResult{
			Success: false,
			Error:   "step failed",
		}

		body, _ := json.Marshal(result)
		req := httptest.NewRequest("POST",
			"/webhook/webhook-fail-flow/"+string(step.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

func TestHookInvalidJSON(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		step := &api.Step{
			ID:   "webhook-badjson",
			Name: "Webhook Bad JSON",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetResponse(step.ID, api.Args{})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "webhook-badjson-flow", StepID: step.ID},
			func() {
				err = env.Engine.StartFlow("webhook-badjson-flow", plan)
				assert.NoError(t, err)
			},
		)

		flow, err := env.Engine.GetFlowState("webhook-badjson-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range flow.Executions[step.ID].WorkItems {
			tkn = t
			break
		}

		req := httptest.NewRequest("POST",
			"/webhook/webhook-badjson-flow/"+string(step.ID)+"/"+string(tkn),
			bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})
}
