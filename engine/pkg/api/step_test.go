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
					Invoke: api.HTTPAction{
						Endpoint: "http://localhost:8080",
					},
				},
			},
			expectError:   true,
			errorContains: "ID empty",
		},
		{
			name: "invalid_id",
			step: &api.Step{
				ID:   "my:step",
				Name: "Test",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{
						Endpoint: "http://localhost:8080",
					},
				},
			},
			expectError:   true,
			errorContains: "invalid characters",
		},
		{
			name: "missing_name",
			step: &api.Step{
				ID:   "test-id",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{
						Endpoint: "http://localhost:8080",
					},
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
					Invoke: api.HTTPAction{Endpoint: ""},
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
					Invoke: api.HTTPAction{
						Endpoint: "http://localhost:8080",
					},
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
					Invoke: api.HTTPAction{
						Endpoint: "http://localhost:8080",
					},
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
					Invoke: api.HTTPAction{
						Endpoint: "http://localhost:8080",
					},
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
					Invoke: api.HTTPAction{
						Endpoint: "http://localhost:8080",
					},
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
					Invoke: api.HTTPAction{
						Endpoint: "http://localhost:8080",
					},
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
			name: "invalid_flow_space_id",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test Flow",
				Type: api.StepTypeFlow,
				Flow: &api.FlowConfig{
					Goals:   []api.StepID{"goal"},
					SpaceID: "invalid:space",
				},
			},
			expectError:   true,
			errorContains: "space ID contains invalid characters",
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
					Invoke: api.HTTPAction{
						Endpoint: "http://localhost:8080",
					},
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
					Language: api.ScriptLangLua,
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
					Script:   "return 1 + 2",
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
					Script:   "return 1 + 2",
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

	st := helpers.NewTestStepWithArgs(
		[]api.Name{"req1", "req2"},
		[]api.Name{"opt1", "opt2"},
	)

	t.Run("get_all_input_args", func(t *testing.T) {
		args := st.GetAllInputArgs()
		as.Len(args, 4)
		as.Contains(args, api.Name("req1"))
		as.Contains(args, api.Name("req2"))
		as.Contains(args, api.Name("opt1"))
		as.Contains(args, api.Name("opt2"))
	})

	t.Run("is_optional_arg", func(t *testing.T) {
		as.True(st.IsOptionalArg("opt1"))
		as.True(st.IsOptionalArg("opt2"))
		as.False(st.IsOptionalArg("req1"))
		as.False(st.IsOptionalArg("req2"))
		as.False(st.IsOptionalArg("nonexistent"))
	})
}

func TestHTTPMethodValidation(t *testing.T) {
	as := assert.New(t)

	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			as.StepValid(&api.Step{
				ID:   "valid-step",
				Name: api.Name("Valid " + method + " Step"),
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{
						Endpoint: "https://example.com/items",
						Method:   method,
					},
				},
				Attributes: api.AttributeSpecs{
					"item_id": {Role: api.RoleRequired, Type: api.TypeString},
				},
			})

			as.StepValid(&api.Step{
				ID:   "placeholder-step",
				Name: api.Name("Placeholder " + method + " Step"),
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{
						Endpoint: "https://example.com/items/{item_id}",
						Method:   method,
					},
				},
				Attributes: api.AttributeSpecs{
					"item_id": {Role: api.RoleRequired, Type: api.TypeString},
				},
			})

			as.StepValid(&api.Step{
				ID:   "mapped-step",
				Name: api.Name("Mapped " + method + " Step"),
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{
						Endpoint: "https://example.com/items/{external_id}",
						Method:   method,
					},
				},
				Attributes: api.AttributeSpecs{
					"item_id": {
						Role: api.RoleRequired,
						Type: api.TypeString,
						Required: &api.RequiredConfig{
							Mapping: &api.MappingConfig{Name: "external_id"},
						},
					},
				},
			})

			as.StepInvalid(&api.Step{
				ID:   "unknown-param-step",
				Name: api.Name("Unknown Param " + method + " Step"),
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{
						Endpoint: "https://example.com/items/{extra}",
						Method:   method,
					},
				},
				Attributes: api.AttributeSpecs{
					"item_id": {Role: api.RoleRequired, Type: api.TypeString},
				},
			}, "unknown parameter")

			as.StepInvalid(&api.Step{
				ID:   "optional-param-step",
				Name: api.Name("Optional Param " + method + " Step"),
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Invoke: api.HTTPAction{
						Endpoint: "https://example.com/items/{item_id}",
						Method:   method,
					},
				},
				Attributes: api.AttributeSpecs{
					"item_id": {Role: api.RoleOptional, Type: api.TypeString},
				},
			}, "unknown parameter")
		})
	}

	as.StepInvalid(&api.Step{
		ID:   "bad-method-step",
		Name: "Bad Method Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "https://example.com/items/{item_id}",
				Method:   "PATCH",
			},
		},
		Attributes: api.AttributeSpecs{
			"item_id": {Role: api.RoleRequired, Type: api.TypeString},
		},
	}, "invalid HTTP method")
}

func TestStepOutputArgs(t *testing.T) {
	as := assert.New(t)

	t.Run("multiple_outputs", func(t *testing.T) {
		st := &api.Step{
			ID:   "multi-output",
			Name: "Multi-Output Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			Attributes: api.AttributeSpecs{
				"result1":  {Role: api.RoleOutput, Type: api.TypeString},
				"result2":  {Role: api.RoleOutput, Type: api.TypeNumber},
				"metadata": {Role: api.RoleOutput, Type: api.TypeObject},
			},
		}
		outputs := st.GetOutputArgs()
		as.Len(outputs, 3)
		as.Contains(outputs, api.Name("result1"))
		as.Contains(outputs, api.Name("result2"))
		as.Contains(outputs, api.Name("metadata"))
	})

	t.Run("no_outputs", func(t *testing.T) {
		st := &api.Step{
			ID:   "no-output",
			Name: "Terminal Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			Attributes: api.AttributeSpecs{
				"data": {Role: api.RoleRequired, Type: api.TypeString},
			},
		}
		as.Empty(st.GetOutputArgs())
	})
}

func TestSortedArgNames(t *testing.T) {
	as := assert.New(t)

	st := &api.Step{
		ID:   "sorted-args",
		Name: "Sorted Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080",
			},
		},
		Attributes: api.AttributeSpecs{
			"zebra":  {Role: api.RoleRequired, Type: api.TypeString},
			"apple":  {Role: api.RoleRequired, Type: api.TypeString},
			"mango":  {Role: api.RoleOptional, Type: api.TypeString},
			"banana": {Role: api.RoleOptional, Type: api.TypeString},
			"carrot": {
				Role:  api.RoleConst,
				Type:  api.TypeString,
				Const: &api.ConstConfig{Value: `"fixed"`},
			},
		},
	}

	sorted := st.SortedArgNames()
	as.Len(sorted, 5)
	as.Equal("apple", sorted[0])
	as.Equal("banana", sorted[1])
	as.Equal("carrot", sorted[2])
	as.Equal("mango", sorted[3])
	as.Equal("zebra", sorted[4])
}

func TestMappedArgNames(t *testing.T) {
	as := assert.New(t)

	st := &api.Step{
		ID:   "sorted-mapped-args",
		Name: "Sorted Mapped Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080",
			},
		},
		Attributes: api.AttributeSpecs{
			"outer_b": {
				Role: api.RoleRequired,
				Type: api.TypeString,
				Required: &api.RequiredConfig{
					Mapping: &api.MappingConfig{Name: "inner_b"},
				},
			},
			"outer_a": {
				Role: api.RoleOptional,
				Type: api.TypeString,
				Optional: &api.OptionalConfig{
					Mapping: &api.MappingConfig{Name: "inner_a"},
				},
			},
			"plain": {
				Role: api.RoleRequired,
				Type: api.TypeString,
			},
		},
	}

	sorted := st.SortedArgNames()
	as.Equal([]string{"inner_a", "inner_b", "plain"}, sorted)
}

func TestMultiArgNames(t *testing.T) {
	as := assert.New(t)

	st := &api.Step{
		ID:   "multi-args",
		Name: "Multi Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080",
			},
		},
		Attributes: api.AttributeSpecs{
			"users": {
				Role:     api.RoleRequired,
				Type:     api.TypeArray,
				Required: &api.RequiredConfig{ForEach: true},
			},
			"items": {
				Role:     api.RoleOptional,
				Type:     api.TypeArray,
				Optional: &api.OptionalConfig{ForEach: true},
			},
			"config": {
				Role: api.RoleRequired,
				Type: api.TypeObject,
			},
			"messages": {
				Role:     api.RoleOptional,
				Type:     api.TypeArray,
				Optional: &api.OptionalConfig{ForEach: true},
			},
		},
	}

	multiArgs := st.MultiArgNames()
	as.Len(multiArgs, 3)
	as.Contains(multiArgs, api.Name("users"))
	as.Contains(multiArgs, api.Name("items"))
	as.Contains(multiArgs, api.Name("messages"))
	as.NotContains(multiArgs, api.Name("config"))
}

func TestGetRequiredArgs(t *testing.T) {
	as := assert.New(t)

	st := &api.Step{
		ID:   "required-args",
		Name: "Required Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080",
			},
		},
		Attributes: api.AttributeSpecs{
			"user_id": {Role: api.RoleRequired, Type: api.TypeString},
			"email":   {Role: api.RoleRequired, Type: api.TypeString},
			"name":    {Role: api.RoleOptional, Type: api.TypeString},
			"result":  {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	requiredArgs := st.GetRequiredArgs()
	as.Len(requiredArgs, 2)
	as.Contains(requiredArgs, api.Name("user_id"))
	as.Contains(requiredArgs, api.Name("email"))
	as.NotContains(requiredArgs, api.Name("name"))
	as.NotContains(requiredArgs, api.Name("result"))
}

func TestGetOptionalArgs(t *testing.T) {
	as := assert.New(t)

	st := &api.Step{
		ID:   "optional-args",
		Name: "Optional Args Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080",
			},
		},
		Attributes: api.AttributeSpecs{
			"user_id": {Role: api.RoleRequired, Type: api.TypeString},
			"email":   {Role: api.RoleOptional, Type: api.TypeString},
			"name":    {Role: api.RoleOptional, Type: api.TypeString},
			"result":  {Role: api.RoleOutput, Type: api.TypeString},
		},
	}

	optionalArgs := st.GetOptionalArgs()
	as.Len(optionalArgs, 2)
	as.Contains(optionalArgs, api.Name("email"))
	as.Contains(optionalArgs, api.Name("name"))
	as.NotContains(optionalArgs, api.Name("user_id"))
	as.NotContains(optionalArgs, api.Name("result"))
}

func TestEqualHTTP(t *testing.T) {
	as := assert.New(t)

	config1 := &api.HTTPConfig{
		Invoke: api.HTTPAction{
			Endpoint: "http://localhost:8080",
			Timeout:  30,
		},
		Health: "http://localhost:8080/health",
	}

	config2 := &api.HTTPConfig{
		Invoke: api.HTTPAction{
			Endpoint: "http://localhost:8080",
			Timeout:  30,
		},
		Health: "http://localhost:8080/health",
	}

	config3 := &api.HTTPConfig{
		Invoke: api.HTTPAction{
			Endpoint: "http://localhost:9090",
			Timeout:  30,
		},
		Health: "http://localhost:8080/health",
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
		Language: api.ScriptLangLua,
		Script:   "return 1 + 2",
	}

	config2 := &api.ScriptConfig{
		Language: api.ScriptLangLua,
		Script:   "return 1 + 2",
	}

	config3 := &api.ScriptConfig{
		Language: api.ScriptLangLua,
		Script:   "return 2 + 2",
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
	config4 := &api.FlowConfig{
		Goals:   []api.StepID{"goal-1"},
		SpaceID: "restricted",
	}

	as.True(config1.Equal(config2))
	as.False(config1.Equal(config3))
	as.False(config1.Equal(config4))
	as.True((*api.FlowConfig)(nil).Equal(nil))
	as.False(config1.Equal(nil))
}

func TestFlowConfigWithGoals(t *testing.T) {
	as := assert.New(t)

	base := &api.FlowConfig{
		Goals: []api.StepID{"goal-1"},
	}

	updated := base.WithGoals("goal-2", "goal-3")
	as.Equal([]api.StepID{"goal-1"}, base.Goals)
	as.Equal([]api.StepID{"goal-2", "goal-3"}, updated.Goals)
}

func TestEqualWorkConfig(t *testing.T) {
	as := assert.New(t)

	config1 := &api.WorkConfig{
		Parallelism: 5,
		MaxRetries:  3,
		InitBackoff: 1000,
		MaxBackoff:  60000,
		BackoffType: api.BackoffTypeExponential,
	}

	config2 := &api.WorkConfig{
		Parallelism: 5,
		MaxRetries:  3,
		InitBackoff: 1000,
		MaxBackoff:  60000,
		BackoffType: api.BackoffTypeExponential,
	}

	config3 := &api.WorkConfig{
		Parallelism: 10,
		MaxRetries:  3,
		InitBackoff: 1000,
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
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080",
			},
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
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080",
			},
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
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080",
			},
		},
		Attributes: api.AttributeSpecs{
			"arg1": {Role: api.RoleRequired, Type: api.TypeString},
		},
	}

	as.True(step1.Equal(step2))
	as.False(step1.Equal(step3))
}

func TestEqualFields(t *testing.T) {
	as := assert.New(t)
	base := func() *api.Step {
		return &api.Step{
			ID:         "test",
			Name:       "Test",
			Type:       api.StepTypeSync,
			Attributes: api.AttributeSpecs{},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://step"},
			},
		}
	}

	tests := []struct {
		change func(*api.Step)
		name   string
	}{
		{name: "handling", change: func(s *api.Step) {
			s.Handling = api.HandlingMemoized
		}},
		{name: "flow", change: func(s *api.Step) {
			s.Flow = &api.FlowConfig{Goals: []api.StepID{"goal"}}
		}},
		{name: "script", change: func(s *api.Step) {
			s.Script = &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "x",
			}
		}},
		{name: "labels", change: func(s *api.Step) {
			s.Labels = api.Labels{"team": "core"}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := base()
			right := base()
			tt.change(right)
			as.False(left.Equal(right))
		})
	}
}

func TestCompensateConfig(t *testing.T) {
	as := assert.New(t)
	step := &api.Step{
		ID:       "test",
		Name:     "Test",
		Type:     api.StepTypeSync,
		Handling: api.HandlingCompensated,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://step/{input}"},
			Compensate: &api.HTTPAction{
				Endpoint: "http://step/{input}/{result}",
			},
		},
		Attributes: api.AttributeSpecs{
			"input": {
				Role:        api.RoleRequired,
				Type:        api.TypeString,
				Compensated: true,
				Required: &api.RequiredConfig{
					Mapping: &api.MappingConfig{Name: "input"},
				},
			},
			"result": {
				Role:        api.RoleOutput,
				Type:        api.TypeString,
				Compensated: true,
			},
			"fixed": {
				Role:  api.RoleConst,
				Type:  api.TypeString,
				Const: &api.ConstConfig{Value: `"fixed"`},
			},
		},
	}
	as.NoError(step.Validate())

	step.HTTP.Compensate.Endpoint = ""
	as.ErrorIs(step.Validate(), api.ErrCompensateRequired)
	step.HTTP.Compensate.Endpoint = "http://step/{input}/{result}"

	step.HTTP.Compensate.Method = "PATCH"
	as.ErrorIs(step.Validate(), api.ErrInvalidHTTPMethod)
	step.HTTP.Compensate.Method = ""

	step.HTTP.Invoke.Endpoint = "http://step/{missing}"
	as.ErrorIs(step.Validate(), api.ErrUnknownURLParam)
	step.HTTP.Invoke.Endpoint = "http://step/{input}"

	step.HTTP.Compensate.Endpoint = "http://step/{missing}"
	as.ErrorIs(step.Validate(), api.ErrUnknownURLParam)
	step.HTTP.Compensate.Endpoint = "http://step/{input}/{result}"
	step.Attributes["result"].Output = &api.OutputConfig{
		Mapping: &api.MappingConfig{Name: "input"},
	}
	as.ErrorIs(step.Validate(), api.ErrCompensateArgConflict)
	step.Attributes["result"].Output = nil

	step.WorkConfig = &api.WorkConfig{Parallelism: -1}
	as.ErrorIs(step.Validate(), api.ErrInvalidParallelism)
	step.WorkConfig = nil

	step.HTTP.Invoke.Timeout = 100
	step.HTTP.Compensate.Timeout = 200
	as.Equal(int64(200), step.HTTP.CompensateTimeout())

	_, err := step.HashKey()
	as.NoError(err)
}

func TestHandling(t *testing.T) {
	tests := []struct {
		change func(*api.Step)
		want   error
		name   string
	}{
		{
			name: "invalid",
			change: func(step *api.Step) {
				step.Handling = "invalid"
			},
			want: api.ErrInvalidHandling,
		},
		{
			name: "compensated_without_endpoint",
			change: func(step *api.Step) {
				step.Handling = api.HandlingCompensated
			},
			want: api.ErrCompensateRequired,
		},
		{
			name: "compensated_with_empty_endpoint",
			change: func(step *api.Step) {
				step.Handling = api.HandlingCompensated
				step.HTTP.Compensate = &api.HTTPAction{}
			},
			want: api.ErrCompensateRequired,
		},
		{
			name: "standard_with_endpoint",
			change: func(step *api.Step) {
				step.HTTP.Compensate = &api.HTTPAction{
					Endpoint: "http://example.com/undo",
				}
			},
			want: api.ErrCompensateHandling,
		},
		{
			name: "standard_with_compensated_attribute",
			change: func(step *api.Step) {
				step.Attributes["input"].Compensated = true
			},
			want: api.ErrAttributeCompensated,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			as := assert.New(t)
			step := helpers.NewTestStep()
			test.change(step)
			as.ErrorIs(step.Validate(), test.want)
		})
	}
}

func TestStepInvalidAttributes(t *testing.T) {
	as := assert.New(t)
	step := helpers.NewTestStep()
	step.Attributes["nil"] = nil
	as.ErrorIs(step.Validate(), api.ErrAttributeNil)

	step = helpers.NewTestStep()
	step.Attributes["bad"] = &api.AttributeSpec{Role: "bad"}
	as.ErrorIs(step.Validate(), api.ErrInvalidAttributeRole)
}

func TestLabelsEqualMismatch(t *testing.T) {
	as := assert.New(t)
	as.False(api.Labels{"a": "1"}.Equal(api.Labels{}))
	as.False(api.Labels{"a": "1"}.Equal(api.Labels{"a": "2"}))
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
		st := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			WorkConfig: &api.WorkConfig{
				InitBackoff: -1,
			},
		}
		as.StepInvalid(st, "backoff cannot be negative")
	})

	t.Run("max_backoff_too_small", func(t *testing.T) {
		st := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			WorkConfig: &api.WorkConfig{
				InitBackoff: 1000,
				MaxBackoff:  500,
			},
		}
		as.StepInvalid(st, "max_backoff")
	})

	t.Run("missing_backoff_type_uses_default", func(t *testing.T) {
		st := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries: 3,
			},
		}
		as.StepValid(st)
	})

	t.Run("invalid_backoff_type", func(t *testing.T) {
		st := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries:  3,
				BackoffType: "invalid",
			},
		}
		as.StepInvalid(st, "invalid backoff type")
	})

	t.Run("valid_work_config", func(t *testing.T) {
		st := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries:  3,
				InitBackoff: 1000,
				MaxBackoff:  60000,
				BackoffType: api.BackoffTypeExponential,
			},
		}
		as.StepValid(st)
	})
}

func TestStepWithDefaults(t *testing.T) {
	as := assert.New(t)

	defaults := &api.WorkConfig{
		MaxRetries:  10,
		InitBackoff: 1000,
		MaxBackoff:  60000,
		BackoffType: api.BackoffTypeExponential,
	}

	t.Run("nil_work_config", func(t *testing.T) {
		st := &api.Step{ID: "step", Name: "Step", Type: api.StepTypeSync}
		res := st.WithWorkDefaults(defaults)

		as.NotNil(res.WorkConfig)
		as.Equal(10, res.WorkConfig.MaxRetries)
		as.Equal(int64(1000), res.WorkConfig.InitBackoff)
		as.Equal(int64(60000), res.WorkConfig.MaxBackoff)
		as.Equal(api.BackoffTypeExponential, res.WorkConfig.BackoffType)
	})

	t.Run("fills_retry_fields_only", func(t *testing.T) {
		st := &api.Step{
			ID:   "step",
			Name: "Step",
			Type: api.StepTypeSync,
			WorkConfig: &api.WorkConfig{
				Parallelism: 4,
			},
		}
		res := st.WithWorkDefaults(defaults)

		as.NotNil(res.WorkConfig)
		as.Equal(4, res.WorkConfig.Parallelism)
		as.Equal(10, res.WorkConfig.MaxRetries)
		as.Equal(int64(1000), res.WorkConfig.InitBackoff)
		as.Equal(int64(60000), res.WorkConfig.MaxBackoff)
		as.Equal(api.BackoffTypeExponential, res.WorkConfig.BackoffType)
		as.Equal(0, st.WorkConfig.MaxRetries)
		as.Equal(int64(0), st.WorkConfig.InitBackoff)
		as.Equal(int64(0), st.WorkConfig.MaxBackoff)
		as.Equal("", st.WorkConfig.BackoffType)
	})

	t.Run("explicit_retry_values_preserved", func(t *testing.T) {
		st := &api.Step{
			ID:   "step",
			Name: "Step",
			Type: api.StepTypeSync,
			WorkConfig: &api.WorkConfig{
				MaxRetries:  2,
				InitBackoff: 250,
				MaxBackoff:  5000,
				BackoffType: api.BackoffTypeLinear,
			},
		}
		res := st.WithWorkDefaults(defaults)

		as.Equal(2, res.WorkConfig.MaxRetries)
		as.Equal(int64(250), res.WorkConfig.InitBackoff)
		as.Equal(int64(5000), res.WorkConfig.MaxBackoff)
		as.Equal(api.BackoffTypeLinear, res.WorkConfig.BackoffType)
	})
}

func TestStepCopy(t *testing.T) {
	as := assert.New(t)

	t.Run("shallow_copy", func(t *testing.T) {
		st := &api.Step{
			ID:   "copy-step",
			Name: "Copy Step",
			Type: api.StepTypeFlow,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"goal-a"},
			},
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return 1 + 2",
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return true",
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries:  3,
				Parallelism: 2,
			},
			Labels: api.Labels{
				"team": "core",
			},
			Attributes: api.AttributeSpecs{
				"input": {
					Role: api.RoleRequired,
					Type: api.TypeString,
					Required: &api.RequiredConfig{
						Mapping: &api.MappingConfig{
							Name: "arg_input",
							Script: &api.ScriptConfig{
								Language: api.ScriptLangJPath,
								Script:   "$.input",
							},
						},
					},
				},
			},
		}

		cpy := st.Copy()
		as.NotNil(cpy)
		as.NotSame(st, cpy)
		as.True(st.Equal(cpy))

		cpy.Name = "Changed Name"
		as.Equal(api.Name("Copy Step"), st.Name)

		cpy.HTTP.Invoke.Endpoint = "http://localhost:8081"
		cpy.Flow.Goals[0] = "goal-b"
		cpy.Script.Script = "(* 2 3)"
		cpy.Predicate.Script = "return false"
		cpy.WorkConfig.MaxRetries = 9
		cpy.Labels["team"] = "platform"
		cpy.Attributes["input"].Type = api.TypeNumber
		cpy.Attributes["input"].Required.Mapping.Name = "changed"
		cpy.Attributes["input"].Required.Mapping.Script.Script = "$.changed"

		as.Equal("http://localhost:8081", st.HTTP.Invoke.Endpoint)
		as.Equal(api.StepID("goal-b"), st.Flow.Goals[0])
		as.Equal("(* 2 3)", st.Script.Script)
		as.Equal("return false", st.Predicate.Script)
		as.Equal(9, st.WorkConfig.MaxRetries)
		as.Equal("platform", st.Labels["team"])
		as.Equal(api.TypeNumber, st.Attributes["input"].Type)
		as.Equal("changed", st.Attributes["input"].Required.Mapping.Name)
		as.Equal(
			"$.changed", st.Attributes["input"].Required.Mapping.Script.Script,
		)
	})
}

func TestStepEqualEdgeCases(t *testing.T) {
	as := assert.New(t)

	baseStep := &api.Step{
		ID:   "test",
		Name: "Test",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{
				Endpoint: "http://localhost:8080",
			},
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
				Language: api.ScriptLangLua,
				Script:   "return 1 + 2",
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return 1 + 2",
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   `return status == "ready"`,
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			Predicate: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   `return status == "pending"`,
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
		}
		step2 := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
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
				Invoke: api.HTTPAction{
					Endpoint: "http://localhost:8080",
				},
			},
			WorkConfig: &api.WorkConfig{
				MaxRetries:  5,
				BackoffType: api.BackoffTypeExponential,
			},
		}
		as.False(step1.Equal(step2))
	})
}

func TestStepHashKey(t *testing.T) {
	as := assert.New(t)

	t.Run("deterministic", func(t *testing.T) {
		s := &api.Step{
			ID:   "test-step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://test"},
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
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://a"},
			},
		}
		s2 := &api.Step{
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{Endpoint: "http://b"},
			},
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

	t.Run("includes_handling", func(t *testing.T) {
		s1 := &api.Step{
			Type: api.StepTypeSync,
		}
		s2 := &api.Step{
			Type:     api.StepTypeSync,
			Handling: api.HandlingMemoized,
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
				Language: "lua",
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

func TestMappingValidation(t *testing.T) {
	as := assert.New(t)

	t.Run("duplicate_input_inner_names", func(t *testing.T) {
		st := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://example.com",
					Timeout:  30 * api.Second,
				},
			},
			Attributes: api.AttributeSpecs{
				"email": {
					Role: api.RoleRequired,
					Required: &api.RequiredConfig{
						Mapping: &api.MappingConfig{Name: "user_email"},
					},
				},
				"contact": {
					Role: api.RoleRequired,
					Required: &api.RequiredConfig{
						Mapping: &api.MappingConfig{Name: "user_email"},
					},
				},
			},
		}
		err := st.Validate()
		as.ErrorIs(err, api.ErrDuplicateInnerName)
		as.ErrorContains(err, "user_email")
	})

	t.Run("duplicate_output_inner_names", func(t *testing.T) {
		st := &api.Step{
			ID:   "test",
			Name: "Test",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://example.com",
					Timeout:  30 * api.Second,
				},
			},
			Attributes: api.AttributeSpecs{
				"sent": {
					Role: api.RoleOutput,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{Name: "status"},
					},
				},
				"delivered": {
					Role: api.RoleOutput,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{Name: "status"},
					},
				},
			},
		}
		err := st.Validate()
		as.ErrorIs(err, api.ErrDuplicateInnerName)
		as.ErrorContains(err, "status")
	})

	t.Run("same_inner_name_single_compensation_side", func(t *testing.T) {
		st := &api.Step{
			ID:       "test",
			Name:     "Test",
			Type:     api.StepTypeSync,
			Handling: api.HandlingCompensated,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://example.com",
					Timeout:  30 * api.Second,
				},
				Compensate: &api.HTTPAction{
					Endpoint: "http://example.com/{data}",
				},
			},
			Attributes: api.AttributeSpecs{
				"user_data": {
					Role:        api.RoleRequired,
					Compensated: true,
					Required: &api.RequiredConfig{
						Mapping: &api.MappingConfig{Name: "data"},
					},
				},
				"result_data": {
					Role: api.RoleOutput,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{Name: "data"},
					},
				},
			},
		}
		err := st.Validate()
		as.NoError(err)
	})

	t.Run("same_inner_name_compensation_conflict", func(t *testing.T) {
		st := &api.Step{
			ID:       "test",
			Name:     "Test",
			Type:     api.StepTypeSync,
			Handling: api.HandlingCompensated,
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://example.com",
					Timeout:  30 * api.Second,
				},
				Compensate: &api.HTTPAction{
					Endpoint: "http://example.com/undo",
				},
			},
			Attributes: api.AttributeSpecs{
				"request": {
					Role:        api.RoleRequired,
					Compensated: true,
					Required: &api.RequiredConfig{
						Mapping: &api.MappingConfig{Name: "data"},
					},
				},
				"result": {
					Role:        api.RoleOutput,
					Compensated: true,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{Name: "data"},
					},
				},
			},
		}
		err := st.Validate()
		as.ErrorIs(err, api.ErrCompensateArgConflict)
	})
}
