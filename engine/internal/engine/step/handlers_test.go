package step_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/step"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRegistryValidateNilHandler(t *testing.T) {
	reg := newRegistry(&testClient{})
	// sync handler has no Validate func -- should return nil
	err := reg.Validate(&api.Step{
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{Endpoint: "http://example.test"},
	})
	assert.NoError(t, err)
}

func TestRegistryHealthUnknownWhenNilHandler(t *testing.T) {
	reg := newRegistry(&testClient{})
	// sync handler has no Health func -- should return HealthUnknown
	h, err := reg.Health(&api.Step{Type: api.StepTypeSync})
	assert.NoError(t, err)
	assert.Equal(t, api.HealthUnknown, h.Status)
}

func TestRegistryHealthUnknownType(t *testing.T) {
	reg := newRegistry(&testClient{})
	_, err := reg.Health(&api.Step{Type: "unknown"})
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

func TestRegistryChildrenNilWhenNoHandler(t *testing.T) {
	reg := newRegistry(&testClient{})
	// sync handler has no Children func -- should return nil
	ids, err := reg.Children(&api.Step{Type: api.StepTypeSync})
	assert.NoError(t, err)
	assert.Nil(t, ids)
}

func TestRegistryChildrenUnknownType(t *testing.T) {
	reg := newRegistry(&testClient{})
	_, err := reg.Children(&api.Step{Type: "unknown"})
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

func TestRegistryCompensatorNilWhenNoHandler(t *testing.T) {
	reg := newRegistry(&testClient{})
	comp, err := reg.Compensator(&api.Step{Type: api.StepTypeScript})
	assert.NoError(t, err)
	assert.Nil(t, comp)
}

func TestRegistryCompensatorUnknownType(t *testing.T) {
	reg := newRegistry(&testClient{})
	_, err := reg.Compensator(&api.Step{Type: "unknown"})
	assert.ErrorIs(t, err, api.ErrInvalidStepType)
}

func TestFlowChildrenNilFlow(t *testing.T) {
	reg := newRegistry(&testClient{})
	ids, err := reg.Children(&api.Step{Type: api.StepTypeFlow})
	assert.NoError(t, err)
	assert.Nil(t, ids)
}

func TestFlowChildrenReturnsGoals(t *testing.T) {
	reg := newRegistry(&testClient{})
	st := &api.Step{
		Type: api.StepTypeFlow,
		Flow: &api.FlowConfig{Goals: []api.StepID{"a", "b"}},
	}
	ids, err := reg.Children(st)
	assert.NoError(t, err)
	assert.Equal(t, []api.StepID{"a", "b"}, ids)
}

func TestHTTPCompensatorInvokes(t *testing.T) {
	cl := &testClient{}
	reg := newRegistry(cl)
	comp, err := reg.Compensator(&api.Step{Type: api.StepTypeSync})
	assert.NoError(t, err)
	assert.NotNil(t, comp)
	err = comp(
		&api.Step{Type: api.StepTypeSync},
		api.Args{"in": "v"}, api.Args{"out": "v"}, api.Metadata{},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, cl.compens)
}

func TestApplyMetaInputsMapsAttribute(t *testing.T) {
	cl := &testClient{outputs: api.Args{}}
	reg := newRegistry(cl)
	handler, err := reg.Lookup(api.StepTypeSync)
	assert.NoError(t, err)

	rt, _ := newRuntime("flow-1", "step-1", api.Metadata{}, "")
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{Endpoint: "http://example.test"},
		Attributes: api.AttributeSpecs{
			"token": {
				Role: api.RoleMeta,
				Meta: &api.MetaConfig{Key: api.MetaReceiptToken},
			},
		},
	}

	err = handler.Execute(rt, st, api.Args{}, "my-token")
	assert.NoError(t, err)
	assert.Equal(t, api.Token("my-token"), cl.inputs[api.Name("token")])
}

func TestScriptValidatorRejectsNilScript(t *testing.T) {
	reg := newRegistry(&testClient{})
	err := reg.Validate(&api.Step{Type: api.StepTypeScript})
	assert.ErrorIs(t, err, api.ErrScriptRequired)
}

func TestScriptValidatorRejectsJPath(t *testing.T) {
	reg := newRegistry(&testClient{})
	err := reg.Validate(&api.Step{
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   "$.x",
		},
	})
	assert.ErrorIs(t, err, step.ErrLangNotValid)
}

func TestScriptValidatorAcceptsValidAle(t *testing.T) {
	reg := newRegistry(&testClient{})
	err := reg.Validate(&api.Step{
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangAle,
			Script:   "{:result 42}",
		},
		Attributes: api.AttributeSpecs{
			"result": {Role: api.RoleOutput},
		},
	})
	assert.NoError(t, err)
}

func TestScriptHealthHealthy(t *testing.T) {
	reg := newRegistry(&testClient{})
	st := &api.Step{
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangAle,
			Script:   "{:result 42}",
		},
		Attributes: api.AttributeSpecs{
			"result": {Role: api.RoleOutput},
		},
	}
	h, err := reg.Health(st)
	assert.NoError(t, err)
	assert.Equal(t, api.HealthHealthy, h.Status)
}

func TestScriptHealthUnhealthyOnBadScript(t *testing.T) {
	reg := newRegistry(&testClient{})
	st := &api.Step{
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangAle,
			Script:   "!!!bad",
		},
	}
	h, err := reg.Health(st)
	assert.NoError(t, err)
	assert.Equal(t, api.HealthUnhealthy, h.Status)
	assert.NotEmpty(t, h.Error)
}

func TestScriptExecutorProducesOutput(t *testing.T) {
	reg := newRegistry(&testClient{})
	handler, err := reg.Lookup(api.StepTypeScript)
	assert.NoError(t, err)

	rt, calls := newRuntime("flow-1", "step-1", nil, "")
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangAle,
			Script:   "{:result 42}",
		},
		Attributes: api.AttributeSpecs{
			"result": {Role: api.RoleOutput},
		},
	}

	err = handler.Execute(rt, st, api.Args{}, "token-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, calls.completeCalls)
	assert.Equal(t, api.Args{"result": 42}, calls.completeOut)
	assert.Equal(t, api.HealthHealthy, calls.healthStatus)
}

func TestScriptExecutorMarksUnhealthyOnBadScript(t *testing.T) {
	reg := newRegistry(&testClient{})
	handler, err := reg.Lookup(api.StepTypeScript)
	assert.NoError(t, err)

	rt, calls := newRuntime("flow-1", "step-1", nil, "")
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangAle,
			Script:   "!!!bad",
		},
	}

	err = handler.Execute(rt, st, api.Args{}, "token-1")
	assert.Error(t, err)
	assert.ErrorIs(t, err, step.ErrScriptCompileFailed)
	assert.Equal(t, api.HealthUnhealthy, calls.healthStatus)
	assert.NotEmpty(t, calls.healthError)
}
