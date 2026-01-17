package api_test

import (
	"testing"

	"github.com/kode4food/argyll/engine/internal/assert"
	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestStepValidation(t *testing.T) {
	as := assert.New(t)

	tests := []struct {
		step          *api.Step
		name          string
		errorContains string
		expectError   bool
	}{
		{
			name:        "valid_step",
			step:        helpers.NewTestStep(),
			expectError: false,
		},
		{
			name: "missing_id",
			step: &api.Step{
				Name: "Test",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost:8080",
				},
			},
			expectError:   true,
			errorContains: "ID empty",
		},
		{
			name: "missing_name",
			step: &api.Step{
				ID:   "test-id",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost:8080",
				},
			},
			expectError:   true,
			errorContains: "name empty",
		},
		{
			name: "missing_http_config",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test",
				Type: api.StepTypeSync,
			},
			expectError:   true,
			errorContains: "http required",
		},
		{
			name: "missing_endpoint",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "",
				},
			},
			expectError:   true,
			errorContains: "endpoint empty",
		},
		{
			name: "empty_required_arg_name",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost:8080",
				},
				Attributes: api.AttributeSpecs{
					"": {Role: api.RoleRequired, Type: api.TypeString},
				},
			},
			expectError:   true,
			errorContains: "argument name empty",
		},
		{
			name: "empty_optional_arg_name",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost:8080",
				},
				Attributes: api.AttributeSpecs{
					"": {Role: api.RoleOptional, Type: api.TypeString},
				},
			},
			expectError:   true,
			errorContains: "argument name empty",
		},
		{
			name: "missing_script_config",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Script",
				Type: api.StepTypeScript,
			},
			expectError:   true,
			errorContains: "script required",
		},
		{
			name: "empty_script",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Script",
				Type: api.StepTypeScript,
				Script: &api.ScriptConfig{
					Language: api.ScriptLangAle,
					Script:   "",
				},
			},
			expectError:   true,
			errorContains: "script empty",
		},
		{
			name: "missing_script_language",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Script",
				Type: api.StepTypeScript,
				Script: &api.ScriptConfig{
					Language: "",
					Script:   "(+ 1 2)",
				},
			},
			expectError:   true,
			errorContains: "script language empty",
		},
		{
			name: "invalid_script_language",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Script",
				Type: api.StepTypeScript,
				Script: &api.ScriptConfig{
					Language: "internal",
					Script:   "(+ 1 2)",
				},
			},
			expectError:   true,
			errorContains: "invalid script language",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectError {
				as.StepInvalid(tt.step, tt.errorContains)
				return
			}
			as.StepValid(tt.step)
		})
	}
}

func TestStepHelperMethods(t *testing.T) {
	as := assert.New(t)

	step := helpers.NewTestStepWithArgs(
		[]api.Name{"req1", "req2"},
		[]api.Name{"opt1", "opt2"},
	)

	t.Run("get_all_input_args", func(t *testing.T) {
		args := step.GetAllInputArgs()
		as.Len(args, 4)
		as.Contains(args, api.Name("req1"))
		as.Contains(args, api.Name("req2"))
		as.Contains(args, api.Name("opt1"))
		as.Contains(args, api.Name("opt2"))
	})

	t.Run("is_optional_arg", func(t *testing.T) {
		as.True(step.IsOptionalArg("opt1"))
		as.True(step.IsOptionalArg("opt2"))
		as.False(step.IsOptionalArg("req1"))
		as.False(step.IsOptionalArg("req2"))
		as.False(step.IsOptionalArg("nonexistent"))
	})
}

func TestStepOutputArgs(t *testing.T) {
	as := assert.New(t)

	t.Run("multiple_outputs", func(t *testing.T) {
		step := &api.Step{
			ID:   "multi-output",
			Name: "Multi-Output Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			Attributes: api.AttributeSpecs{
				"result1":  {Role: api.RoleOutput, Type: api.TypeString},
				"result2":  {Role: api.RoleOutput, Type: api.TypeNumber},
				"metadata": {Role: api.RoleOutput, Type: api.TypeObject},
			},
		}
		outputs := step.GetOutputArgs()
		as.Len(outputs, 3)
		as.Contains(outputs, api.Name("result1"))
		as.Contains(outputs, api.Name("result2"))
		as.Contains(outputs, api.Name("metadata"))
	})

	t.Run("no_outputs", func(t *testing.T) {
		step := &api.Step{
			ID:   "no-output",
			Name: "Terminal Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			Attributes: api.AttributeSpecs{
				"data": {Role: api.RoleRequired, Type: api.TypeString},
			},
		}
		as.Empty(step.GetOutputArgs())
	})
}

func TestSortedArgNames(t *testing.T) {
	as := assert.New(t)

	step := &api.Step{
		ID:   "sorted-args",
		Name: "Sorted Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
		Attributes: api.AttributeSpecs{
			"zebra":  {Role: api.RoleRequired, Type: api.TypeString},
			"apple":  {Role: api.RoleRequired, Type: api.TypeString},
			"mango":  {Role: api.RoleOptional, Type: api.TypeString},
			"banana": {Role: api.RoleOptional, Type: api.TypeString},
		},
	}

	sorted := step.SortedArgNames()
	as.Len(sorted, 4)
	as.Equal("apple", sorted[0])
	as.Equal("banana", sorted[1])
	as.Equal("mango", sorted[2])
	as.Equal("zebra", sorted[3])
}
func TestMultiArgNames(t *testing.T) {
	as := assert.New(t)

	step := &api.Step{
		ID:   "multi-args",
		Name: "Multi Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
		Attributes: api.AttributeSpecs{
			"users": {
				Role:    api.RoleRequired,
				Type:    api.TypeArray,
				ForEach: true,
			},
			"items": {
				Role:    api.RoleOptional,
				Type:    api.TypeArray,
				ForEach: true,
			},
			"config": {
				Role: api.RoleRequired,
				Type: api.TypeObject,
			},
			"messages": {
				Role:    api.RoleOptional,
				Type:    api.TypeArray,
				ForEach: true,
			},
		},
	}

	multiArgs := step.MultiArgNames()
	as.Len(multiArgs, 3)
	as.Contains(multiArgs, api.Name("users"))
	as.Contains(multiArgs, api.Name("items"))
	as.Contains(multiArgs, api.Name("messages"))
	as.NotContains(multiArgs, api.Name("config"))
}

func TestGetRequiredArgs(t *testing.T) {
	as := assert.New(t)

	step := &api.Step{
		ID:   "required-args",
		Name: "Required Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
		Attributes: api.AttributeSpecs{
			"user_id": {Role: api.RoleRequired, Type: api.TypeString},
			"email":   {Role: api.RoleRequired, Type: api.TypeString},
			"name":    {Role: api.RoleOptional, Type: api.TypeString},
			"result":  {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	requiredArgs := step.GetRequiredArgs()
	as.Len(requiredArgs, 2)
	as.Contains(requiredArgs, api.Name("user_id"))
	as.Contains(requiredArgs, api.Name("email"))
	as.NotContains(requiredArgs, api.Name("name"))
	as.NotContains(requiredArgs, api.Name("result"))
}

func TestGetOptionalArgs(t *testing.T) {
	as := assert.New(t)

	step := &api.Step{
		ID:   "optional-args",
		Name: "Optional Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
		Attributes: api.AttributeSpecs{
			"user_id": {Role: api.RoleRequired, Type: api.TypeString},
			"email":   {Role: api.RoleOptional, Type: api.TypeString},
			"name":    {Role: api.RoleOptional, Type: api.TypeString},
			"result":  {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	optionalArgs := step.GetOptionalArgs()
	as.Len(optionalArgs, 2)
	as.Contains(optionalArgs, api.Name("email"))
	as.Contains(optionalArgs, api.Name("name"))
	as.NotContains(optionalArgs, api.Name("user_id"))
	as.NotContains(optionalArgs, api.Name("result"))
}

func TestNewResult(t *testing.T) {
	as := assert.New(t)

	result := api.NewResult()
	as.True(result.Success)
	as.Nil(result.Outputs)
	as.Empty(result.Error)
}

func TestWithOutput(t *testing.T) {
	as := assert.New(t)

	result := &api.StepResult{Success: true}
	result = result.WithOutput("key", "value")

	as.NotNil(result.Outputs)
	as.Equal("value", result.Outputs["key"])
}

func TestWithError(t *testing.T) {
	as := assert.New(t)

	result := &api.StepResult{Success: true}
	result = result.WithError(api.ErrStepIDEmpty)

	as.False(result.Success)
	as.Contains(result.Error, "ID")
}

func TestEqualHTTP(t *testing.T) {
	as := assert.New(t)

	config1 := &api.HTTPConfig{
		Endpoint:    "http://localhost:8080",
		HealthCheck: "http://localhost:8080/health",
		Timeout:     30,
	}

	config2 := &api.HTTPConfig{
		Endpoint:    "http://localhost:8080",
		HealthCheck: "http://localhost:8080/health",
		Timeout:     30,
	}

	config3 := &api.HTTPConfig{
		Endpoint:    "http://localhost:9090",
		HealthCheck: "http://localhost:8080/health",
		Timeout:     30,
	}

	as.True(config1.Equal(config2))
	as.False(config1.Equal(config3))
	as.True((*api.HTTPConfig)(nil).Equal(nil))
	as.False(config1.Equal(nil))
	as.False((*api.HTTPConfig)(nil).Equal(config1))
}

func TestEqualScript(t *testing.T) {
	as := assert.New(t)

	config1 := &api.ScriptConfig{
		Language: api.ScriptLangAle,
		Script:   "(+ 1 2)",
	}

	config2 := &api.ScriptConfig{
		Language: api.ScriptLangAle,
		Script:   "(+ 1 2)",
	}

	config3 := &api.ScriptConfig{
		Language: api.ScriptLangLua,
		Script:   "return 1 + 2",
	}

	as.True(config1.Equal(config2))
	as.False(config1.Equal(config3))
	as.True((*api.ScriptConfig)(nil).Equal(nil))
	as.False(config1.Equal(nil))
}

func TestEqualWorkConfig(t *testing.T) {
	as := assert.New(t)

	config1 := &api.WorkConfig{
		Parallelism:  5,
		MaxRetries:   3,
		BackoffMs:    1000,
		MaxBackoffMs: 60000,
		BackoffType:  api.BackoffTypeExponential,
	}

	config2 := &api.WorkConfig{
		Parallelism:  5,
		MaxRetries:   3,
		BackoffMs:    1000,
		MaxBackoffMs: 60000,
		BackoffType:  api.BackoffTypeExponential,
	}

	config3 := &api.WorkConfig{
		Parallelism:  10,
		MaxRetries:   3,
		BackoffMs:    1000,
		MaxBackoffMs: 60000,
		BackoffType:  api.BackoffTypeExponential,
	}

	as.True(config1.Equal(config2))
	as.False(config1.Equal(config3))
	as.True((*api.WorkConfig)(nil).Equal(nil))
	as.False(config1.Equal(nil))
}

func TestEqualStep(t *testing.T) {
	as := assert.New(t)

	step1 := &api.Step{
		ID:   "test-step",
		Name: "Test Step",
		Type: api.StepTypeSync,
		Labels: api.Labels{
			"description": "test step",
			"team":        "core",
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
		Attributes: api.AttributeSpecs{
			"arg1": {Role: api.RoleRequired, Type: api.TypeString},
		},
	}

	step2 := &api.Step{
		ID:   "test-step",
		Name: "Test Step",
		Type: api.StepTypeSync,
		Labels: api.Labels{
			"description": "test step",
			"team":        "core",
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
		Attributes: api.AttributeSpecs{
			"arg1": {Role: api.RoleRequired, Type: api.TypeString},
		},
	}

	step3 := &api.Step{
		ID:   "different-step",
		Name: "Test Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
		Attributes: api.AttributeSpecs{
			"arg1": {Role: api.RoleRequired, Type: api.TypeString},
		},
	}

	as.True(step1.Equal(step2))
	as.False(step1.Equal(step3))
}

func TestValidateWorkConfig(t *testing.T) {
	as := assert.New(t)

	t.Run("negative_backoff", func(t *testing.T) {
		step := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			WorkConfig: &api.WorkConfig{
				BackoffMs: -1,
			},
		}
		as.StepInvalid(step, "backoff_ms cannot be negative")
	})

	t.Run("max_backoff_too_small", func(t *testing.T) {
		step := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			WorkConfig: &api.WorkConfig{
				BackoffMs:    1000,
				MaxBackoffMs: 500,
			},
		}
		as.StepInvalid(step, "max_backoff_ms")
	})

	t.Run("missing_backoff_type", func(t *testing.T) {
		step := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries: 3,
			},
		}
		as.StepInvalid(step, "invalid retry config")
	})

	t.Run("invalid_backoff_type", func(t *testing.T) {
		step := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries:  3,
				BackoffType: "invalid",
			},
		}
		as.StepInvalid(step, "invalid backoff type")
	})

	t.Run("valid_work_config", func(t *testing.T) {
		step := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries:   3,
				BackoffMs:    1000,
				MaxBackoffMs: 60000,
				BackoffType:  api.BackoffTypeExponential,
			},
		}
		as.StepValid(step)
	})
}

func TestStepEqualEdgeCases(t *testing.T) {
	as := assert.New(t)

	baseStep := &api.Step{
		ID:   "test",
		Name: "Test",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
		Attributes: api.AttributeSpecs{
			"arg1": {Role: api.RoleRequired, Type: api.TypeString},
		},
	}

	t.Run("nil_http_configs", func(t *testing.T) {
		step1 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "(+ 1 2)",
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "(+ 1 2)",
			},
		}
		as.True(step1.Equal(step2))
	})

	t.Run("one_nil_http_one_not", func(t *testing.T) {
		step1 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
		}
		as.False(step1.Equal(step2))
	})

	t.Run("nil_script_configs", func(t *testing.T) {
		step1 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
		}
		as.True(step1.Equal(step2))
	})

	t.Run("different_versions", func(t *testing.T) {
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
		}
		as.False(baseStep.Equal(step2))
	})

	t.Run("different_types", func(t *testing.T) {
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeAsync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
		}
		as.False(baseStep.Equal(step2))
	})

	t.Run("different_attribute_maps", func(t *testing.T) {
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			Attributes: api.AttributeSpecs{
				"arg1": {Role: api.RoleOptional, Type: api.TypeString},
			},
		}
		as.False(baseStep.Equal(step2))
	})

	t.Run("nil_predicate_configs", func(t *testing.T) {
		step1 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
		}
		as.True(step1.Equal(step2))
	})

	t.Run("different_predicates", func(t *testing.T) {
		step1 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "(= status \"ready\")",
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "(= status \"pending\")",
			},
		}
		as.False(step1.Equal(step2))
	})

	t.Run("nil_work_configs", func(t *testing.T) {
		step1 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
		}
		as.True(step1.Equal(step2))
	})

	t.Run("different_work_configs", func(t *testing.T) {
		step1 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries:  3,
				BackoffType: api.BackoffTypeFixed,
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://localhost:8080",
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries:  5,
				BackoffType: api.BackoffTypeExponential,
			},
		}
		as.False(step1.Equal(step2))
	})
}

func TestResultEdgeCases(t *testing.T) {
	as := assert.New(t)

	t.Run("multiple_sequential_outputs", func(t *testing.T) {
		result := &api.StepResult{Success: true}
		result = result.
			WithOutput("key1", "value1").
			WithOutput("key2", 42).
			WithOutput("key3", true)

		as.NotNil(result.Outputs)
		as.Len(result.Outputs, 3)
		as.Equal("value1", result.Outputs["key1"])
		as.Equal(42, result.Outputs["key2"])
		as.Equal(true, result.Outputs["key3"])
	})

	t.Run("overwrite_existing_output", func(t *testing.T) {
		result := &api.StepResult{Success: true}
		result = result.
			WithOutput("key", "original").
			WithOutput("key", "updated")

		as.Equal("updated", result.Outputs["key"])
		as.Len(result.Outputs, 1)
	})

	t.Run("with_output_on_nil_outputs", func(t *testing.T) {
		result := &api.StepResult{
			Success: true,
			Outputs: nil,
		}
		result = result.WithOutput("key", "value")

		as.NotNil(result.Outputs)
		as.Equal("value", result.Outputs["key"])
	})

	t.Run("with_output_complex_types", func(t *testing.T) {
		result := &api.StepResult{Success: true}
		complexData := map[string]interface{}{
			"nested": map[string]interface{}{
				"value": 123,
			},
		}
		result = result.WithOutput("complex", complexData)

		as.Equal(complexData, result.Outputs["complex"])
	})

	t.Run("with_output_array", func(t *testing.T) {
		result := &api.StepResult{Success: true}
		arrayData := []interface{}{"a", "b", "c"}
		result = result.WithOutput("array", arrayData)

		as.Equal(arrayData, result.Outputs["array"])
	})

	t.Run("with_output_preserves_success_state", func(t *testing.T) {
		result := &api.StepResult{Success: true}
		result = result.WithOutput("key", "value")

		as.True(result.Success)
	})
}
