package api_test

import (
	"testing"

	"github.com/kode4food/spuds/engine/internal/assert"
	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/pkg/api"
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
				Version: "1.0.0",
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
				Version: "1.0.0",
			},
			expectError:   true,
			errorContains: "name empty",
		},
		{
			name: "missing_http_config",
			step: &api.Step{
				ID:      "test-id",
				Name:    "Test",
				Type:    api.StepTypeSync,
				Version: "1.0.0",
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
				Version: "1.0.0",
			},
			expectError:   true,
			errorContains: "endpoint empty",
		},
		{
			name: "missing_version",
			step: &api.Step{
				ID:   "test-id",
				Name: "Test",
				Type: api.StepTypeSync,
				HTTP: &api.HTTPConfig{
					Endpoint: "http://localhost:8080",
				},
			},
			expectError:   true,
			errorContains: "version empty",
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
				Version: "1.0.0",
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
				Version: "1.0.0",
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
				ID:      "test-id",
				Name:    "Test Script",
				Type:    api.StepTypeScript,
				Version: "1.0.0",
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
				Version: "1.0.0",
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
				Version: "1.0.0",
			},
			expectError:   true,
			errorContains: "script language empty",
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
			Version: "1.0.0",
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
			Version: "1.0.0",
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
		Version: "1.0.0",
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
