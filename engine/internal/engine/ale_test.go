package engine_test

import (
	"testing"

	"github.com/kode4food/ale/data"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestAleCacheForSameScript(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		env := engine.NewAleEnv()

		step := &api.Step{
			ID:   "test-step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Script:   "{:result (+ a b)}",
				Language: api.ScriptLangAle,
			},
			Attributes: api.AttributeSpecs{
				"a":      {Role: api.RoleRequired},
				"b":      {Role: api.RoleRequired},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		proc1, err := env.Compile(step, step.Script)
		assert.NoError(t, err)

		proc2, err := env.Compile(step, step.Script)
		assert.NoError(t, err)

		// Verify scripts are cached by checking same object returned
		assert.Equal(t, proc1, proc2)
	})
}

func TestAleCacheIncludesArgNames(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		env := engine.NewAleEnv()

		stepOuter := &api.Step{
			ID:   "outer-step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Script:   "{:result (* amount 2)}",
				Language: api.ScriptLangAle,
			},
			Attributes: api.AttributeSpecs{
				"amount": {Role: api.RoleRequired, Type: api.TypeNumber},
				"result": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
		}

		stepMapped := &api.Step{
			ID:   "mapped-step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Script:   "{:result (* amount 2)}",
				Language: api.ScriptLangAle,
			},
			Attributes: api.AttributeSpecs{
				"amount": {
					Role: api.RoleRequired,
					Type: api.TypeNumber,
					Mapping: &api.AttributeMapping{
						Name: "inner_amount",
					},
				},
				"result": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
		}

		_, err := env.Compile(stepOuter, stepOuter.Script)
		assert.NoError(t, err)

		_, err = env.Compile(stepMapped, stepMapped.Script)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "amount")
	})
}

func TestAleCompileViaRegistry(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		registry := engine.NewScriptRegistry()

		script := &api.Step{
			ID:   "script-step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Script:   "{:result (* a b)}",
				Language: api.ScriptLangAle,
			},
			Attributes: api.AttributeSpecs{
				"a":      {Role: api.RoleRequired},
				"b":      {Role: api.RoleRequired},
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		pred := &api.Step{
			ID:   "predicate-step",
			Type: api.StepTypeSync,
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "(> x 10)",
			},
			Attributes: api.AttributeSpecs{
				"x":      {Role: api.RoleRequired},
				"output": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		scriptComp, err := registry.Compile(script, script.Script)
		assert.NoError(t, err)
		assert.NotNil(t, scriptComp)

		scriptProc, ok := scriptComp.(data.Procedure)
		assert.True(t, ok)
		assert.NotNil(t, scriptProc)

		predComp, err := registry.Compile(pred, pred.Predicate)
		assert.NoError(t, err)
		assert.NotNil(t, predComp)

		predProc, ok := predComp.(data.Procedure)
		assert.True(t, ok)
		assert.NotNil(t, predProc)
	})
}

func TestAleExecuteScript(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		env := engine.NewAleEnv()

		step := &api.Step{
			ID:   "test",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Script:   "{:result (+ a b)}",
				Language: api.ScriptLangAle,
			},
			Attributes: api.AttributeSpecs{
				"a":      {Role: api.RoleRequired},
				"b":      {Role: api.RoleRequired},
				"result": {Role: api.RoleRequired},
			},
		}

		proc, err := env.Compile(step, step.Script)
		assert.NoError(t, err)

		args := api.Args{
			"a": 5,
			"b": 10,
		}

		result, err := env.ExecuteScript(proc, step, args)
		assert.NoError(t, err)

		assert.Contains(t, result, api.Name("result"))
		assert.Equal(t, 15, result["result"])
	})
}

func TestAleEvaluatePredicate(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		env := engine.NewAleEnv()

		tests := []struct {
			name      string
			predicate string
			args      api.Args
			expected  bool
		}{
			{
				name:      "true_condition",
				predicate: "(> x 10)",
				args:      api.Args{"x": 15},
				expected:  true,
			},
			{
				name:      "false_condition",
				predicate: "(> x 10)",
				args:      api.Args{"x": 5},
				expected:  false,
			},
			{
				name:      "complex_condition",
				predicate: "(and (> x 10) (< y 20))",
				args:      api.Args{"x": 15, "y": 15},
				expected:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				step := &api.Step{
					ID:   "test",
					Type: api.StepTypeSync,
					Predicate: &api.ScriptConfig{
						Script:   tt.predicate,
						Language: api.ScriptLangAle,
					},
					Attributes: api.AttributeSpecs{
						"x": {Role: api.RoleRequired},
						"y": {Role: api.RoleRequired},
					},
				}

				cfg := &api.ScriptConfig{
					Script:   tt.predicate,
					Language: api.ScriptLangAle,
				}
				comp, err := env.Compile(step, cfg)
				assert.NoError(t, err)

				result, err := env.EvaluatePredicate(comp, step, tt.args)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestAleValidate(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		env := engine.NewAleEnv()

		tests := []struct {
			name        string
			script      string
			expectError bool
		}{
			{
				name:        "valid_script",
				script:      "{:result 42}",
				expectError: false,
			},
			{
				name:        "invalid_syntax",
				script:      "{:result",
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				step := &api.Step{ID: "test", Type: api.StepTypeScript}
				err := env.Validate(step, tt.script)
				if tt.expectError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
			})
		}
	})
}

func TestAleComplexConversion(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		env := engine.NewAleEnv()

		step := &api.Step{
			ID:   "complex-types",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script: `{
				:bool_val is_active
				:string_val name
				:int_val count
				:float_val price
				:array_val items
				:nested nested_obj
				:null_val optional
			}`,
			},
			Attributes: api.AttributeSpecs{
				"is_active":  {Role: api.RoleRequired},
				"name":       {Role: api.RoleRequired},
				"count":      {Role: api.RoleRequired},
				"price":      {Role: api.RoleRequired},
				"items":      {Role: api.RoleRequired},
				"nested_obj": {Role: api.RoleRequired},
				"optional":   {Role: api.RoleRequired},
				"bool_val":   {Role: api.RoleRequired},
				"string_val": {Role: api.RoleRequired},
				"int_val":    {Role: api.RoleRequired},
				"float_val":  {Role: api.RoleRequired},
				"array_val":  {Role: api.RoleRequired},
				"nested":     {Role: api.RoleRequired},
				"null_val":   {Role: api.RoleRequired},
			},
		}

		comp, err := env.Compile(step, step.Script)
		assert.NoError(t, err)

		args := api.Args{
			"is_active": true,
			"name":      "test-item",
			"count":     int64(42),
			"price":     99.99,
			"items":     []any{"item1", "item2", "item3"},
			"nested_obj": map[string]any{
				"key1": "value1",
				"key2": 123,
			},
			"optional": nil,
		}

		result, err := env.ExecuteScript(comp, step, args)
		assert.NoError(t, err)

		assert.Equal(t, true, result["bool_val"])
		assert.Equal(t, "test-item", result["string_val"])
		assert.Equal(t, 42, result["int_val"])
		assert.Equal(t, 99.99, result["float_val"])

		arrVal, ok := result["array_val"].([]any)
		assert.True(t, ok)
		assert.Len(t, arrVal, 3)
		assert.Equal(t, "item1", arrVal[0])

		nested, ok := result["nested"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "value1", nested["key1"])
		assert.Equal(t, 123, nested["key2"])

		assert.Nil(t, result["null_val"])
	})
}

func TestAleListConversion(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		env := engine.NewAleEnv()

		step := &api.Step{
			ID:   "list-test",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   `{:list_result (list 1 2 3 4 5)}`,
			},
			Attributes: api.AttributeSpecs{
				"list_result": {Role: api.RoleRequired},
			},
		}

		comp, err := env.Compile(step, step.Script)
		assert.NoError(t, err)

		result, err := env.ExecuteScript(comp, step, api.Args{})
		assert.NoError(t, err)

		listVal, ok := result["list_result"].([]any)
		assert.True(t, ok)
		assert.Len(t, listVal, 5)
		assert.Equal(t, 1, listVal[0])
		assert.Equal(t, 5, listVal[4])
	})
}
