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

		st := &api.Step{
			ID:   "invalid-work-step",
			Name: "Invalid Work Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("invalid-work-flow", pl)
		assert.NoError(t, err)

		body, _ := json.Marshal(api.Args{})
		req := httptest.NewRequest("POST",
			"/webhook/invalid-work-flow/"+string(st.ID)+"/fake-token",
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
		st := &api.Step{
			ID:   "known-step",
			Name: "Known Step",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartFlow("missing-exec-flow", pl)
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

		st := &api.Step{
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

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "double-complete-flow", StepID: st.ID},
			func() {
				err = env.Engine.StartFlow("double-complete-flow", pl)
				assert.NoError(t, err)
			},
		)

		fl, err := env.Engine.GetFlowState("double-complete-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range fl.Executions[st.ID].WorkItems {
			tkn = t
			break
		}

		body, _ := json.Marshal(api.Args{"output": "value1"})
		req := httptest.NewRequest("POST",
			"/webhook/double-complete-flow/"+string(st.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		router := env.Server.SetupRoutes()
		var w *httptest.ResponseRecorder
		ex := env.WaitForStepStatus("double-complete-flow", st.ID, func() {
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
		})

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, api.StepCompleted, ex.Status)

		// Second webhook call with same token should be rejected (400) due to
		// invalid work state transition
		body, _ = json.Marshal(api.Args{"output": "value2"})
		req = httptest.NewRequest("POST",
			"/webhook/double-complete-flow/"+string(st.ID)+"/"+string(tkn),
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

		st := &api.Step{
			ID:   "double-fail",
			Name: "Double Fail",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "double-fail-flow", StepID: st.ID},
			func() {
				err = env.Engine.StartFlow("double-fail-flow", pl)
				assert.NoError(t, err)
			},
		)

		fl, err := env.Engine.GetFlowState("double-fail-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range fl.Executions[st.ID].WorkItems {
			tkn = t
			break
		}

		// First FailWork should succeed
		body, _ := json.Marshal(api.NewProblem(
			http.StatusUnprocessableEntity, "error1",
		))
		req := httptest.NewRequest("POST",
			"/webhook/double-fail-flow/"+string(st.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.ProblemJSONContentType)
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		// Second FailWork with same token should be rejected (400)
		body, _ = json.Marshal(api.NewProblem(
			http.StatusUnprocessableEntity, "error2",
		))
		req = httptest.NewRequest("POST",
			"/webhook/double-fail-flow/"+string(st.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.ProblemJSONContentType)
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

		st := &api.Step{
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

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "webhook-success-flow", StepID: st.ID},
			func() {
				err = env.Engine.StartFlow("webhook-success-flow", pl)
				assert.NoError(t, err)
			},
		)

		fl, err := env.Engine.GetFlowState("webhook-success-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range fl.Executions[st.ID].WorkItems {
			tkn = t
			break
		}

		body, _ := json.Marshal(api.Args{"result": "success"})
		req := httptest.NewRequest("POST",
			"/webhook/webhook-success-flow/"+string(st.ID)+"/"+string(tkn),
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

		st := &api.Step{
			ID:   "webhook-fail",
			Name: "Webhook Fail",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "webhook-fail-flow", StepID: st.ID},
			func() {
				err = env.Engine.StartFlow("webhook-fail-flow", pl)
				assert.NoError(t, err)
			},
		)

		fl, err := env.Engine.GetFlowState("webhook-fail-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range fl.Executions[st.ID].WorkItems {
			tkn = t
			break
		}

		body, _ := json.Marshal(api.NewProblem(
			http.StatusUnprocessableEntity, "step failed",
		))
		req := httptest.NewRequest("POST",
			"/webhook/webhook-fail-flow/"+string(st.ID)+"/"+string(tkn),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.ProblemJSONContentType)
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

		st := &api.Step{
			ID:   "webhook-badjson",
			Name: "Webhook Bad JSON",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{FlowID: "webhook-badjson-flow", StepID: st.ID},
			func() {
				err = env.Engine.StartFlow("webhook-badjson-flow", pl)
				assert.NoError(t, err)
			},
		)

		fl, err := env.Engine.GetFlowState("webhook-badjson-flow")
		assert.NoError(t, err)

		var tkn api.Token
		for t := range fl.Executions[st.ID].WorkItems {
			tkn = t
			break
		}

		req := httptest.NewRequest("POST",
			"/webhook/webhook-badjson-flow/"+string(st.ID)+"/"+string(tkn),
			bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})
}
