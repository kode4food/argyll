package script_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type registryTestEnv struct{}

func (r registryTestEnv) Validate(*api.Step, string) error {
	return nil
}

func (r registryTestEnv) Compile(
	*api.Step, *api.ScriptConfig,
) (script.Compiled, error) {
	return "compiled", nil
}

func (r registryTestEnv) Evaluate(
	script.Compiled, *api.Step, api.Args,
) (any, error) {
	return true, nil
}

func TestRegistryCompilation(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := &api.Step{
			ID:   "registry-step",
			Name: "Registry Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {result = x * 2}",
			},
			Attributes: api.AttributeSpecs{
				"x":      {Role: api.RoleRequired, Type: api.TypeNumber},
				"result": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
		}

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Script)
		assert.NoError(t, err)
		assert.NotNil(t, comp)

		inputs := api.Args{"x": float64(21)}
		outputs, err := evalScript(env, comp, st, inputs)
		assert.NoError(t, err)
		assert.Equal(t, 42, outputs["result"])
	})
}

func TestLuaCompilation(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := &api.Step{
			ID:   "lua-step",
			Name: "Lua Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {result = x * 2}",
			},
			Attributes: api.AttributeSpecs{
				"x":      {Role: api.RoleRequired, Type: api.TypeNumber},
				"result": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
		}

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Script)
		assert.NoError(t, err)
		assert.NotNil(t, comp)

		inputs := api.Args{"x": float64(21)}
		outputs, err := evalScript(env, comp, st, inputs)
		assert.NoError(t, err)
		assert.Equal(t, 42, outputs["result"])
	})
}

func TestRegistryPredicateTrue(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := &api.Step{
			ID:   "pred-step",
			Name: "Predicate Step",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return x > 10",
			},
			Attributes: api.AttributeSpecs{
				"x": {Role: api.RoleRequired, Type: api.TypeNumber},
			},
		}

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Predicate)
		assert.NoError(t, err)

		inputs := api.Args{"x": float64(15)}
		result, err := evalPredicate(env, comp, st, inputs)
		assert.NoError(t, err)
		assert.True(t, result)
	})
}

func TestRegistryPredicateFalse(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := &api.Step{
			ID:   "pred-step",
			Name: "Predicate Step",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return x > 10",
			},
			Attributes: api.AttributeSpecs{
				"x": {Role: api.RoleRequired, Type: api.TypeNumber},
			},
		}

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Predicate)
		assert.NoError(t, err)

		inputs := api.Args{"x": float64(5)}
		result, err := evalPredicate(env, comp, st, inputs)
		assert.NoError(t, err)
		assert.False(t, result)
	})
}

func TestLuaPredicateTrue(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := &api.Step{
			ID:   "lua-pred-step",
			Name: "Lua Predicate Step",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return x > 10",
			},
			Attributes: api.AttributeSpecs{
				"x": {Role: api.RoleRequired, Type: api.TypeNumber},
			},
		}

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Predicate)
		assert.NoError(t, err)

		inputs := api.Args{"x": float64(15)}
		result, err := evalPredicate(env, comp, st, inputs)
		assert.NoError(t, err)
		assert.True(t, result)
	})
}

func TestLuaPredicateFalse(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := &api.Step{
			ID:   "lua-pred-step",
			Name: "Lua Predicate Step",
			Type: api.StepTypeService,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test:8080"},
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return x > 10",
			},
			Attributes: api.AttributeSpecs{
				"x": {Role: api.RoleRequired, Type: api.TypeNumber},
			},
		}

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Predicate)
		assert.NoError(t, err)

		inputs := api.Args{"x": float64(5)}
		result, err := evalPredicate(env, comp, st, inputs)
		assert.NoError(t, err)
		assert.False(t, result)
	})
}

func TestArgExPredicateMatches(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := helpers.NewStepWithPredicate(
			"argex-pred-step", api.ScriptLangArgyll, "flag == null",
		)
		st.Attributes["flag"] = &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeAny,
		}

		env, err := registry.Get(api.ScriptLangArgyll)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Predicate)
		assert.NoError(t, err)

		passed, err := evalPredicate(env, comp, st, api.Args{
			"flag": nil,
		})
		assert.NoError(t, err)
		assert.True(t, passed)
	})
}

func TestArgExPredicateNoMatch(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := helpers.NewStepWithPredicate(
			"argex-pred-step", api.ScriptLangArgyll, "flag ?? false",
		)

		env, err := registry.Get(api.ScriptLangArgyll)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Predicate)
		assert.NoError(t, err)

		passed, err := evalPredicate(env, comp, st, api.Args{})
		assert.NoError(t, err)
		assert.False(t, passed)
	})
}

func TestArgExInvalidSyntax(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := helpers.NewStepWithPredicate(
			"argex-invalid", api.ScriptLangArgyll, "value[",
		)

		env, err := registry.Get(api.ScriptLangArgyll)
		assert.NoError(t, err)

		_, err = env.Compile(st, st.Predicate)
		assert.Error(t, err)
		assert.ErrorIs(t, err, script.ErrArgExCompile)
	})
}

func TestArgExEvaluateScript(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		env, err := registry.Get(api.ScriptLangArgyll)
		assert.NoError(t, err)

		st := helpers.NewStepWithPredicate(
			"argex-exec", api.ScriptLangArgyll, "input.foo",
		)

		comp, err := env.Compile(st, st.Predicate)
		assert.NoError(t, err)

		outputs, err := evalScript(env, comp, nil, api.Args{
			"input": map[string]any{"foo": "bar"},
		})
		assert.NoError(t, err)
		assert.Equal(t, "bar", outputs["result"])
	})
}

func TestUnsupportedLanguage(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		_, err := registry.Get("python")
		assert.ErrorIs(t, err, api.ErrInvalidScriptLanguage)
	})
}

func TestCompileViaRegistry(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		firstStep := helpers.NewScriptStep(
			"first-step", api.ScriptLangLua, "return {result = 42}", "result",
		)

		luaStep := helpers.NewScriptStep(
			"lua-step", api.ScriptLangLua, "return {result = 99}", "result",
		)

		httpStepPred := helpers.NewStepWithPredicate(
			"http-step", api.ScriptLangLua, "return true",
		)

		firstComp, err := registry.Compile(firstStep, firstStep.Script)
		assert.NoError(t, err)
		assert.NotNil(t, firstComp)

		luaComp, err := registry.Compile(luaStep, luaStep.Script)
		assert.NoError(t, err)
		assert.NotNil(t, luaComp)

		predComp, err := registry.Compile(httpStepPred, httpStepPred.Predicate)
		assert.NoError(t, err)
		assert.NotNil(t, predComp)
	})
}

func TestEmptyScript(t *testing.T) {
	for _, language := range []string{api.ScriptLangLua, api.ScriptLangArgyll} {
		t.Run(language, func(t *testing.T) {
			r := script.NewRegistry()
			_, err := r.Compile(
				&api.Step{}, &api.ScriptConfig{Language: language},
			)
			assert.ErrorIs(t, err, api.ErrScriptEmpty)
		})
	}
}

func TestRegistryComplexScript(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := &api.Step{
			ID:   "complex-registry",
			Name: "Complex Registry Script",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script: `return {
				sum = a + b,
				product = a * b,
				greeting = "Hello " .. name
			}`,
			},
			Attributes: api.AttributeSpecs{
				"a":        {Role: api.RoleRequired, Type: api.TypeNumber},
				"b":        {Role: api.RoleRequired, Type: api.TypeNumber},
				"name":     {Role: api.RoleRequired, Type: api.TypeString},
				"sum":      {Role: api.RoleOutput, Type: api.TypeNumber},
				"product":  {Role: api.RoleOutput, Type: api.TypeNumber},
				"greeting": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Script)
		assert.NoError(t, err)

		inputs := api.Args{
			"a":    float64(10),
			"b":    float64(5),
			"name": "World",
		}

		outputs, err := evalScript(env, comp, st, inputs)
		assert.NoError(t, err)
		assert.Equal(t, 15, outputs["sum"])
		assert.Equal(t, 50, outputs["product"])
		assert.Equal(t, "Hello World", outputs["greeting"])
	})
}

func TestLuaComplexScript(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := &api.Step{
			ID:   "complex-lua",
			Name: "Complex Lua",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script: `
				return {
					sum = a + b,
					product = a * b,
					greeting = "Hello " .. name
				}
			`,
			},
			Attributes: api.AttributeSpecs{
				"a":        {Role: api.RoleRequired, Type: api.TypeNumber},
				"b":        {Role: api.RoleRequired, Type: api.TypeNumber},
				"name":     {Role: api.RoleRequired, Type: api.TypeString},
				"sum":      {Role: api.RoleOutput, Type: api.TypeNumber},
				"product":  {Role: api.RoleOutput, Type: api.TypeNumber},
				"greeting": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		comp, err := env.Compile(st, st.Script)
		assert.NoError(t, err)

		inputs := api.Args{
			"a":    float64(10),
			"b":    float64(5),
			"name": "World",
		}

		outputs, err := evalScript(env, comp, st, inputs)
		assert.NoError(t, err)
		assert.Equal(t, 15, outputs["sum"])
		assert.Equal(t, 50, outputs["product"])
		assert.Equal(t, "Hello World", outputs["greeting"])
	})
}

func TestRegistryInvalidSyntax(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := helpers.NewScriptStep(
			"invalid-registry", api.ScriptLangLua, "return {result = ",
		)

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		_, err = env.Compile(st, st.Script)
		assert.Error(t, err)
	})
}

func TestLuaInvalidSyntax(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := script.NewRegistry()

		st := helpers.NewScriptStep(
			"invalid-lua", api.ScriptLangLua, "return {result = ",
		)

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		_, err = env.Compile(st, st.Script)
		assert.Error(t, err)
	})
}

func TestScriptRegistryRegister(t *testing.T) {
	registry := script.NewRegistry()
	env := registryTestEnv{}

	registry.Register("test", env)

	got, err := registry.Get("test")
	assert.NoError(t, err)
	assert.Equal(t, env, got)
}

func evalScript(
	env script.Environment, c script.Compiled, st *api.Step, args api.Args,
) (api.Args, error) {
	res, err := env.Evaluate(c, st, args)
	if err != nil {
		return nil, err
	}
	return script.Outputs(res), nil
}

func evalPredicate(
	env script.Environment, c script.Compiled, st *api.Step, args api.Args,
) (bool, error) {
	res, err := env.Evaluate(c, st, args)
	if err != nil {
		return false, err
	}
	return script.Bool(res)
}

func evalMatch(
	env script.Environment, c script.Compiled, value any,
) (bool, error) {
	return evalPredicate(
		env, c, script.MatchStep, api.Args{script.MatchValue: value},
	)
}
