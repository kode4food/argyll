package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type registryTestEnv struct{}

func (r registryTestEnv) Validate(*api.Step, string) error {
	return nil
}

func (r registryTestEnv) Compile(
	*api.Step, *api.ScriptConfig,
) (engine.Compiled, error) {
	return "compiled", nil
}

func (r registryTestEnv) ExecuteScript(
	engine.Compiled, *api.Step, api.Args,
) (api.Args, error) {
	return api.Args{}, nil
}

func (r registryTestEnv) EvaluatePredicate(
	engine.Compiled, *api.Step, api.Args,
) (bool, error) {
	return true, nil
}

func TestAleCompilation(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := &api.Step{
			ID:   "ale-step",
			Name: "Ale Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "{:result (* x 2)}",
			},
			Attributes: api.AttributeSpecs{
				"x":      {Role: api.RoleRequired, Type: api.TypeNumber},
				"result": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
		}

		env, err := registry.Get(api.ScriptLangAle)
		assert.NoError(t, err)

		comp, err := env.Compile(step, step.Script)
		assert.NoError(t, err)
		assert.NotNil(t, comp)

		inputs := api.Args{"x": float64(21)}
		outputs, err := env.ExecuteScript(comp, step, inputs)
		assert.NoError(t, err)
		assert.Equal(t, float64(42), outputs["result"])
	})
}

func TestLuaCompilation(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := &api.Step{
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

		comp, err := env.Compile(step, step.Script)
		assert.NoError(t, err)
		assert.NotNil(t, comp)

		inputs := api.Args{"x": float64(21)}
		outputs, err := env.ExecuteScript(comp, step, inputs)
		assert.NoError(t, err)
		assert.Equal(t, 42, outputs["result"])
	})
}

func TestAlePredicateTrue(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := &api.Step{
			ID:   "pred-step",
			Name: "Predicate Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "(> x 10)",
			},
			Attributes: api.AttributeSpecs{
				"x": {Role: api.RoleRequired, Type: api.TypeNumber},
			},
		}

		env, err := registry.Get(api.ScriptLangAle)
		assert.NoError(t, err)

		comp, err := env.Compile(step, step.Predicate)
		assert.NoError(t, err)

		inputs := api.Args{"x": float64(15)}
		result, err := env.EvaluatePredicate(comp, step, inputs)
		assert.NoError(t, err)
		assert.True(t, result)
	})
}

func TestAlePredicateFalse(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := &api.Step{
			ID:   "pred-step",
			Name: "Predicate Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "(> x 10)",
			},
			Attributes: api.AttributeSpecs{
				"x": {Role: api.RoleRequired, Type: api.TypeNumber},
			},
		}

		env, err := registry.Get(api.ScriptLangAle)
		assert.NoError(t, err)

		comp, err := env.Compile(step, step.Predicate)
		assert.NoError(t, err)

		inputs := api.Args{"x": float64(5)}
		result, err := env.EvaluatePredicate(comp, step, inputs)
		assert.NoError(t, err)
		assert.False(t, result)
	})
}

func TestLuaPredicateTrue(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := &api.Step{
			ID:   "lua-pred-step",
			Name: "Lua Predicate Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
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

		comp, err := env.Compile(step, step.Predicate)
		assert.NoError(t, err)

		inputs := api.Args{"x": float64(15)}
		result, err := env.EvaluatePredicate(comp, step, inputs)
		assert.NoError(t, err)
		assert.True(t, result)
	})
}

func TestLuaPredicateFalse(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := &api.Step{
			ID:   "lua-pred-step",
			Name: "Lua Predicate Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test:8080",
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

		comp, err := env.Compile(step, step.Predicate)
		assert.NoError(t, err)

		inputs := api.Args{"x": float64(5)}
		result, err := env.EvaluatePredicate(comp, step, inputs)
		assert.NoError(t, err)
		assert.False(t, result)
	})
}

func TestUnsupportedLanguage(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		_, err := registry.Get("python")
		assert.ErrorIs(t, err, api.ErrInvalidScriptLanguage)
	})
}

func TestCompileViaRegistry(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		aleStep := helpers.NewScriptStep(
			"ale-step", api.ScriptLangAle, "{:result 42}", "result",
		)

		luaStep := helpers.NewScriptStep(
			"lua-step", api.ScriptLangLua, "return {result = 99}", "result",
		)

		httpStepPred := helpers.NewStepWithPredicate(
			"http-step", api.ScriptLangAle, "true",
		)

		aleComp, err := registry.Compile(aleStep, aleStep.Script)
		assert.NoError(t, err)
		assert.NotNil(t, aleComp)

		luaComp, err := registry.Compile(luaStep, luaStep.Script)
		assert.NoError(t, err)
		assert.NotNil(t, luaComp)

		predComp, err := registry.Compile(httpStepPred, httpStepPred.Predicate)
		assert.NoError(t, err)
		assert.NotNil(t, predComp)
	})
}

func TestAleComplexScript(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := &api.Step{
			ID:   "complex-ale",
			Name: "Complex Ale",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script: `{
				:sum (+ a b)
				:product (* a b)
				:greeting (str "Hello " name)
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

		env, err := registry.Get(api.ScriptLangAle)
		assert.NoError(t, err)

		comp, err := env.Compile(step, step.Script)
		assert.NoError(t, err)

		inputs := api.Args{
			"a":    float64(10),
			"b":    float64(5),
			"name": "World",
		}

		outputs, err := env.ExecuteScript(comp, step, inputs)
		assert.NoError(t, err)
		assert.Equal(t, float64(15), outputs["sum"])
		assert.Equal(t, float64(50), outputs["product"])
		assert.Equal(t, "Hello World", outputs["greeting"])
	})
}

func TestLuaComplexScript(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := &api.Step{
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

		comp, err := env.Compile(step, step.Script)
		assert.NoError(t, err)

		inputs := api.Args{
			"a":    float64(10),
			"b":    float64(5),
			"name": "World",
		}

		outputs, err := env.ExecuteScript(comp, step, inputs)
		assert.NoError(t, err)
		assert.Equal(t, 15, outputs["sum"])
		assert.Equal(t, 50, outputs["product"])
		assert.Equal(t, "Hello World", outputs["greeting"])
	})
}

func TestAleInvalidSyntax(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := helpers.NewScriptStep(
			"invalid-ale", api.ScriptLangAle, "{:result (+ 1 2",
		)

		env, err := registry.Get(api.ScriptLangAle)
		assert.NoError(t, err)

		_, err = env.Compile(step, step.Script)
		assert.Error(t, err)
	})
}

func TestLuaInvalidSyntax(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		step := helpers.NewScriptStep(
			"invalid-lua", api.ScriptLangLua, "return {result = ",
		)

		env, err := registry.Get(api.ScriptLangLua)
		assert.NoError(t, err)

		_, err = env.Compile(step, step.Script)
		assert.Error(t, err)
	})
}

func TestScriptRegistryRegister(t *testing.T) {
	registry := engine.NewScriptRegistry()
	env := registryTestEnv{}

	registry.Register("test", env)

	got, err := registry.Get("test")
	assert.NoError(t, err)
	assert.Equal(t, env, got)
}
