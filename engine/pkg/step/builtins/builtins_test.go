package builtins_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

type (
	testClient struct {
		inputs  api.Args
		meta    api.Metadata
		outputs api.Args
		invoked int
		compens int
		err     error
	}

	testRuntime struct {
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
)

const testCallbackBase = "http://example.test"

var (
	_ builtins.Client = (*testClient)(nil)
	_ step.Runtime    = (*testRuntime)(nil)
)

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

func TestRegistryLookupMissing(t *testing.T) {
	reg := newRegistry(&testClient{})

	_, err := reg.Lookup("missing")
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

func TestRegistryValidation(t *testing.T) {
	reg := newRegistry(&testClient{})
	// sync handler has no Validate func -- should return nil
	err := reg.Validate(&api.Step{
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://example.test"},
		},
	})
	assert.NoError(t, err)
}

func TestRegistryHealthNil(t *testing.T) {
	reg := newRegistry(&testClient{})
	// sync handler has no Health func -- should return HealthUnknown
	h, err := reg.Health(&api.Step{Type: api.StepTypeService})
	assert.NoError(t, err)
	assert.Equal(t, api.HealthUnknown, h.Status)
}

func TestRegistryHealthType(t *testing.T) {
	reg := newRegistry(&testClient{})
	_, err := reg.Health(&api.Step{Type: "unknown"})
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

func TestRegistryChildrenNil(t *testing.T) {
	reg := newRegistry(&testClient{})
	// sync handler has no Children func -- should return nil
	ids, err := reg.Children(&api.Step{Type: api.StepTypeService})
	assert.NoError(t, err)
	assert.Nil(t, ids)
}

func TestRegistryChildrenType(t *testing.T) {
	reg := newRegistry(&testClient{})
	_, err := reg.Children(&api.Step{Type: "unknown"})
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

func TestRegistryCompensatorNil(t *testing.T) {
	reg := newRegistry(&testClient{})
	comp, err := reg.Compensator(&api.Step{Type: api.StepTypeScript})
	assert.NoError(t, err)
	assert.Nil(t, comp)
}

func TestRegistryCompensatorType(t *testing.T) {
	reg := newRegistry(&testClient{})
	_, err := reg.Compensator(&api.Step{Type: "unknown"})
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

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
