package script_test

import (
	"testing"

	"github.com/kode4food/argex"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestArgExCompile(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangArgyll,
		Script:   "input.foo",
	})
	assert.NoError(t, err)
	assert.NotNil(t, compiled)
	assert.NoError(t, env.Validate(nil, "input.foo"))
	for _, source := range []string{"value[", "unknown()", "length(1)"} {
		_, err := env.Compile(nil, &api.ScriptConfig{Script: source})
		assert.ErrorIs(t, err, script.ErrArgExCompile)
	}
}

func TestArgExPredicate(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Script: "flag ?? false",
	})
	assert.NoError(t, err)
	for _, value := range []any{true, false, nil} {
		passed, err := evalPredicate(env, compiled, nil, api.Args{
			"flag": value,
		})
		assert.NoError(t, err)
		assert.Equal(t, value == true, passed)
	}
	passed, err := evalPredicate(env, compiled, nil, api.Args{})
	assert.NoError(t, err)
	assert.False(t, passed)
}

func TestArgExPredicateInputs(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Script: `product_info.name == "Professional Laptop"`,
	})
	assert.NoError(t, err)
	for _, name := range []string{"Professional Laptop", "Basic Laptop"} {
		passed, err := evalPredicate(env, compiled, nil, api.Args{
			"product_info": map[string]any{"name": name},
		})
		assert.NoError(t, err)
		assert.Equal(t, name == "Professional Laptop", passed)
	}
}

func TestArgExPredicateType(t *testing.T) {
	env := script.NewArgExEnv()
	for _, source := range []string{"null", "1", `"x"`, "[]", "{}"} {
		compiled, err := env.Compile(nil, &api.ScriptConfig{Script: source})
		assert.NoError(t, err)
		_, err = evalPredicate(env, compiled, nil, nil)
		assert.ErrorIs(t, err, script.ErrNotBoolean)
	}
	compiled, err := env.Compile(nil, &api.ScriptConfig{Script: "missing"})
	assert.NoError(t, err)
	_, err = evalPredicate(env, compiled, nil, nil)
	assert.ErrorIs(t, err, script.ErrArgExNoValue)
	compiled, err = env.Compile(nil, &api.ScriptConfig{Script: "1/0"})
	assert.NoError(t, err)
	_, err = evalPredicate(env, compiled, nil, nil)
	assert.ErrorIs(t, err, argex.ErrArithmetic)
}

func TestArgExCheckPredicate(t *testing.T) {
	env := script.NewArgExEnv()
	for _, source := range []string{
		"1 + 2", `"x"`, "[]", "{}", "null", "length", "length([])",
	} {
		compiled, err := env.Compile(nil, &api.ScriptConfig{Script: source})
		if !assert.NoError(t, err) {
			return
		}
		err = env.CheckPredicate(compiled)
		assert.ErrorIs(t, err, script.ErrNotBoolean)
	}
	for _, source := range []string{
		"flag ?? false", "value.ok", `case { a: true, 1 }`, "1 < 2",
		"let size = length: size([1]) == 1", "max(...[1, 2]) == 2",
	} {
		compiled, err := env.Compile(nil, &api.ScriptConfig{
			Script: source,
		})
		if !assert.NoError(t, err) {
			return
		}
		assert.NoError(t, env.CheckPredicate(compiled))
	}
}

func TestArgExMatch(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(script.MatchStep, &api.ScriptConfig{
		Script: `value?.kind == "email"`,
	})
	assert.NoError(t, err)
	for _, kind := range []string{"email", "postal"} {
		matched, err := evalMatch(env, compiled, api.Args{"kind": kind})
		assert.NoError(t, err)
		assert.Equal(t, kind == "email", matched)
	}
	matched, err := evalMatch(env, compiled, nil)
	assert.NoError(t, err)
	assert.False(t, matched)
	compiled, err = env.Compile(script.MatchStep, &api.ScriptConfig{
		Script: "value == null",
	})
	assert.NoError(t, err)
	matched, err = evalMatch(env, compiled, nil)
	assert.NoError(t, err)
	assert.True(t, matched)
}

func TestArgExScript(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Script: "input.foo",
	})
	assert.NoError(t, err)
	outputs, err := evalScript(env, compiled, nil, api.Args{
		"input": api.Args{"foo": "bar"},
	})
	assert.NoError(t, err)
	assert.Equal(t, api.Args{"result": "bar"}, outputs)
}

func TestArgExScriptObject(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Script: "{foo: input.foo, n: 1}",
	})
	assert.NoError(t, err)
	outputs, err := evalScript(env, compiled, nil, api.Args{
		"input": api.Args{"foo": "bar"},
	})
	assert.NoError(t, err)
	assert.Equal(t, api.Args{"foo": "bar", "n": 1.0}, outputs)
}

func TestArgExFunctionValues(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Script: `let size = length, count = size(input.items):
			{count, peak: max(...input.items)}`,
	})
	if !assert.NoError(t, err) {
		return
	}
	outputs, err := evalScript(env, compiled, nil, api.Args{
		"input": api.Args{"items": []any{1.0, 3.0}},
	})
	assert.NoError(t, err)
	assert.Equal(t, api.Args{"count": 2.0, "peak": 3.0}, outputs)
}

func TestArgExScriptArray(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Script: "output.books | map item: item.book",
	})
	assert.NoError(t, err)
	for _, books := range [][]any{
		{}, {api.Args{"book": "A"}},
		{api.Args{"book": "A"}, api.Args{"book": "B"}},
	} {
		outputs, err := evalScript(env, compiled, nil, api.Args{
			"output": api.Args{"books": books},
		})
		assert.NoError(t, err)
		want := make([]any, len(books))
		for i, book := range books {
			want[i] = book.(api.Args)["book"]
		}
		assert.Equal(t, want, outputs["result"])
	}
}

func TestArgExScriptMissing(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Script: "input?.missing",
	})
	assert.NoError(t, err)
	outputs, err := evalScript(env, compiled, nil, api.Args{
		"input": api.Args{"foo": "bar"},
	})
	assert.ErrorIs(t, err, script.ErrArgExNoValue)
	assert.Nil(t, outputs)
	compiled, err = env.Compile(nil, &api.ScriptConfig{Script: "1/0"})
	assert.NoError(t, err)
	_, err = evalScript(env, compiled, nil, nil)
	assert.ErrorIs(t, err, argex.ErrArithmetic)
}

func TestArgExScriptNull(t *testing.T) {
	env := script.NewArgExEnv()
	compiled, err := env.Compile(nil, &api.ScriptConfig{
		Script: "input?.missing ?? null",
	})
	assert.NoError(t, err)
	outputs, err := evalScript(env, compiled, nil, api.Args{})
	assert.NoError(t, err)
	assert.Equal(t, api.Args{"result": nil}, outputs)
}
