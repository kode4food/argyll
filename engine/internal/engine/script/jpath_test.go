package script_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestJPathCompileAndValidate(t *testing.T) {
	env := script.NewJPathEnv()

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
	env := script.NewJPathEnv()

	_, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   "$..[",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, script.ErrJPathCompile)
}

func TestJPathEvaluatePredicate(t *testing.T) {
	env := script.NewJPathEnv()

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
	env := script.NewJPathEnv()

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
	env := script.NewJPathEnv()

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

func TestJPathExecuteScriptSingleMatch(t *testing.T) {
	env := script.NewJPathEnv()

	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   "$.foo",
	})
	assert.NoError(t, err)

	outputs, err := env.ExecuteScript(compiled, nil, api.Args{
		"input": map[string]any{"foo": "bar"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "bar", outputs["value"])
}

func TestJPathExecuteScriptMultiMatch(t *testing.T) {
	env := script.NewJPathEnv()

	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   "$..book",
	})
	assert.NoError(t, err)

	outputs, err := env.ExecuteScript(compiled, nil, api.Args{
		"output": map[string]any{
			"books": []any{
				map[string]any{"book": "A"},
				map[string]any{"book": "B"},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, []any{"A", "B"}, outputs["value"])
}

func TestJPathExecuteScriptNoMatch(t *testing.T) {
	env := script.NewJPathEnv()

	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   "$.missing",
	})
	assert.NoError(t, err)

	outputs, err := env.ExecuteScript(compiled, nil, api.Args{
		"input": map[string]any{"foo": "bar"},
	})
	assert.ErrorIs(t, err, script.ErrJPathNoMatch)
	assert.Nil(t, outputs)
}
