package engine_test

import (
	"testing"

	"github.com/kode4food/ale/data"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestCacheForCurrentSteps(t *testing.T) {
	env := engine.NewAleEnv()

	step := &api.Step{
		ID:   "test-step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script: "{:result (+ a b)}",
		},
		Attributes: map[api.Name]*api.AttributeSpec{
			"a":      {Role: api.RoleRequired},
			"b":      {Role: api.RoleRequired},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	names := step.SortedArgNames()

	proc1, err := env.Compile(step, step.Script.Script, names)
	require.NoError(t, err)

	proc2, err := env.Compile(step, step.Script.Script, names)
	require.NoError(t, err)

	// Verify scripts are cached by checking same object returned
	assert.Equal(t, proc1, proc2)
}

func TestCompileForPlan(t *testing.T) {
	registry := engine.NewScriptRegistry()

	script := &api.Step{
		ID:   "script-step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script:   "{:result (* a b)}",
			Language: api.ScriptLangAle,
		},
		Attributes: map[api.Name]*api.AttributeSpec{
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
		Attributes: map[api.Name]*api.AttributeSpec{
			"x":      {Role: api.RoleRequired},
			"output": {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"script-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			script.ID: {Step: script},
			pred.ID:   {Step: pred},
		},
	}

	err := registry.CompilePlan(plan)
	require.NoError(t, err)

	assert.NotNil(t, plan.Steps[script.ID].Script)
	assert.NotNil(t, plan.Steps[pred.ID].Predicate)

	scriptProc, ok := plan.Steps[script.ID].Script.(data.Procedure)
	assert.True(t, ok)
	assert.NotNil(t, scriptProc)

	predProc, ok := plan.Steps[pred.ID].Predicate.(data.Procedure)
	assert.True(t, ok)
	assert.NotNil(t, predProc)
}

func TestCompiledIndependence(t *testing.T) {
	registry := engine.NewScriptRegistry()

	step := &api.Step{
		ID:   "test-step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script:   "{:result (+ a b)}",
			Language: api.ScriptLangAle,
		},
		Attributes: map[api.Name]*api.AttributeSpec{
			"a":      {Role: api.RoleRequired},
			"b":      {Role: api.RoleRequired},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	pl1 := &api.ExecutionPlan{
		Goals: []timebox.ID{"test-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	pl2 := &api.ExecutionPlan{
		Goals: []timebox.ID{"test-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err := registry.CompilePlan(pl1)
	require.NoError(t, err)

	err = registry.CompilePlan(pl2)
	require.NoError(t, err)

	proc1, ok1 := pl1.Steps[step.ID].Script.(data.Procedure)
	proc2, ok2 := pl2.Steps[step.ID].Script.(data.Procedure)

	assert.True(t, ok1)
	assert.True(t, ok2)

	assert.Equal(t, proc1, proc2)

	pl1.Steps[step.ID].Script = nil

	assert.NotNil(t, pl2.Steps[step.ID].Script)
}

func TestIsolatedUpdate(t *testing.T) {
	registry := engine.NewScriptRegistry()
	env, _ := registry.Get(api.ScriptLangAle)

	oldStep := &api.Step{
		ID:   "test-step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script:   "{:result (+ a b)}",
			Language: api.ScriptLangAle,
		},
		Attributes: map[api.Name]*api.AttributeSpec{
			"a":      {Role: api.RoleRequired},
			"b":      {Role: api.RoleRequired},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"test-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			oldStep.ID: {Step: oldStep},
		},
	}

	err := registry.CompilePlan(plan)
	require.NoError(t, err)

	oldProc := plan.Steps[oldStep.ID].Script

	newStep := &api.Step{
		ID:   "test-step",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script: "{:result (* a b)}",
		},
		Attributes: map[api.Name]*api.AttributeSpec{
			"a":      {Role: api.RoleRequired},
			"b":      {Role: api.RoleRequired},
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	names := newStep.SortedArgNames()
	_, err = env.Compile(newStep, newStep.Script.Script, names)
	require.NoError(t, err)

	assert.Equal(t, oldProc, plan.Steps[oldStep.ID].Script)
}

func TestExecuteScript(t *testing.T) {
	env := engine.NewAleEnv()

	step := &api.Step{
		ID:   "test",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script: "{:result (+ a b)}",
		},
		Attributes: map[api.Name]*api.AttributeSpec{
			"a":      {Role: api.RoleRequired},
			"b":      {Role: api.RoleRequired},
			"result": {Role: api.RoleRequired},
		},
	}

	names := step.SortedArgNames()
	proc, err := env.Compile(step, step.Script.Script, names)
	require.NoError(t, err)

	args := api.Args{
		"a": 5,
		"b": 10,
	}

	result, err := env.ExecuteScript(proc, step, args)
	require.NoError(t, err)

	assert.Contains(t, result, api.Name("result"))
	assert.Equal(t, 15, result["result"])
}

func TestEvaluatePredicate(t *testing.T) {
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
					Script: tt.predicate,
				},
				Attributes: map[api.Name]*api.AttributeSpec{
					"x": {Role: api.RoleRequired},
					"y": {Role: api.RoleRequired},
				},
			}

			names := step.SortedArgNames()
			comp, err := env.Compile(step, tt.predicate, names)
			require.NoError(t, err)

			result, err := env.EvaluatePredicate(comp, step, tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidate(t *testing.T) {
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
}

func TestAleComplexConversion(t *testing.T) {
	env := engine.NewAleEnv()

	step := &api.Step{
		ID:   "complex-types",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
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
		Attributes: map[api.Name]*api.AttributeSpec{
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

	names := step.SortedArgNames()
	comp, err := env.Compile(step, step.Script.Script, names)
	require.NoError(t, err)

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
	require.NoError(t, err)

	assert.Equal(t, true, result["bool_val"])
	assert.Equal(t, "test-item", result["string_val"])
	assert.Equal(t, 42, result["int_val"])
	assert.Equal(t, 99.99, result["float_val"])

	arrVal, ok := result["array_val"].([]any)
	require.True(t, ok)
	assert.Len(t, arrVal, 3)
	assert.Equal(t, "item1", arrVal[0])

	nested, ok := result["nested"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value1", nested["key1"])
	assert.Equal(t, 123, nested["key2"])

	assert.Nil(t, result["null_val"])
}

func TestAleListConversion(t *testing.T) {
	env := engine.NewAleEnv()

	step := &api.Step{
		ID:   "list-test",
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Script: `{:list_result (list 1 2 3 4 5)}`,
		},
		Attributes: map[api.Name]*api.AttributeSpec{
			"list_result": {Role: api.RoleRequired},
		},
	}

	names := step.SortedArgNames()
	comp, err := env.Compile(step, step.Script.Script, names)
	require.NoError(t, err)

	result, err := env.ExecuteScript(comp, step, api.Args{})
	require.NoError(t, err)

	listVal, ok := result["list_result"].([]any)
	require.True(t, ok)
	assert.Len(t, listVal, 5)
	assert.Equal(t, 1, listVal[0])
	assert.Equal(t, 5, listVal[4])
}
