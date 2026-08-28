package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestCompensationCallbackIsPhaseSpecific(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		fs, tkn := seedCompensatingWebhook(t, env, "comp-fail")
		router := env.Server.SetupRoutes()
		base := fmt.Sprintf("/callbacks/%s/%s/%s",
			fs.FlowID, fs.StepID, tkn)

		// A delayed invocation callback must not complete compensation.
		req := httptest.NewRequest("POST", base+"/invoke",
			bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkCompensating,
			fl.Executions[fs.StepID].WorkItems[tkn].Status)

		problem, err := json.Marshal(api.NewProblem(
			http.StatusUnprocessableEntity, "rollback failed",
		))
		assert.NoError(t, err)
		req = httptest.NewRequest("POST", base+"/compensate",
			bytes.NewReader(problem))
		req.Header.Set("Content-Type", api.ProblemJSONContentType)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		fl, err = env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkCompFailed,
			fl.Executions[fs.StepID].WorkItems[tkn].Status)
	})
}

func TestCompensationCallbackSucceedsWithoutBody(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		fs, tkn := seedCompensatingWebhook(t, env, "comp-ok")
		path := fmt.Sprintf("/callbacks/%s/%s/%s/compensate",
			fs.FlowID, fs.StepID, tkn)
		req := httptest.NewRequest("POST", path, nil)
		w := httptest.NewRecorder()

		env.Server.SetupRoutes().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkCompensated,
			fl.Executions[fs.StepID].WorkItems[tkn].Status)
	})
}

func seedCompensatingWebhook(
	t *testing.T, env *testServerEnv, name string,
) (api.FlowStep, api.Token) {
	t.Helper()
	id := api.FlowID(name + "-flow")
	tkn := api.Token("work")
	st := &api.Step{ID: api.StepID(name + "-step")}
	pl := &api.ExecutionPlan{
		Goals: []api.StepID{st.ID},
		Steps: api.Steps{st.ID: st},
	}
	events := []helpers.FlowEvent{
		{Type: api.EventTypeFlowStarted, Data: api.FlowStartedEvent{
			FlowID: id, Plan: pl, Init: api.InitArgs{}, Compensate: true,
		}},
		{Type: api.EventTypeStepStarted, Data: api.StepStartedEvent{
			FlowID: id, StepID: st.ID, Inputs: api.Args{},
			WorkItems: map[api.Token]api.Args{tkn: {}},
		}},
		{Type: api.EventTypeWorkStarted, Data: api.WorkStartedEvent{
			FlowID: id, StepID: st.ID, Token: tkn, Inputs: api.Args{},
		}},
		{Type: api.EventTypeWorkSucceeded, Data: api.WorkSucceededEvent{
			FlowID: id, StepID: st.ID, Token: tkn, Outputs: api.Args{},
		}},
		{Type: api.EventTypeStepFailed, Data: api.StepFailedEvent{
			FlowID: id, StepID: st.ID, Error: "forced failure",
		}},
		{Type: api.EventTypeFlowFailed, Data: api.FlowFailedEvent{
			FlowID: id, Error: "forced failure",
		}},
		{Type: api.EventTypeCompStarted, Data: api.CompStartedEvent{
			FlowID: id, StepID: st.ID, Token: tkn,
		}},
	}
	assert.NoError(t, env.RaiseFlowEvents(id, events...))
	return api.FlowStep{FlowID: id, StepID: st.ID}, tkn
}

func TestHookInvalidWorkItem(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := &api.Step{
			ID:   "invalid-work-step",
			Name: "Invalid Work Step",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://test:8080",
					Mode:     api.ActionModeAsync,
				},
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
			"/callbacks/invalid-work-flow/"+string(st.ID)+"/fake-token/invoke",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})
}

func TestHookFlowMissing(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		req := httptest.NewRequest("POST",
			"/callbacks/missing-flow/step/token/invoke",
			bytes.NewReader([]byte(`{"success":true}`)))
		req.Header.Set("Content-Type", api.JSONContentType)
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
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://test:8080",
					Mode:     api.ActionModeAsync,
				},
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
			"/callbacks/missing-exec-flow/unknown-step/token/invoke",
			bytes.NewReader([]byte(`{"success":true}`)))
		req.Header.Set("Content-Type", api.JSONContentType)
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
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://test:8080",
					Mode:     api.ActionModeAsync,
				},
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
			"/callbacks/double-complete-flow/"+string(st.ID)+"/"+
				string(tkn)+"/invoke",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.JSONContentType)

		router := env.Server.SetupRoutes()
		var w *httptest.ResponseRecorder
		ex := env.WaitForStepStatus("double-complete-flow", st.ID, func() {
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
		})

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, api.StepCompleted, ex.Status)

		// Second webhook call with same token is a duplicate terminal callback
		body, _ = json.Marshal(api.Args{"output": "value2"})
		req = httptest.NewRequest("POST",
			"/callbacks/double-complete-flow/"+string(st.ID)+"/"+
				string(tkn)+"/invoke",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.JSONContentType)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code, "duplicate webhook call should be ignored")
		assert.Empty(t, w.Body.String())
	})
}

func TestHookFailTwice(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := &api.Step{
			ID:   "double-fail",
			Name: "Double Fail",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://test:8080",
					Mode:     api.ActionModeAsync,
				},
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
			"/callbacks/double-fail-flow/"+string(st.ID)+"/"+
				string(tkn)+"/invoke",
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
			"/callbacks/double-fail-flow/"+string(st.ID)+"/"+
				string(tkn)+"/invoke",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.ProblemJSONContentType)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code, "duplicate fail webhook should be ignored")
		assert.Empty(t, w.Body.String())
	})
}

func TestHookSuccess(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := &api.Step{
			ID:   "webhook-success",
			Name: "Webhook Success",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://test:8080",
					Mode:     api.ActionModeAsync,
				},
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
		path := "/callbacks/webhook-success-flow/" + string(st.ID) + "/" +
			string(tkn)
		wrong := httptest.NewRequest("POST", path+"/compensate", nil)
		w := httptest.NewRecorder()
		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, wrong)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		before := fl.Executions[st.ID].WorkItems[tkn].Status
		after, err := env.Engine.GetFlowState("webhook-success-flow")
		assert.NoError(t, err)
		assert.Equal(t, before,
			after.Executions[st.ID].WorkItems[tkn].Status)

		body, _ := json.Marshal(api.Args{"result": "success"})
		req := httptest.NewRequest("POST",
			path+"/invoke",
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.JSONContentType)
		w = httptest.NewRecorder()

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
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://test:8080",
					Mode:     api.ActionModeAsync,
				},
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
			"/callbacks/webhook-fail-flow/"+string(st.ID)+"/"+
				string(tkn)+"/invoke",
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
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://test:8080",
					Mode:     api.ActionModeAsync,
				},
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
			"/callbacks/webhook-badjson-flow/"+string(st.ID)+"/"+
				string(tkn)+"/invoke",
			bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})
}
