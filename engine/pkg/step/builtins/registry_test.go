package builtins_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

type testClient struct {
	inputs  api.Args
	meta    api.Metadata
	outputs api.Args
	invoked int
	compens int
	err     error
}

var _ builtins.Client = (*testClient)(nil)

func (c *testClient) Invoke(
	_ *api.Step, inputs api.Args, meta api.Metadata,
) (api.Args, error) {
	c.inputs = inputs
	c.meta = meta
	c.invoked++
	if c.err != nil {
		return nil, c.err
	}
	return c.outputs, nil
}

func (c *testClient) Compensate(req step.CompensateRequest) error {
	c.compens++
	c.meta = req.Metadata
	return c.err
}

type testRuntime struct {
	flowID        api.FlowID
	stepID        api.StepID
	meta          api.Metadata
	completeToken api.Token
	completeOut   api.Args
	completeCalls int
	healthStatus  api.HealthStatus
	healthError   string
	healthCalls   int
}

var _ step.Runtime = (*testRuntime)(nil)

const testCallbackBase = "http://example.test"

func newRegistry(c builtins.Client) *step.Registry {
	return step.NewRegistry(builtins.All(
		c, builtins.BaseCallbackURL(testCallbackBase),
	))
}

func newRuntime(
	fid api.FlowID, sid api.StepID, meta api.Metadata,
) (*testRuntime, *testRuntime) {
	rt := &testRuntime{
		flowID: fid,
		stepID: sid,
		meta:   meta,
	}
	return rt, rt
}

func (r *testRuntime) FlowID() api.FlowID {
	return r.flowID
}

func (r *testRuntime) StepID() api.StepID {
	return r.stepID
}

func (r *testRuntime) Metadata() api.Metadata {
	return r.meta
}

func (r *testRuntime) CompleteWork(
	tkn api.Token, outputs api.Args,
) error {
	r.completeToken = tkn
	r.completeOut = outputs
	r.completeCalls++
	return nil
}

func (r *testRuntime) UpdateHealth(
	status api.HealthStatus, errMsg string,
) error {
	r.healthStatus = status
	r.healthError = errMsg
	r.healthCalls++
	return nil
}

func TestRegistryRegistersBuiltIns(t *testing.T) {
	reg := newRegistry(&testClient{})

	for _, typ := range []api.StepType{
		api.StepTypeFlow, api.StepTypeService, api.StepTypeScript,
	} {
		_, err := reg.Lookup(typ)
		assert.NoError(t, err)
	}
}

func TestRegistryRejectsUnknownStepType(t *testing.T) {
	reg := newRegistry(&testClient{})

	err := reg.Validate(&api.Step{Type: "unknown"})
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

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

func TestFlowHandlerHasNoExternalExecution(t *testing.T) {
	reg := newRegistry(&testClient{})
	handler, err := reg.Lookup(api.StepTypeFlow)
	assert.NoError(t, err)
	assert.Nil(t, handler.Invoke)
}

func TestRegistryLookupMissing(t *testing.T) {
	reg := newRegistry(&testClient{})

	_, err := reg.Lookup("missing")
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

func TestRegistryIncludesBootstrappedHandler(t *testing.T) {
	handler := &step.Handler{}
	reg := step.NewRegistry(step.Handlers{"custom": handler})

	got, err := reg.Lookup("custom")
	assert.NoError(t, err)
	assert.Same(t, handler, got)
}
