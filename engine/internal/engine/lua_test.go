package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestLuaCompile(t *testing.T) {
	env := engine.NewLuaEnv()

	step := &api.Step{
		ID:   "test",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script:   "return {result = a + b}",
			Language: api.ScriptLangLua,
		},
		Attributes: api.AttributeSpecs{
			"a":      {Role: api.RoleRequired},
			"b":      {Role: api.RoleRequired},
			"result": {Role: api.RoleRequired},
		},
	}

	comp, err := env.Compile(step, step.Script)
	assert.NoError(t, err)
	assert.NotNil(t, comp)
}

func TestLuaExecuteScript(t *testing.T) {
	env := engine.NewLuaEnv()

	step := &api.Step{
		ID:   "test",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script:   "return {result = a + b}",
			Language: api.ScriptLangLua,
		},
		Attributes: api.AttributeSpecs{
			"a":      {Role: api.RoleRequired},
			"b":      {Role: api.RoleRequired},
			"result": {Role: api.RoleRequired},
		},
	}

	comp, err := env.Compile(step, step.Script)
	assert.NoError(t, err)

	args := api.Args{
		"a": 5,
		"b": 10,
	}

	result, err := env.ExecuteScript(comp, step, args)
	assert.NoError(t, err)

	assert.Contains(t, result, api.Name("result"))
	assert.Equal(t, 15, result["result"])
}

func TestLuaEvaluatePredicate(t *testing.T) {
	env := engine.NewLuaEnv()

	tests := []struct {
		name      string
		predicate string
		args      api.Args
		expected  bool
	}{
		{
			name:      "true_condition",
			predicate: "return x > 10",
			args:      api.Args{"x": 15},
			expected:  true,
		},
		{
			name:      "false_condition",
			predicate: "return x > 10",
			args:      api.Args{"x": 5},
			expected:  false,
		},
		{
			name:      "complex_condition",
			predicate: "return x > 10 and y < 20",
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
					Language: api.ScriptLangLua,
				},
				Attributes: api.AttributeSpecs{
					"x": {Role: api.RoleRequired},
					"y": {Role: api.RoleRequired},
				},
			}

			comp, err := env.Compile(step, &api.ScriptConfig{
				Script:   tt.predicate,
				Language: api.ScriptLangLua,
			})
			assert.NoError(t, err)

			result, err := env.EvaluatePredicate(comp, step, tt.args)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLuaValidate(t *testing.T) {
	env := engine.NewLuaEnv()

	tests := []struct {
		name        string
		script      string
		expectError bool
	}{
		{
			name:        "valid_script",
			script:      "return {result = 42}",
			expectError: false,
		},
		{
			name:        "invalid_syntax",
			script:      "return {result =",
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
}

func TestLuaScriptCache(t *testing.T) {
	env := engine.NewLuaEnv()

	step := &api.Step{
		ID:   "test",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script:   "return {result = a + b}",
			Language: api.ScriptLangLua,
		},
		Attributes: api.AttributeSpecs{
			"a":      {Role: api.RoleRequired},
			"b":      {Role: api.RoleRequired},
			"result": {Role: api.RoleRequired},
		},
	}

	proc1, err := env.Compile(step, step.Script)
	assert.NoError(t, err)

	proc2, err := env.Compile(step, step.Script)
	assert.NoError(t, err)

	assert.Equal(t, proc1, proc2)
}

func TestLuaCompileViaRegistry(t *testing.T) {
	registry := engine.NewScriptRegistry()

	script := &api.Step{
		ID:   "test",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return {x = 42}",
		},
		Attributes: api.AttributeSpecs{
			"x": {Role: api.RoleRequired},
		},
	}

	pred := &api.Step{
		ID:   "test",
		Type: api.StepTypeSync,
		Predicate: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   "return x > 10",
		},
		Attributes: api.AttributeSpecs{
			"x": {Role: api.RoleRequired},
		},
	}

	scriptComp, err := registry.Compile(script, script.Script)
	assert.NoError(t, err)
	assert.NotNil(t, scriptComp)

	predComp, err := registry.Compile(pred, pred.Predicate)
	assert.NoError(t, err)
	assert.NotNil(t, predComp)
}

func TestLuaComplexConversion(t *testing.T) {
	env := engine.NewLuaEnv()

	step := &api.Step{
		ID:   "complex-types",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script: `
				return {
					bool_val = is_active,
					string_val = name,
					int_val = count,
					float_val = price
				}
			`,
		},
		Attributes: api.AttributeSpecs{
			"is_active":  {Role: api.RoleRequired},
			"name":       {Role: api.RoleRequired},
			"count":      {Role: api.RoleRequired},
			"price":      {Role: api.RoleRequired},
			"bool_val":   {Role: api.RoleRequired},
			"string_val": {Role: api.RoleRequired},
			"int_val":    {Role: api.RoleRequired},
			"float_val":  {Role: api.RoleRequired},
		},
	}

	comp, err := env.Compile(step, step.Script)
	assert.NoError(t, err)

	args := api.Args{
		"is_active": true,
		"name":      "test-item",
		"count":     42,
		"price":     99.99,
	}

	result, err := env.ExecuteScript(comp, step, args)
	assert.NoError(t, err)

	assert.Equal(t, true, result["bool_val"])
	assert.Equal(t, "test-item", result["string_val"])
	assert.Equal(t, 42, result["int_val"])
	assert.Equal(t, 99.99, result["float_val"])
}

func TestLuaArrayTableConversion(t *testing.T) {
	env := engine.NewLuaEnv()

	step := &api.Step{
		ID:   "array-test",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   `return {numbers = {1, 2, 3, 4, 5}, count = 5}`,
		},
		Attributes: api.AttributeSpecs{
			"numbers": {Role: api.RoleRequired},
			"count":   {Role: api.RoleRequired},
		},
	}

	comp, err := env.Compile(step, step.Script)
	assert.NoError(t, err)

	result, err := env.ExecuteScript(comp, step, api.Args{})
	assert.NoError(t, err)

	numbers, ok := result["numbers"].([]any)
	assert.True(t, ok)
	assert.Equal(t, 5, len(numbers))
	assert.Equal(t, 1, numbers[0])
	assert.Equal(t, 5, numbers[4])

	assert.Equal(t, 5, result["count"])
}

func TestLuaInputTypes(t *testing.T) {
	env := engine.NewLuaEnv()

	tests := []struct {
		name     string
		script   string
		inputs   api.Args
		expected api.Args
	}{
		{
			name:   "int64_input",
			script: "return {result = val}",
			inputs: api.Args{"val": int64(123456789)},
			expected: api.Args{
				"result": 123456789,
			},
		},
		{
			name:   "float64_input",
			script: "return {result = val * 2}",
			inputs: api.Args{"val": 3.14},
			expected: api.Args{
				"result": 6.28,
			},
		},
		{
			name:   "nil_input",
			script: "return {result = val == nil}",
			inputs: api.Args{"val": nil},
			expected: api.Args{
				"result": true,
			},
		},
		{
			name:   "array_input",
			script: "return {result = #items}",
			inputs: api.Args{"items": []any{1, 2, 3}},
			expected: api.Args{
				"result": 3,
			},
		},
		{
			name:   "map_input",
			script: "return {result = data.key}",
			inputs: api.Args{"data": map[string]any{"key": "value"}},
			expected: api.Args{
				"result": "value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := &api.Step{
				ID:   api.StepID(tt.name),
				Type: api.StepTypeScript,
				Script: &api.ScriptConfig{
					Language: api.ScriptLangLua,
					Script:   tt.script,
				},
				Attributes: api.AttributeSpecs{
					"val":    {Role: api.RoleRequired},
					"items":  {Role: api.RoleRequired},
					"data":   {Role: api.RoleRequired},
					"result": {Role: api.RoleRequired},
				},
			}

			comp, err := env.Compile(step, step.Script)
			assert.NoError(t, err)

			result, err := env.ExecuteScript(comp, step, tt.inputs)
			assert.NoError(t, err)

			for key, expected := range tt.expected {
				assert.Equal(t, expected, result[key])
			}
		})
	}
}


func TestLuaEmptyArray(t *testing.T) {
	env := engine.NewLuaEnv()

	step := &api.Step{
		ID:   "empty-array",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script:   `return {items = {}}`,
		},
		Attributes: api.AttributeSpecs{
			"items": {Role: api.RoleRequired},
		},
	}

	comp, err := env.Compile(step, step.Script)
	assert.NoError(t, err)

	result, err := env.ExecuteScript(comp, step, api.Args{})
	assert.NoError(t, err)

	items, ok := result["items"].(map[string]any)
	assert.True(t, ok)
	assert.Empty(t, items)
}

func TestLuaNestedArrays(t *testing.T) {
	env := engine.NewLuaEnv()

	step := &api.Step{
		ID:   "nested-arrays",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script: `
				return {
					matrix = {{1, 2}, {3, 4}, {5, 6}}
				}
			`,
		},
		Attributes: api.AttributeSpecs{
			"matrix": {Role: api.RoleRequired},
		},
	}

	comp, err := env.Compile(step, step.Script)
	assert.NoError(t, err)

	result, err := env.ExecuteScript(comp, step, api.Args{})
	assert.NoError(t, err)

	matrix, ok := result["matrix"].([]any)
	assert.True(t, ok)
	assert.Len(t, matrix, 3)

	row1, ok := matrix[0].([]any)
	assert.True(t, ok)
	assert.Equal(t, 1, row1[0])
	assert.Equal(t, 2, row1[1])
}

func TestLuaLargeArray(t *testing.T) {
	env := engine.NewLuaEnv()

	step := &api.Step{
		ID:   "large-array",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: api.ScriptLangLua,
			Script: `
				local arr = {}
				for i = 1, 10 do
					arr[i] = i * 10
				end
				return {numbers = arr}
			`,
		},
		Attributes: api.AttributeSpecs{
			"numbers": {Role: api.RoleRequired},
		},
	}

	comp, err := env.Compile(step, step.Script)
	assert.NoError(t, err)

	result, err := env.ExecuteScript(comp, step, api.Args{})
	assert.NoError(t, err)

	numbers, ok := result["numbers"].([]any)
	assert.True(t, ok)
	assert.Len(t, numbers, 10)
	assert.Equal(t, 10, numbers[0])
	assert.Equal(t, 100, numbers[9])
}
