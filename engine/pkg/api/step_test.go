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
			name: "http_with_flow_config",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost:8080",
				},
				Flow: &api.FlowConfig{
					Goals: []api.StepID{"goal"},
				},
			},
			expectError:   true,
			errorContains: "flow not allowed",
		},
		{
			name: "http_with_script_config",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost:8080",
				},
				Script: &api.ScriptConfig{
					Language: api.ScriptLangLua,
					Script:   "return {}",
				},
			},
			expectError:   true,
			errorContains: "script not allowed",
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
			name: "script_with_http_config",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Script",
				Type: api.StepTypeScript,
				Script: &api.ScriptConfig{
					Language: api.ScriptLangLua,
					Script:   "return {}",
				},
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost:8080",
				},
			},
			expectError:   true,
			errorContains: "http not allowed",
		},
		{
			name: "script_with_flow_config",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Script",
				Type: api.StepTypeScript,
				Script: &api.ScriptConfig{
					Language: api.ScriptLangLua,
					Script:   "return {}",
				},
				Flow: &api.FlowConfig{
					Goals: []api.StepID{"goal"},
				},
			},
			expectError:   true,
			errorContains: "flow not allowed",
		},
		{
			name: "missing_flow_config",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Flow",
				Type: api.StepTypeFlow,
			},
			expectError:   true,
			errorContains: "flow required",
		},
		{
			name: "missing_flow_goals",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Flow",
				Type: api.StepTypeFlow,
				Flow: &api.FlowConfig{},
			},
			expectError:   true,
			errorContains: "flow goals required",
		},
		{
			name: "flow_with_http_config",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Flow",
				Type: api.StepTypeFlow,
				Flow: &api.FlowConfig{
					Goals: []api.StepID{"goal"},
				},
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost:8080",
				},
			},
			expectError:   true,
			errorContains: "http not allowed",
		},
		{
			name: "flow_with_script_config",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Flow",
				Type: api.StepTypeFlow,
				Flow: &api.FlowConfig{
					Goals: []api.StepID{"goal"},
				},
				Script: &api.ScriptConfig{
					Language: api.ScriptLangLua,
					Script:   "return {}",
				},
			},
			expectError:   true,
			errorContains: "script not allowed",
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
			"carrot": {
				Role:    api.RoleConst,
				Type:    api.TypeString,
				Default: `"fixed"`,
			},
		},
	}

	sorted := step.SortedArgNames()
	as.Len(sorted, 5)
	as.Equal("apple", sorted[0])
	as.Equal("banana", sorted[1])
	as.Equal("carrot", sorted[2])
	as.Equal("mango", sorted[3])
	as.Equal("zebra", sorted[4])
}

func TestSortedArgNamesUsesMappingNames(t *testing.T) {
	as := assert.New(t)

	step := &api.Step{
		ID:   "sorted-mapped-args",
		Name: "Sorted Mapped Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
		Attributes: api.AttributeSpecs{
			"outer_b": {
				Role: api.RoleRequired,
				Type: api.TypeString,
				Mapping: &api.AttributeMapping{
					Name: "inner_b",
				},
			},
			"outer_a": {
				Role: api.RoleOptional,
				Type: api.TypeString,
				Mapping: &api.AttributeMapping{
					Name: "inner_a",
				},
			},
			"plain": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
		},
	}

	sorted := step.SortedArgNames()
	as.Equal([]string{"inner_a", "inner_b", "plain"}, sorted)
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

func TestEqualFlowConfig(t *testing.T) {
	as := assert.New(t)

	config1 := &api.FlowConfig{
		Goals: []api.StepID{"goal-1"},
	}

	config2 := &api.FlowConfig{
		Goals: []api.StepID{"goal-1"},
	}

	config3 := &api.FlowConfig{
		Goals: []api.StepID{"goal-2"},
	}

	as.True(config1.Equal(config2))
	as.False(config1.Equal(config3))
	as.True((*api.FlowConfig)(nil).Equal(nil))
	as.False(config1.Equal(nil))
}

func TestEqualWorkConfig(t *testing.T) {
	as := assert.New(t)

	config1 := &api.WorkConfig{
		Parallelism: 5,
		MaxRetries:  3,
		Backoff:     1000,
		MaxBackoff:  60000,
		BackoffType: api.BackoffTypeExponential,
	}

	config2 := &api.WorkConfig{
		Parallelism: 5,
		MaxRetries:  3,
		Backoff:     1000,
		MaxBackoff:  60000,
		BackoffType: api.BackoffTypeExponential,
	}

	config3 := &api.WorkConfig{
		Parallelism: 10,
		MaxRetries:  3,
		Backoff:     1000,
		MaxBackoff:  60000,
		BackoffType: api.BackoffTypeExponential,
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

func TestLabelsApply(t *testing.T) {
	as := assert.New(t)

	t.Run("empty_other", func(t *testing.T) {
		base := api.Labels{"team": "core"}
		applied := base.Apply(nil)

		as.Equal(api.Labels{"team": "core"}, applied)
	})

	t.Run("nil_base", func(t *testing.T) {
		applied := api.Labels(nil).Apply(api.Labels{"team": "core"})

		as.Equal(api.Labels{"team": "core"}, applied)
	})

	t.Run("merge_override", func(t *testing.T) {
		base := api.Labels{"team": "core", "env": "dev"}
		applied := base.Apply(api.Labels{"team": "other"})

		as.Equal(api.Labels{"team": "other", "env": "dev"}, applied)
	})

	t.Run("base_unchanged", func(t *testing.T) {
		base := api.Labels{"team": "core"}
		_ = base.Apply(api.Labels{"env": "dev"})

		as.Equal(api.Labels{"team": "core"}, base)
	})
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
				Backoff: -1,
			},
		}
		as.StepInvalid(step, "backoff cannot be negative")
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
				Backoff:    1000,
				MaxBackoff: 500,
			},
		}
		as.StepInvalid(step, "max_backoff")
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
				MaxRetries:  3,
				Backoff:     1000,
				MaxBackoff:  60000,
				BackoffType: api.BackoffTypeExponential,
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
				Script:   "(eq status \"ready\")",
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
				Script:   "(eq status \"pending\")",
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
		complexData := map[string]any{
			"nested": map[string]any{
				"value": 123,
			},
		}
		result = result.WithOutput("complex", complexData)

		as.Equal(complexData, result.Outputs["complex"])
	})

	t.Run("with_output_array", func(t *testing.T) {
		result := &api.StepResult{Success: true}
		arrayData := []any{"a", "b", "c"}
		result = result.WithOutput("array", arrayData)

		as.Equal(arrayData, result.Outputs["array"])
	})

	t.Run("with_output_preserves_success_state", func(t *testing.T) {
		result := &api.StepResult{Success: true}
		result = result.WithOutput("key", "value")

		as.True(result.Success)
	})
}

func TestStepHashKey(t *testing.T) {
	as := assert.New(t)

	t.Run("deterministic", func(t *testing.T) {
		s := &api.Step{
			ID:   "test-step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://test",
			},
		}
		h1, err := s.HashKey()
		as.NoError(err)
		h2, err := s.HashKey()
		as.NoError(err)
		as.Equal(h1, h2)
	})

	t.Run("cached", func(t *testing.T) {
		s := &api.Step{
			ID:   "test-step",
			Type: api.StepTypeSync,
		}
		h1, err := s.HashKey()
		as.NoError(err)
		h2, err := s.HashKey()
		as.NoError(err)
		as.Equal(h1, h2)
	})

	t.Run("different_types", func(t *testing.T) {
		s1 := &api.Step{Type: api.StepTypeSync}
		s2 := &api.Step{Type: api.StepTypeAsync}
		h1, err := s1.HashKey()
		as.NoError(err)
		h2, err := s2.HashKey()
		as.NoError(err)
		as.NotEqual(h1, h2)
	})

	t.Run("different_http_configs", func(t *testing.T) {
		s1 := &api.Step{
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{Endpoint: "http://a"},
		}
		s2 := &api.Step{
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{Endpoint: "http://b"},
		}
		h1, err := s1.HashKey()
		as.NoError(err)
		h2, err := s2.HashKey()
		as.NoError(err)
		as.NotEqual(h1, h2)
	})

	t.Run("ignores_id", func(t *testing.T) {
		s1 := &api.Step{
			ID:   "id1",
			Type: api.StepTypeSync,
		}
		s2 := &api.Step{
			ID:   "id2",
			Type: api.StepTypeSync,
		}
		h1, err := s1.HashKey()
		as.NoError(err)
		h2, err := s2.HashKey()
		as.NoError(err)
		as.Equal(h1, h2)
	})

	t.Run("ignores_name", func(t *testing.T) {
		s1 := &api.Step{
			Type: api.StepTypeSync,
			Name: "name1",
		}
		s2 := &api.Step{
			Type: api.StepTypeSync,
			Name: "name2",
		}
		h1, err := s1.HashKey()
		as.NoError(err)
		h2, err := s2.HashKey()
		as.NoError(err)
		as.Equal(h1, h2)
	})

	t.Run("ignores_labels", func(t *testing.T) {
		s1 := &api.Step{
			Type:   api.StepTypeSync,
			Labels: map[string]string{"env": "dev"},
		}
		s2 := &api.Step{
			Type:   api.StepTypeSync,
			Labels: map[string]string{"env": "prod"},
		}
		h1, err := s1.HashKey()
		as.NoError(err)
		h2, err := s2.HashKey()
		as.NoError(err)
		as.Equal(h1, h2)
	})

	t.Run("includes_memoizable", func(t *testing.T) {
		s1 := &api.Step{
			Type:       api.StepTypeSync,
			Memoizable: false,
		}
		s2 := &api.Step{
			Type:       api.StepTypeSync,
			Memoizable: true,
		}
		h1, err := s1.HashKey()
		as.NoError(err)
		h2, err := s2.HashKey()
		as.NoError(err)
		as.NotEqual(h1, h2)
	})

	t.Run("includes_attributes", func(t *testing.T) {
		s1 := &api.Step{
			Type: api.StepTypeSync,
			Attributes: map[api.Name]*api.AttributeSpec{
				"in": {Role: api.RoleRequired},
			},
		}
		s2 := &api.Step{
			Type: api.StepTypeSync,
			Attributes: map[api.Name]*api.AttributeSpec{
				"out": {Role: api.RoleOutput},
			},
		}
		h1, err := s1.HashKey()
		as.NoError(err)
		h2, err := s2.HashKey()
		as.NoError(err)
		as.NotEqual(h1, h2)
	})

	t.Run("with_script", func(t *testing.T) {
		s := &api.Step{
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: "ale",
				Script:   "return 42",
			},
		}
		h, err := s.HashKey()
		as.NoError(err)
		as.NotEmpty(h)
	})

	t.Run("with_flow", func(t *testing.T) {
		s := &api.Step{
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"goal1", "goal2"},
			},
		}
		h, err := s.HashKey()
		as.NoError(err)
		as.NotEmpty(h)
	})
}

func TestStepValidateMappingNames(t *testing.T) {
	as := assert.New(t)

	t.Run("duplicate_input_inner_names", func(t *testing.T) {
		step := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
				Timeout:  30 * api.Second,
			},
			Attributes: api.AttributeSpecs{
				"email": {
					Role:    api.RoleRequired,
					Mapping: &api.AttributeMapping{Name: "user_email"},
				},
				"contact": {
					Role:    api.RoleRequired,
					Mapping: &api.AttributeMapping{Name: "user_email"},
				},
			},
		}
		err := step.Validate()
		as.ErrorIs(err, api.ErrDuplicateInnerName)
		as.ErrorContains(err, "user_email")
	})

	t.Run("duplicate_output_inner_names", func(t *testing.T) {
		step := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
				Timeout:  30 * api.Second,
			},
			Attributes: api.AttributeSpecs{
				"sent": {
					Role:    api.RoleOutput,
					Mapping: &api.AttributeMapping{Name: "status"},
				},
				"delivered": {
					Role:    api.RoleOutput,
					Mapping: &api.AttributeMapping{Name: "status"},
				},
			},
		}
		err := step.Validate()
		as.ErrorIs(err, api.ErrDuplicateInnerName)
		as.ErrorContains(err, "status")
	})

	t.Run("same_inner_name_input_and_output_allowed", func(t *testing.T) {
		step := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
				Timeout:  30 * api.Second,
			},
			Attributes: api.AttributeSpecs{
				"user_data": {
					Role:    api.RoleRequired,
					Mapping: &api.AttributeMapping{Name: "data"},
				},
				"result_data": {
					Role:    api.RoleOutput,
					Mapping: &api.AttributeMapping{Name: "data"},
				},
			},
		}
		err := step.Validate()
		as.NoError(err)
	})
}
