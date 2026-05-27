package step_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/internal/engine/step"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type testClient struct {
	inputs  api.Args
	meta    api.Metadata
	outputs api.Args
	invoked int
	compens int
	err     error
}

var _ client.Client = (*testClient)(nil)

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

func (c *testClient) InvokeCompensate(
	_ *api.Step, _, _ api.Args, _ api.Metadata,
) error {
	c.compens++
	return nil
}

type testRuntime struct {
	flowID        api.FlowID
	stepID        api.StepID
	meta          api.Metadata
	webhookURL    string
	completeToken api.Token
	completeOut   api.Args
	completeCalls int
	startToken    api.Token
	startInit     api.InitArgs
	startCalls    int
	healthStatus  api.HealthStatus
	healthError   string
	healthCalls   int
	webhookCalls  int
}

var _ step.Runtime = (*testRuntime)(nil)

func newRuntime(
	flowID api.FlowID, stepID api.StepID, meta api.Metadata, webhookURL string,
) (*testRuntime, *testRuntime) {
	rt := &testRuntime{
		flowID:     flowID,
		stepID:     stepID,
		meta:       meta,
		webhookURL: webhookURL,
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

func (r *testRuntime) WebhookURL(api.Token) string {
	r.webhookCalls++
	return r.webhookURL
}

func (r *testRuntime) CompleteWork(
	tkn api.Token, outputs api.Args,
) error {
	r.completeToken = tkn
	r.completeOut = outputs
	r.completeCalls++
	return nil
}

func (r *testRuntime) StartChildFlow(
	tkn api.Token, init api.InitArgs,
) (api.FlowID, error) {
	r.startToken = tkn
	r.startInit = init
	r.startCalls++
	return "child-flow", nil
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
		api.StepTypeFlow, api.StepTypeSync, api.StepTypeAsync,
		api.StepTypeScript,
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
	handler, err := reg.Lookup(api.StepTypeSync)
	assert.NoError(t, err)

	rt, calls := newRuntime(
		"flow-1", "step-1", api.Metadata{"source": "test"},
		"http://example.test/webhook",
	)
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{Endpoint: "http://example.test/execute"},
	}

	err = handler.Execute(rt, st, api.Args{"input": "value"}, "token-1")
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
	handler, err := reg.Lookup(api.StepTypeAsync)
	assert.NoError(t, err)

	rt, calls := newRuntime(
		"flow-1", "step-1", api.Metadata{"source": "test"},
		"http://example.test/webhook",
	)
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeAsync,
		HTTP: &api.HTTPConfig{Endpoint: "http://example.test/execute"},
	}

	err = handler.Execute(rt, st, api.Args{"input": "value"}, "token-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, cl.invoked)
	assert.Equal(t, "http://example.test/webhook", cl.meta[api.MetaWebhookURL])
	assert.Equal(t, api.Token("token-1"), cl.meta[api.MetaReceiptToken])
	assert.Equal(t, 0, calls.completeCalls)
	assert.Equal(t, 1, calls.webhookCalls)
}

func TestFlowHandlerStartsChildFlow(t *testing.T) {
	reg := newRegistry(&testClient{})
	handler, err := reg.Lookup(api.StepTypeFlow)
	assert.NoError(t, err)

	rt, calls := newRuntime("flow-parent", "step-parent", nil, "")
	st := &api.Step{
		ID:   "step-parent",
		Type: api.StepTypeFlow,
	}

	err = handler.Execute(rt, st, api.Args{"foo": "bar"}, "token-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, calls.startCalls)
	assert.Equal(t, api.Token("token-1"), calls.startToken)
	assert.Equal(t,
		api.InitArgs{"foo": []any{"bar"}}, calls.startInit,
	)
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

func newRegistry(c client.Client) *step.Registry {
	return step.NewRegistry(step.DefaultHandlers(script.NewRegistry(), c))
}
