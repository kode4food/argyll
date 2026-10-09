package builtins_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

func TestScriptValidationNil(t *testing.T) {
	reg := newRegistry(&testClient{})
	err := reg.Validate(&api.Step{Type: api.StepTypeScript})
	assert.ErrorIs(t, err, api.ErrScriptRequired)
}

func TestScriptValidationJPath(t *testing.T) {
	reg := newRegistry(&testClient{})
	err := reg.Validate(&api.Step{
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangJPath,
			Script:   "$.x",
		},
	})
	assert.ErrorIs(t, err, builtins.ErrLangNotValid)
}

func TestScriptValidationLua(t *testing.T) {
	reg := newRegistry(&testClient{})
	err := reg.Validate(&api.Step{
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return {result = 42}",
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
			Language: api.ScriptLangLua,
			Script:   "return {result = 42}",
		},
		Attributes: api.AttributeSpecs{
			"result": {Role: api.RoleOutput},
		},
	}
	h, err := reg.Health(st)
	assert.NoError(t, err)
	assert.Equal(t, api.HealthHealthy, h.Status)
}

func TestScriptHealthInvalid(t *testing.T) {
	reg := newRegistry(&testClient{})
	st := &api.Step{
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "!!!bad",
		},
	}
	h, err := reg.Health(st)
	assert.NoError(t, err)
	assert.Equal(t, api.HealthUnhealthy, h.Status)
	assert.NotEmpty(t, h.Error)
}

func TestScriptOutput(t *testing.T) {
	reg := newRegistry(&testClient{})
	handler, err := reg.Lookup(api.StepTypeScript)
	assert.NoError(t, err)

	rt, calls := newRuntime("flow-1", "step-1", nil)
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return {result = 42}",
		},
		Attributes: api.AttributeSpecs{
			"result": {Role: api.RoleOutput},
		},
	}

	err = handler.Invoke(rt, st, api.Args{}, "token-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, calls.completeCalls)
	assert.Equal(t, api.Args{"result": 42}, calls.completeOut)
	assert.Equal(t, api.HealthHealthy, calls.healthStatus)
}

func TestScriptFailure(t *testing.T) {
	reg := newRegistry(&testClient{})
	handler, err := reg.Lookup(api.StepTypeScript)
	assert.NoError(t, err)

	rt, calls := newRuntime("flow-1", "step-1", nil)
	st := &api.Step{
		ID:   "step-1",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "!!!bad",
		},
	}

	err = handler.Invoke(rt, st, api.Args{}, "token-1")
	assert.Error(t, err)
	assert.ErrorIs(t, err, builtins.ErrScriptCompileFailed)
	assert.Equal(t, api.HealthUnhealthy, calls.healthStatus)
	assert.NotEmpty(t, calls.healthError)
}
