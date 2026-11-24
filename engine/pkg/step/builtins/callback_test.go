package builtins_test

import (
	"errors"
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
	usage   api.Usage
}

const (
	callbackFlow  api.FlowID = "flow"
	callbackStep  api.StepID = "step"
	callbackToken api.Token  = "token"
	callbackKey              = "secret"
	compensateKey            = "undo-secret"
)

var errEngineDown = errors.New("engine down")

func TestCallbackLookup(t *testing.T) {
	eng := newCallbackEngine(api.WorkActive)
	eng.flowErr = api.ErrFlowNotFound
	err := builtins.Callback(eng, invokeRequest(api.Args{}))
	assert.ErrorIs(t, err, builtins.ErrInvalidCallback)

	eng = newCallbackEngine(api.WorkActive)
	req := invokeRequest(api.Args{})
	req.FlowStep.StepID = "missing"
	err = builtins.Callback(eng, req)
	assert.ErrorIs(t, err, builtins.ErrCallbackExecutionNotFound)

	req = invokeRequest(api.Args{})
	req.Token = "missing"
	err = builtins.Callback(eng, req)
	assert.ErrorIs(t, err, api.ErrWorkItemNotFound)

	req = invokeRequest(api.Args{})
	req.Action = "bogus"
	err = builtins.Callback(eng, req)
	assert.ErrorIs(t, err, builtins.ErrInvalidCallbackAction)
	assert.Empty(t, eng.called)
}

func TestCallbackSecret(t *testing.T) {
	for _, secret := range []string{"", "wrong"} {
		eng := newCallbackEngine(api.WorkActive)
		req := invokeRequest(api.Args{})
		req.Secret = secret
		err := builtins.Callback(eng, req)
		assert.ErrorIs(t, err, builtins.ErrCallbackSecret)
		assert.Empty(t, eng.called)
	}

	// A late callback from an earlier attempt still settles the work
	eng := newCallbackEngine(api.WorkActive)
	req := invokeRequest(api.Args{})
	req.Secret = "earlier attempt"
	assert.NoError(t, builtins.Callback(eng, req))
	assert.Equal(t, "CompleteWork", eng.called)

	// Each phase's secret settles only that phase
	eng = newCallbackEngine(api.WorkCompensating)
	comp := compensateRequest(nil)
	comp.Secret = callbackKey
	assert.ErrorIs(t, builtins.Callback(eng, comp), builtins.ErrCallbackSecret)
	req = invokeRequest(api.Args{})
	req.Secret = compensateKey
	assert.ErrorIs(t, builtins.Callback(eng, req), builtins.ErrCallbackSecret)
	assert.Empty(t, eng.called)
}

func TestCallbackInvoke(t *testing.T) {
	for _, action := range []api.CallbackAction{api.ActionInvoke, api.ActionAgent} {
		t.Run(string(action), func(t *testing.T) {
			usage := api.Usage{"input_tokens": 10}
			eng := newCallbackEngine(api.WorkActive)
			err := builtins.Callback(eng, &builtins.CallbackRequest{
				FlowStep: callbackFlowStep(),
				Token:    callbackToken,
				Action:   action,
				Secret:   callbackKey,
				Outputs:  api.Args{"result": "ok"},
				Usage:    usage,
			})
			assert.NoError(t, err)
			assert.Equal(t, "CompleteWork", eng.called)
			assert.Equal(t, api.Args{"result": "ok"}, eng.outputs)
			assert.Equal(t, usage, eng.usage)
		})
	}
}

func TestCallbackInvokeProblem(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		called string
	}{
		{name: "permanent", status: 422, called: "FailWork"},
		{name: "retryable", status: 503, called: "RetryWork"},
		{name: "no status", called: "FailWork"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usage := api.Usage{"input_tokens": 3}
			eng := newCallbackEngine(api.WorkActive)
			err := builtins.Callback(eng, &builtins.CallbackRequest{
				FlowStep: callbackFlowStep(),
				Token:    callbackToken,
				Action:   api.ActionInvoke,
				Secret:   callbackKey,
				Problem: &api.ProblemDetails{
					Status: tc.status,
					Detail: "boom",
				},
				Usage: usage,
			})
			assert.NoError(t, err)
			assert.Equal(t, tc.called, eng.called)
			assert.Equal(t, "boom", eng.errMsg)
			assert.Equal(t, usage, eng.usage)
		})
	}
}

func TestCallbackInvokeErrors(t *testing.T) {
	eng := newCallbackEngine(api.WorkActive)
	eng.workErr = api.ErrInvalidWorkTransition
	err := builtins.Callback(eng, invokeRequest(api.Args{}))
	assert.NoError(t, err)

	eng.workErr = errEngineDown
	err = builtins.Callback(eng, invokeRequest(api.Args{}))
	assert.ErrorIs(t, err, errEngineDown)
	assert.NotErrorIs(t, err, builtins.ErrInvalidCallback)
}

func TestCallbackCompensate(t *testing.T) {
	eng := newCallbackEngine(api.WorkActive)
	err := builtins.Callback(eng, compensateRequest(nil))
	assert.ErrorIs(t, err, builtins.ErrCallbackNotCompensating)
	assert.Empty(t, eng.called)

	eng = newCallbackEngine(api.WorkCompensating)
	err = builtins.Callback(eng, compensateRequest(nil))
	assert.NoError(t, err)
	assert.Equal(t, "CompleteCompensation", eng.called)

	eng = newCallbackEngine(api.WorkCompensating)
	err = builtins.Callback(eng, compensateRequest(&api.ProblemDetails{Detail: "rollback failed"}))
	assert.NoError(t, err)
	assert.Equal(t, "FailCompensation", eng.called)
	assert.Equal(t, "rollback failed", eng.errMsg)

	eng = newCallbackEngine(api.WorkCompensating)
	eng.workErr = errEngineDown
	err = builtins.Callback(eng, compensateRequest(nil))
	assert.ErrorIs(t, err, errEngineDown)
}

func (e *callbackEngine) GetFlowState(api.FlowID) (api.FlowState, error) {
	return e.flow, e.flowErr
}

func (e *callbackEngine) CompleteWork(
	_ api.FlowStep, _ api.Token, outputs api.Args, usage api.Usage,
) error {
	e.called = "CompleteWork"
	e.outputs = outputs
	e.usage = usage
	return e.workErr
}

func (e *callbackEngine) RetryWork(
	_ api.FlowStep, _ api.Token, cause error, usage api.Usage,
) error {
	e.called = "RetryWork"
	e.errMsg = cause.Error()
	e.usage = usage
	return e.workErr
}

func (e *callbackEngine) FailWork(
	_ api.FlowStep, _ api.Token, cause error, usage api.Usage,
) error {
	e.called = "FailWork"
	e.errMsg = cause.Error()
	e.usage = usage
	return e.workErr
}

func (e *callbackEngine) CompleteCompensation(api.FlowStep, api.Token) error {
	e.called = "CompleteCompensation"
	return e.workErr
}

func (e *callbackEngine) FailCompensation(_ api.FlowStep, _ api.Token, errMsg string) error {
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
						callbackToken: {
							Status: status,
							Callbacks: []string{
								api.CallbackHash(api.ActionInvoke, "earlier attempt"),
								api.CallbackHash(api.ActionInvoke, callbackKey),
								api.CallbackHash(api.ActionCompensate, compensateKey),
							},
						},
					},
				},
			},
		},
	}
}

func callbackFlowStep() api.FlowStep {
	return api.FlowStep{
		FlowID: callbackFlow,
		StepID: callbackStep,
	}
}

func invokeRequest(outputs api.Args) *builtins.CallbackRequest {
	return &builtins.CallbackRequest{
		FlowStep: callbackFlowStep(),
		Token:    callbackToken,
		Action:   api.ActionInvoke,
		Secret:   callbackKey,
		Outputs:  outputs,
	}
}

func compensateRequest(problem *api.ProblemDetails) *builtins.CallbackRequest {
	return &builtins.CallbackRequest{
		FlowStep: callbackFlowStep(),
		Token:    callbackToken,
		Action:   api.ActionCompensate,
		Secret:   compensateKey,
		Problem:  problem,
	}
}
