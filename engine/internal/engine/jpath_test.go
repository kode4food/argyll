package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestJPathCompileAndValidate(t *testing.T) {
	env := engine.NewJPathEnv()

	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   "$.foo",
	})
	assert.NoError(t, err)
	assert.NotNil(t, compiled)

	err = env.Validate(nil, "$.foo")
	assert.NoError(t, err)
}

func TestJPathCompileInvalid(t *testing.T) {
	env := engine.NewJPathEnv()

	_, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   "$..[",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, engine.ErrJPathCompile)
}

func TestJPathEvaluatePredicate(t *testing.T) {
	env := engine.NewJPathEnv()

	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   "$.flag",
	})
	assert.NoError(t, err)
	assert.NotNil(t, compiled)

	passed, err := env.EvaluatePredicate(compiled, nil, api.Args{
		"flag": true,
	})
	assert.NoError(t, err)
	assert.True(t, passed)

	passed, err = env.EvaluatePredicate(compiled, nil, api.Args{})
	assert.NoError(t, err)
	assert.False(t, passed)
}

func TestJPathEvaluatePredicateTopLevelFilter(t *testing.T) {
	env := engine.NewJPathEnv()

	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   `$.product_info.name == "Professional Laptop"`,
	})
	assert.NoError(t, err)
	assert.NotNil(t, compiled)

	passed, err := env.EvaluatePredicate(compiled, nil, api.Args{
		"product_info": map[string]any{
			"name": "Professional Laptop",
			"sku":  "123",
		},
	})
	assert.NoError(t, err)
	assert.True(t, passed)

	passed, err = env.EvaluatePredicate(compiled, nil, api.Args{
		"product_info": map[string]any{
			"name": "Basic Laptop",
			"sku":  "123",
		},
	})
	assert.NoError(t, err)
	assert.False(t, passed)
}

func TestJPathEvaluatePredicateNullMatch(t *testing.T) {
	env := engine.NewJPathEnv()

	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   "$.flag",
	})
	assert.NoError(t, err)

	passed, err := env.EvaluatePredicate(compiled, nil, api.Args{
		"flag": nil,
	})
	assert.NoError(t, err)
	assert.True(t, passed)
}

func TestJPathEvaluatePredicateBadCompiledType(t *testing.T) {
	env := engine.NewJPathEnv()

	passed, err := env.EvaluatePredicate("not-compiled", nil, api.Args{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, engine.ErrJPathBadCompiledType)
	assert.False(t, passed)
}

func TestJPathExecUnsupported(t *testing.T) {
	env := engine.NewJPathEnv()

	outputs, err := env.ExecuteScript(nil, nil, nil)
	assert.ErrorIs(t, err, engine.ErrJPathExecuteScript)
	assert.Nil(t, outputs)
}
