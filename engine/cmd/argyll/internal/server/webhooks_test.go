package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kode4food/timebox/raft"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/server"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestCompensationCallbackIsPhaseSpecific(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		fs, tkn := seedCompensatingWebhook(t, env, "comp-fail")
		router := env.Server.SetupRoutes()
		invoke := api.CallbackPath(fs, tkn, api.ActionInvoke, helpers.CallbackSecret)
		compensate := api.CallbackPath(fs, tkn, api.ActionCompensate, helpers.CallbackSecret)

		// A delayed invocation callback must not complete compensation
		req := httptest.NewRequest("POST", invoke,
			bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkCompensating, fl.Executions[fs.StepID].WorkItems[tkn].Status)

		problem, err := json.Marshal(api.NewProblem(
			http.StatusUnprocessableEntity, "rollback failed",
		))
		assert.NoError(t, err)
		req = httptest.NewRequest("POST", compensate, bytes.NewReader(problem))
		req.Header.Set("Content-Type", api.ProblemJSONContentType)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		fl, err = env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkCompFailed, fl.Executions[fs.StepID].WorkItems[tkn].Status)
	})
}

func TestCompensationCallbackSucceedsWithoutBody(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		fs, tkn := seedCompensatingWebhook(t, env, "comp-ok")
		path := api.CallbackPath(fs, tkn, api.ActionCompensate, helpers.CallbackSecret)
		req := httptest.NewRequest("POST", path, nil)
		w := httptest.NewRecorder()

		env.Server.SetupRoutes().ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkCompensated, fl.Executions[fs.StepID].WorkItems[tkn].Status)
	})
}

func TestCallbackRejectsMalformedProblem(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		fs, tkn := seedCompensatingWebhook(t, env, "comp-bad")
		path := api.CallbackPath(fs, tkn, api.ActionCompensate, helpers.CallbackSecret)
		req := httptest.NewRequest("POST", path, bytes.NewReader([]byte(`{`)))
		req.Header.Set("Content-Type", api.ProblemJSONContentType)
		w := httptest.NewRecorder()

		env.Server.SetupRoutes().ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkCompensating, fl.Executions[fs.StepID].WorkItems[tkn].Status)
	})
}

func TestAgentCallbackUsage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		status api.WorkStatus
	}{
		{
			name:   "outputs with usage",
			body:   `{"outputs":{"result":"ok"},"usage":{"input_tokens":10}}`,
			status: api.WorkSucceeded,
		},
		{
			name:   "problem with usage",
			body:   `{"problem":{"status":422,"detail":"declined"},"usage":{"input_tokens":10}}`,
			status: api.WorkFailed,
		},
		{
			name:   "empty outputs with usage",
			body:   `{"outputs":{},"usage":{"input_tokens":10}}`,
			status: api.WorkSucceeded,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withTestServerEnv(t, func(env *testServerEnv) {
				fs, tkn := seedActiveWebhook(t, env, api.StepTypeAgent)
				path := api.CallbackPath(fs, tkn, api.ActionAgent, helpers.CallbackSecret)
				w := postCallback(env, path, bytes.NewBufferString(tc.body))

				assert.Equal(t, http.StatusOK, w.Code)
				fl, err := env.Engine.GetFlowState(fs.FlowID)
				assert.NoError(t, err)
				work := fl.Executions[fs.StepID].WorkItems[tkn]
				assert.Equal(t, tc.status, work.Status)
				assert.Equal(t, api.Usage{"input_tokens": 10}, work.Usage)
			})
		})
	}
}

func TestAgentCallbackRejects(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{`},
		{name: "malformed usage", body: `{"outputs":{},"usage":{"cost":"ten"}}`},
		{name: "non-finite usage", body: `{"outputs":{},"usage":{"cost":1e999}}`},
		{name: "missing outcome", body: `{"usage":{"cost":1}}`},
		{name: "ambiguous outcome", body: `{"outputs":{},"problem":{"status":422}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withTestServerEnv(t, func(env *testServerEnv) {
				fs, tkn := seedActiveWebhook(t, env, api.StepTypeAgent)
				path := api.CallbackPath(fs, tkn, api.ActionAgent, helpers.CallbackSecret)
				w := postCallback(env, path, bytes.NewBufferString(tc.body))

				assert.Equal(t, http.StatusBadRequest, w.Code)
				fl, err := env.Engine.GetFlowState(fs.FlowID)
				assert.NoError(t, err)
				assert.Equal(t, api.WorkActive, fl.Executions[fs.StepID].WorkItems[tkn].Status)
			})
		})
	}
}

func TestCallbackMissingExecution(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		fs, tkn := seedActiveWebhook(t, env, api.StepTypeService)
		fs.StepID = "missing"
		path := api.CallbackPath(fs, tkn, api.ActionInvoke, helpers.CallbackSecret)
		w := postCallback(env, path, bytes.NewBufferString(`{}`))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestCallbackBodyLimit(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		fs, tkn := seedActiveWebhook(t, env, api.StepTypeService)
		env.Server.WithCallbackLimit(16)
		path := api.CallbackPath(fs, tkn, api.ActionInvoke, helpers.CallbackSecret)
		body := `{"outputs":{"result":"far more than sixteen bytes"}}`
		w := postCallback(env, path, bytes.NewBufferString(body))
		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkActive, fl.Executions[fs.StepID].WorkItems[tkn].Status)
	})
}

func TestCallbackRejectsWrongSecret(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		fs, tkn := seedActiveWebhook(t, env, api.StepTypeService)
		path := api.CallbackPath(fs, tkn, api.ActionInvoke, "guessed")
		w := postCallback(env, path, bytes.NewBufferString(`{"outputs":{}}`))
		assert.Equal(t, http.StatusForbidden, w.Code)

		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkActive, fl.Executions[fs.StepID].WorkItems[tkn].Status)
	})
}

func TestFollowerCallbackMiss(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		withState := func(state raft.State) *testServerEnv {
			return &testServerEnv{
				Server: server.NewServer(env.Engine, env.EventHub, func() map[string]any {
					return map[string]any{"backend": map[string]any{"state": state}}
				}),
				TestEngineEnv: env,
			}
		}
		follower := withState(raft.StateFollower)
		leader := withState(raft.StateLeader)
		fs, tkn := seedActiveWebhook(t, follower, api.StepTypeService)
		missing := api.FlowStep{
			FlowID: "missing-flow",
			StepID: fs.StepID,
		}

		for _, tc := range []struct {
			name     string
			path     string
			body     string
			follower int
			leader   int
		}{
			{
				name:     "unknown flow",
				path:     api.CallbackPath(missing, tkn, api.ActionInvoke, helpers.CallbackSecret),
				body:     `{"outputs":{}}`,
				follower: http.StatusServiceUnavailable,
				leader:   http.StatusBadRequest,
			},
			{
				name:     "unknown token",
				path:     api.CallbackPath(fs, "missing", api.ActionInvoke, helpers.CallbackSecret),
				body:     `{"outputs":{}}`,
				follower: http.StatusServiceUnavailable,
				leader:   http.StatusBadRequest,
			},
			{
				name:     "wrong secret",
				path:     api.CallbackPath(fs, tkn, api.ActionInvoke, "guessed"),
				body:     `{"outputs":{}}`,
				follower: http.StatusServiceUnavailable,
				leader:   http.StatusForbidden,
			},
			{
				name:     "malformed body",
				path:     api.CallbackPath(fs, tkn, api.ActionInvoke, helpers.CallbackSecret),
				body:     `{`,
				follower: http.StatusBadRequest,
				leader:   http.StatusBadRequest,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				w := postCallback(follower, tc.path, strings.NewReader(tc.body))
				assert.Equal(t, tc.follower, w.Code)
				w = postCallback(leader, tc.path, strings.NewReader(tc.body))
				assert.Equal(t, tc.leader, w.Code)
			})
		}

		// A follower holding the work item settles it and accepts the repeat as a duplicate
		path := api.CallbackPath(fs, tkn, api.ActionInvoke, helpers.CallbackSecret)
		w := postCallback(follower, path, strings.NewReader(`{"outputs":{}}`))
		assert.Equal(t, http.StatusOK, w.Code)
		w = postCallback(follower, path, strings.NewReader(`{"outputs":{}}`))
		assert.Equal(t, http.StatusOK, w.Code)
		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		assert.Equal(t, api.WorkSucceeded, fl.Executions[fs.StepID].WorkItems[tkn].Status)
	})
}

func TestServiceCallbackKeepsRawOutputs(t *testing.T) {
	withTestServerEnv(t, func(env *testServerEnv) {
		fs, tkn := seedActiveWebhook(t, env, api.StepTypeService)
		body := `{"outputs":{"result":"ok"},"usage":{"cost":1}}`
		path := api.CallbackPath(fs, tkn, api.ActionInvoke, helpers.CallbackSecret)
		w := postCallback(env, path, bytes.NewBufferString(body))
		assert.Equal(t, http.StatusOK, w.Code)
		fl, err := env.Engine.GetFlowState(fs.FlowID)
		assert.NoError(t, err)
		work := fl.Executions[fs.StepID].WorkItems[tkn]
		assert.Equal(t, api.WorkSucceeded, work.Status)
		assert.Nil(t, work.Usage)
		assert.Contains(t, work.Outputs, api.Name("usage"))
	})
}

// dispatchedSecret returns the callback secret the engine handed the step's last invocation
func dispatchedSecret(t *testing.T, env *testServerEnv, sid api.StepID) string {
	t.Helper()
	assert.True(t, env.MockClient.WaitForInvocation(sid, time.Second))
	url, _ := env.MockClient.LastMetadata(sid)[api.MetaWebhookURL].(string)
	return url[strings.LastIndexByte(url, '/')+1:]
}

func postCallback(env *testServerEnv, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, body)
	req.Header.Set("Content-Type", api.JSONContentType)
	w := httptest.NewRecorder()
	env.Server.SetupRoutes().ServeHTTP(w, req)
	return w
}

func seedActiveWebhook(
	t *testing.T, env *testServerEnv, stepType api.StepType,
) (api.FlowStep, api.Token) {
	t.Helper()
	st := &api.Step{
		ID:   "active-step",
		Type: stepType,
	}
	fs := api.FlowStep{
		FlowID: "active-flow",
		StepID: st.ID,
	}
	tkn := api.Token("work")
	assert.NoError(t, env.SeedStartedWork(fs, &api.ExecutionPlan{
		Goals: api.Goals{Steps: []api.StepID{st.ID}},
		Steps: api.Steps{st.ID: st},
	}, tkn))
	return fs, tkn
}

func seedCompensatingWebhook(
	t *testing.T, env *testServerEnv, name string,
) (api.FlowStep, api.Token) {
	t.Helper()
	id := api.FlowID(name + "-flow")
	tkn := api.Token("work")
	st := &api.Step{
		ID:   api.StepID(name + "-step"),
		Type: api.StepTypeService,
	}
	pl := &api.ExecutionPlan{
		Goals: api.Goals{Steps: []api.StepID{st.ID}},
		Steps: api.Steps{st.ID: st},
	}
	events := []helpers.FlowEvent{
		{
			Type: api.EventTypeFlowStarted,
			Data: api.FlowStartedEvent{
				FlowID: id,
				Plan:   pl,
				Init:   api.InitArgs{},
			},
		},
		{
			Type: api.EventTypeStepStarted,
			Data: api.StepStartedEvent{
				FlowID:    id,
				StepID:    st.ID,
				Inputs:    api.Args{},
				WorkItems: map[api.Token]api.Args{tkn: {}},
			},
		},
		{
			Type: api.EventTypeWorkStarted,
			Data: api.WorkStartedEvent{
				FlowID:   id,
				StepID:   st.ID,
				Token:    tkn,
				Inputs:   api.Args{},
				Callback: api.CallbackHash(api.ActionInvoke, helpers.CallbackSecret),
			},
		},
		{
			Type: api.EventTypeWorkSucceeded,
			Data: api.WorkSucceededEvent{
				FlowID:  id,
				StepID:  st.ID,
				Token:   tkn,
				Outputs: api.Args{},
			},
		},
		{
			Type: api.EventTypeStepFailed,
			Data: api.StepFailedEvent{
				FlowID: id,
				StepID: st.ID,
				Error:  "forced failure",
			},
		},
		{
			Type: api.EventTypeFlowFailed,
			Data: api.FlowFailedEvent{
				FlowID: id,
				Error:  "forced failure",
			},
		},
		{
			Type: api.EventTypeCompStarted,
			Data: api.CompStartedEvent{
				FlowID:   id,
				StepID:   st.ID,
				Token:    tkn,
				Callback: api.CallbackHash(api.ActionCompensate, helpers.CallbackSecret),
			},
		},
	}
	assert.NoError(t, env.RaiseFlowEvents(id, events...))
	return api.FlowStep{
		FlowID: id,
		StepID: st.ID,
	}, tkn
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartPlan("invalid-work-flow", pl)
		assert.NoError(t, err)

		body, _ := json.Marshal(api.Args{})
		req := httptest.NewRequest("POST",
			"/callbacks/invalid-work-flow/"+string(st.ID)+"/fake-token/invoke/secret",
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
			"/callbacks/missing-flow/step/token/invoke/secret",
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		err = env.Engine.StartPlan("missing-exec-flow", pl)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST",
			"/callbacks/missing-exec-flow/unknown-step/token/invoke/secret",
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{
				FlowID: "double-complete-flow",
				StepID: st.ID,
			},
			func() {
				err = env.Engine.StartPlan("double-complete-flow", pl)
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
				string(tkn)+"/invoke/"+dispatchedSecret(t, env, st.ID),
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
				string(tkn)+"/invoke/"+dispatchedSecret(t, env, st.ID),
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{
				FlowID: "double-fail-flow",
				StepID: st.ID,
			},
			func() {
				err = env.Engine.StartPlan("double-fail-flow", pl)
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
		body, _ := json.Marshal(api.NewProblem(http.StatusUnprocessableEntity, "error1"))
		req := httptest.NewRequest("POST",
			"/callbacks/double-fail-flow/"+string(st.ID)+"/"+
				string(tkn)+"/invoke/"+dispatchedSecret(t, env, st.ID),
			bytes.NewReader(body))
		req.Header.Set("Content-Type", api.ProblemJSONContentType)
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		// Second FailWork with same token should be rejected (400)
		body, _ = json.Marshal(api.NewProblem(http.StatusUnprocessableEntity, "error2"))
		req = httptest.NewRequest("POST",
			"/callbacks/double-fail-flow/"+string(st.ID)+"/"+
				string(tkn)+"/invoke/"+dispatchedSecret(t, env, st.ID),
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{
				FlowID: "webhook-success-flow",
				StepID: st.ID,
			},
			func() {
				err = env.Engine.StartPlan("webhook-success-flow", pl)
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
		secret := "/" + dispatchedSecret(t, env, st.ID)

		settled := helpers.WaitForFlowState(t, env.Engine,
			helpers.FlowStateQuery{
				FlowID:  "webhook-success-flow",
				Timeout: time.Second,
				Accept: func(fl api.FlowState) bool {
					return fl.Executions[st.ID].WorkItems[tkn].Status ==
						api.WorkActive
				},
			})
		before := settled.Executions[st.ID].WorkItems[tkn].Status

		wrong := httptest.NewRequest("POST", path+"/compensate"+secret, nil)
		w := httptest.NewRecorder()
		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, wrong)
		assert.Equal(t, http.StatusForbidden, w.Code)

		after, err := env.Engine.GetFlowState("webhook-success-flow")
		assert.NoError(t, err)
		assert.Equal(t, before, after.Executions[st.ID].WorkItems[tkn].Status)

		body, _ := json.Marshal(api.Args{"result": "success"})
		req := httptest.NewRequest("POST", path+"/invoke"+secret, bytes.NewReader(body))
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{
				FlowID: "webhook-fail-flow",
				StepID: st.ID,
			},
			func() {
				err = env.Engine.StartPlan("webhook-fail-flow", pl)
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

		body, _ := json.Marshal(api.NewProblem(http.StatusUnprocessableEntity, "step failed"))
		req := httptest.NewRequest("POST",
			"/callbacks/webhook-fail-flow/"+string(st.ID)+"/"+
				string(tkn)+"/invoke/"+dispatchedSecret(t, env, st.ID),
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
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForStepStarted(
			api.FlowStep{
				FlowID: "webhook-badjson-flow",
				StepID: st.ID,
			},
			func() {
				err = env.Engine.StartPlan("webhook-badjson-flow", pl)
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
				string(tkn)+"/invoke/"+dispatchedSecret(t, env, st.ID),
			bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", api.JSONContentType)
		w := httptest.NewRecorder()

		router := env.Server.SetupRoutes()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})
}
