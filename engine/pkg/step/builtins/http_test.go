package builtins_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

func TestHTTPHandlerPropagatesMetadata(t *testing.T) {
	cl := &testClient{outputs: api.Args{"result": "ok"}}
	reg := newRegistry(cl)
	handler, err := reg.Lookup(api.StepTypeService)
	assert.NoError(t, err)

	rt, calls := newRuntime(
		"flow-1", "step-1", api.Metadata{"source": "test"},
	)
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://example.test/execute"},
		},
	}

	err = handler.Invoke(rt, st, api.Args{"input": "value"}, "token-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, cl.invoked)
	assert.Equal(t, "value", cl.inputs[api.Name("input")])
	assert.Equal(t, api.FlowID("flow-1"), cl.meta[api.MetaFlowID])
	assert.Equal(t, api.StepID("step-1"), cl.meta[api.MetaStepID])
	assert.Equal(t, api.Token("token-1"), cl.meta[api.MetaReceiptToken])
	assert.Equal(t, 0, calls.healthCalls)
	assert.Equal(t, 1, calls.completeCalls)
	assert.Equal(t, api.Token("token-1"), calls.completeToken)
	assert.Equal(t, api.Args{"result": "ok"}, calls.completeOut)
}

func TestHTTPHandlerAsyncAddsWebhookURL(t *testing.T) {
	cl := &testClient{}
	reg := newRegistry(cl)
	handler, err := reg.Lookup(api.StepTypeService)
	assert.NoError(t, err)

	rt, calls := newRuntime(
		"flow-1", "step-1", api.Metadata{"source": "test"},
	)
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://example.test/execute",
				Mode:     api.ActionModeAsync,
			},
		},
	}

	err = handler.Invoke(rt, st, api.Args{"input": "value"}, "token-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, cl.invoked)
	assert.Equal(t,
		testCallbackBase+"/callbacks/flow-1/step-1/token-1/invoke",
		cl.meta[api.MetaWebhookURL])
	assert.Equal(t, api.Token("token-1"), cl.meta[api.MetaReceiptToken])
	assert.Equal(t, 0, calls.completeCalls)
}

func TestAsyncStepNeedsCallbackURL(t *testing.T) {
	reg := step.NewRegistry(builtins.All(
		&testClient{}, nil,
	))

	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://example.test/execute",
				Mode:     api.ActionModeAsync,
			},
		},
	}
	assert.ErrorIs(t, reg.Validate(st), builtins.ErrNoCallbackURL)

	st.HTTP.Invoke.Mode = ""
	assert.NoError(t, reg.Validate(st))
}

func TestHTTPCompensatorInvokes(t *testing.T) {
	cl := &testClient{}
	reg := newRegistry(cl)
	st := &api.Step{
		Type:     api.StepTypeService,
		Handling: api.HandlingCompensated,
		HTTP: &api.HTTPConfig{
			Compensate: &api.HTTPAction{Endpoint: "http://test/undo"},
		},
	}
	comp, err := reg.Compensator(st)
	assert.NoError(t, err)
	assert.NotNil(t, comp)
	completed, err := comp(step.CompensateRequest{
		Step:     st,
		Inputs:   api.Args{"in": "v"},
		Outputs:  api.Args{"out": "v"},
		Metadata: api.Metadata{},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, cl.compens)
	assert.True(t, completed)
}

func TestCompensatorEndpoint(t *testing.T) {
	reg := newRegistry(&testClient{})
	comp, err := reg.Compensator(&api.Step{
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://test/work"},
		},
	})
	assert.NoError(t, err)
	assert.Nil(t, comp)
}

func TestCompensationCallback(t *testing.T) {
	cl := &testClient{}
	fs := api.FlowStep{FlowID: "flow", StepID: "step"}
	token := api.Token("token")
	callback := func(
		got api.FlowStep, tkn api.Token, action api.CallbackAction,
	) string {
		assert.Equal(t, fs, got)
		assert.Equal(t, token, tkn)
		assert.Equal(t, api.ActionCompensate, action)
		return "https://host.test/custom/undo"
	}
	h := builtins.HTTP(cl, callback)
	meta := api.Metadata{api.MetaFlowID: fs.FlowID}
	completed, err := h.Compensate(step.CompensateRequest{
		Step: &api.Step{
			ID: fs.StepID,
			HTTP: &api.HTTPConfig{
				Compensate: &api.HTTPAction{
					Endpoint: "https://step.test/undo",
					Mode:     api.ActionModeAsync,
				},
			},
		},
		FlowID:   fs.FlowID,
		Token:    token,
		Metadata: meta,
	})
	assert.NoError(t, err)
	assert.False(t, completed)
	assert.Equal(t, "https://host.test/custom/undo",
		cl.meta[api.MetaWebhookURL])
	assert.NotContains(t, meta, api.MetaWebhookURL)
}

func TestMetaInputs(t *testing.T) {
	cl := &testClient{outputs: api.Args{}}
	reg := newRegistry(cl)
	handler, err := reg.Lookup(api.StepTypeService)
	assert.NoError(t, err)

	rt, _ := newRuntime("flow-1", "step-1", api.Metadata{})
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://example.test"},
		},
		Attributes: api.AttributeSpecs{
			"token": {
				Role: api.RoleMeta,
				Meta: &api.MetaConfig{Key: api.MetaReceiptToken},
			},
		},
	}

	err = handler.Invoke(rt, st, api.Args{}, "my-token")
	assert.NoError(t, err)
	assert.Equal(t, api.Token("my-token"), cl.inputs[api.Name("token")])
}
