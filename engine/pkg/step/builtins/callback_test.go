package builtins_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

type callbackEngine struct {
	flow    api.FlowState
	flowErr error
	workErr error
	called  string
	errMsg  string
	outputs api.Args
}

const (
	callbackFlow  api.FlowID = "flow"
	callbackStep  api.StepID = "step"
	callbackToken api.Token  = "token"
)

var errEngineDown = errors.New("engine down")

func TestCallbackLookup(t *testing.T) {
	eng := newCallbackEngine(api.WorkActive)
	eng.flowErr = api.ErrFlowNotFound
	status, err := builtins.Callback(eng, invokeRequest(`{}`))
	assert.Equal(t, http.StatusBadRequest, status)
	assert.ErrorIs(t, err, api.ErrFlowNotFound)

	eng = newCallbackEngine(api.WorkActive)
	req := invokeRequest(`{}`)
	req.FlowStep.StepID = "missing"
	status, err = builtins.Callback(eng, req)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.ErrorIs(t, err, builtins.ErrInvalidCallback)

	req = invokeRequest(`{}`)
	req.Token = "missing"
	status, err = builtins.Callback(eng, req)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.ErrorIs(t, err, api.ErrWorkItemNotFound)

	req = invokeRequest(`{}`)
	req.Action = "bogus"
	status, err = builtins.Callback(eng, req)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.ErrorIs(t, err, builtins.ErrInvalidCallback)
	assert.Empty(t, eng.called)
}

func TestCallbackInvoke(t *testing.T) {
	eng := newCallbackEngine(api.WorkActive)
	status, err := builtins.Callback(eng, invokeRequest(`{"result":"ok"}`))
	assert.Equal(t, http.StatusOK, status)
	assert.NoError(t, err)
	assert.Equal(t, "CompleteWork", eng.called)
	assert.Equal(t, api.Args{"result": "ok"}, eng.outputs)

	eng = newCallbackEngine(api.WorkActive)
	req := invokeRequest(`{"detail":"boom"}`)
	req.Request.Header.Set("Content-Type", api.ProblemJSONContentType)
	status, err = builtins.Callback(eng, req)
	assert.Equal(t, http.StatusOK, status)
	assert.NoError(t, err)
	assert.Equal(t, "FailWork", eng.called)
	assert.Equal(t, "boom", eng.errMsg)
}

func TestCallbackInvokeErrors(t *testing.T) {
	eng := newCallbackEngine(api.WorkActive)
	status, err := builtins.Callback(eng, invokeRequest("invalid json"))
	assert.Equal(t, http.StatusBadRequest, status)
	assert.ErrorIs(t, err, builtins.ErrInvalidCallback)

	status, err = builtins.Callback(eng, invokeRequest(""))
	assert.Equal(t, http.StatusBadRequest, status)
	assert.ErrorIs(t, err, builtins.ErrInvalidCallback)
	assert.Empty(t, eng.called)

	eng.workErr = api.ErrInvalidWorkTransition
	status, err = builtins.Callback(eng, invokeRequest(`{}`))
	assert.Equal(t, http.StatusOK, status)
	assert.NoError(t, err)

	eng.workErr = errEngineDown
	status, err = builtins.Callback(eng, invokeRequest(`{}`))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.ErrorIs(t, err, errEngineDown)
}

func TestCallbackCompensate(t *testing.T) {
	eng := newCallbackEngine(api.WorkActive)
	status, err := builtins.Callback(eng, compensateRequest(""))
	assert.Equal(t, http.StatusBadRequest, status)
	assert.ErrorIs(t, err, builtins.ErrInvalidCallback)
	assert.Empty(t, eng.called)

	eng = newCallbackEngine(api.WorkCompensating)
	status, err = builtins.Callback(eng, compensateRequest(""))
	assert.Equal(t, http.StatusOK, status)
	assert.NoError(t, err)
	assert.Equal(t, "CompleteCompensation", eng.called)

	eng = newCallbackEngine(api.WorkCompensating)
	req := compensateRequest(`{"detail":"rollback failed"}`)
	req.Request.Header.Set("Content-Type", api.ProblemJSONContentType)
	status, err = builtins.Callback(eng, req)
	assert.Equal(t, http.StatusOK, status)
	assert.NoError(t, err)
	assert.Equal(t, "FailCompensation", eng.called)
	assert.Equal(t, "rollback failed", eng.errMsg)

	eng = newCallbackEngine(api.WorkCompensating)
	req = compensateRequest("invalid json")
	req.Request.Header.Set("Content-Type", api.ProblemJSONContentType)
	status, err = builtins.Callback(eng, req)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.ErrorIs(t, err, builtins.ErrInvalidCallback)

	eng = newCallbackEngine(api.WorkCompensating)
	eng.workErr = errEngineDown
	status, err = builtins.Callback(eng, compensateRequest(""))
	assert.Equal(t, http.StatusInternalServerError, status)
	assert.ErrorIs(t, err, errEngineDown)
}

func (e *callbackEngine) GetFlowState(api.FlowID) (api.FlowState, error) {
	return e.flow, e.flowErr
}

func (e *callbackEngine) CompleteWork(
	_ api.FlowStep, _ api.Token, outputs api.Args,
) error {
	e.called = "CompleteWork"
	e.outputs = outputs
	return e.workErr
}

func (e *callbackEngine) FailWork(
	_ api.FlowStep, _ api.Token, errMsg string,
) error {
	e.called = "FailWork"
	e.errMsg = errMsg
	return e.workErr
}

func (e *callbackEngine) CompleteCompensation(api.FlowStep, api.Token) error {
	e.called = "CompleteCompensation"
	return e.workErr
}

func (e *callbackEngine) FailCompensation(
	_ api.FlowStep, _ api.Token, errMsg string,
) error {
	e.called = "FailCompensation"
	e.errMsg = errMsg
	return e.workErr
}

func newCallbackEngine(status api.WorkStatus) *callbackEngine {
	return &callbackEngine{
		flow: api.FlowState{
			ID: callbackFlow,
			Executions: api.Executions{
				callbackStep: {
					WorkItems: api.WorkItems{
						callbackToken: {Status: status},
					},
				},
			},
		},
	}
}

func invokeRequest(body string) *builtins.CallbackRequest {
	return callbackRequest(api.ActionInvoke, body)
}

func compensateRequest(body string) *builtins.CallbackRequest {
	return callbackRequest(api.ActionCompensate, body)
}

func callbackRequest(
	action api.CallbackAction, body string,
) *builtins.CallbackRequest {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", api.JSONContentType)
	return &builtins.CallbackRequest{
		FlowStep: api.FlowStep{FlowID: callbackFlow, StepID: callbackStep},
		Token:    callbackToken,
		Action:   action,
		Request:  r,
	}
}
