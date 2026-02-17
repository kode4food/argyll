package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestValidateDefault(t *testing.T) {
	tests := []struct {
		name      string
		spec      *api.AttributeSpec
		attrName  api.Name
		expectErr bool
	}{
		{
			name: "const with valid string default",
			spec: &api.AttributeSpec{
				Role:    api.RoleConst,
				Type:    api.TypeString,
				Default: `"fixed"`,
			},
			attrName:  "mode",
			expectErr: false,
		},
		{
			name: "const with invalid string default",
			spec: &api.AttributeSpec{
				Role:    api.RoleConst,
				Type:    api.TypeString,
				Default: "fixed",
			},
			attrName:  "mode",
			expectErr: true,
		},
		{
			name: "const with missing default",
			spec: &api.AttributeSpec{
				Role: api.RoleConst,
				Type: api.TypeString,
			},
			attrName:  "mode",
			expectErr: true,
		},
		{
			name: "optional with valid number default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeNumber,
				Default: "42",
			},
			attrName:  "count",
			expectErr: false,
		},
		{
			name: "optional with invalid number default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeNumber,
				Default: "not-a-number",
			},
			attrName:  "count",
			expectErr: true,
		},
		{
			name: "optional with valid boolean true default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeBoolean,
				Default: "true",
			},
			attrName:  "enabled",
			expectErr: false,
		},
		{
			name: "optional with valid boolean false default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeBoolean,
				Default: "false",
			},
			attrName:  "enabled",
			expectErr: false,
		},
		{
			name: "optional with invalid boolean default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeBoolean,
				Default: "yes",
			},
			attrName:  "enabled",
			expectErr: true,
		},
		{
			name: "optional with valid object default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeObject,
				Default: `{"key": "value"}`,
			},
			attrName:  "config",
			expectErr: false,
		},
		{
			name: "optional with invalid object default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeObject,
				Default: `[1, 2, 3]`,
			},
			attrName:  "config",
			expectErr: true,
		},
		{
			name: "optional with valid array default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeArray,
				Default: `[1, 2, 3]`,
			},
			attrName:  "items",
			expectErr: false,
		},
		{
			name: "optional with invalid array default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeArray,
				Default: `{"key": "value"}`,
			},
			attrName:  "items",
			expectErr: true,
		},
		{
			name: "optional with valid string default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeString,
				Default: `"hello"`,
			},
			attrName:  "message",
			expectErr: false,
		},
		{
			name: "optional with invalid string default - unquoted",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeString,
				Default: "hello",
			},
			attrName:  "message",
			expectErr: true,
		},
		{
			name: "optional with valid null default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeNull,
				Default: "null",
			},
			attrName:  "optional_field",
			expectErr: false,
		},
		{
			name: "optional with invalid null default",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeNull,
				Default: "nil",
			},
			attrName:  "optional_field",
			expectErr: true,
		},
		{
			name: "optional with any type - valid number",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeAny,
				Default: "42",
			},
			attrName:  "data",
			expectErr: false,
		},
		{
			name: "optional with any type - valid object",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeAny,
				Default: `{"key":"value"}`,
			},
			attrName:  "data",
			expectErr: false,
		},
		{
			name: "optional with any type - invalid JSON",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeAny,
				Default: "not json",
			},
			attrName:  "data",
			expectErr: true,
		},
		{
			name: "required with default should fail",
			spec: &api.AttributeSpec{
				Role:    api.RoleRequired,
				Type:    api.TypeString,
				Default: `"value"`,
			},
			attrName:  "name",
			expectErr: true,
		},
		{
			name: "output with default should fail",
			spec: &api.AttributeSpec{
				Role:    api.RoleOutput,
				Type:    api.TypeString,
				Default: `"value"`,
			},
			attrName:  "result",
			expectErr: true,
		},
		{
			name: "input with valid mapping",
			spec: &api.AttributeSpec{
				Role:    api.RoleRequired,
				Type:    api.TypeObject,
				Mapping: "$.foo",
			},
			attrName:  "input",
			expectErr: false,
		},
		{
			name: "output with valid mapping",
			spec: &api.AttributeSpec{
				Role:    api.RoleOutput,
				Type:    api.TypeAny,
				Mapping: "$..book",
			},
			attrName:  "result",
			expectErr: false,
		},
		{
			name: "const with mapping should fail",
			spec: &api.AttributeSpec{
				Role:    api.RoleConst,
				Type:    api.TypeObject,
				Default: "{}",
				Mapping: "$.foo",
			},
			attrName:  "input",
			expectErr: true,
		},
		{
			name: "invalid mapping syntax is engine-level validation",
			spec: &api.AttributeSpec{
				Role:    api.RoleOptional,
				Type:    api.TypeObject,
				Mapping: "$[?",
			},
			attrName:  "input",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate(tt.attrName)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsRequired(t *testing.T) {
	required := &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}
	assert.True(t, required.IsRequired())

	optional := &api.AttributeSpec{
		Role: api.RoleOptional,
		Type: api.TypeString,
	}
	assert.False(t, optional.IsRequired())

	output := &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: api.TypeString,
	}
	assert.False(t, output.IsRequired())
}

func TestIsConst(t *testing.T) {
	spec := &api.AttributeSpec{
		Role: api.RoleConst,
		Type: api.TypeString,
	}
	assert.True(t, spec.IsConst())
	assert.False(t, spec.IsOptional())
	assert.False(t, spec.IsRequired())
	assert.False(t, spec.IsOutput())
}

func TestEqual(t *testing.T) {
	spec1 := &api.AttributeSpec{
		Role:    api.RoleRequired,
		Type:    api.TypeString,
		Default: `"hello"`,
		ForEach: false,
	}

	spec2 := &api.AttributeSpec{
		Role:    api.RoleRequired,
		Type:    api.TypeString,
		Default: `"hello"`,
		ForEach: false,
	}

	spec3 := &api.AttributeSpec{
		Role:    api.RoleOptional,
		Type:    api.TypeString,
		Default: `"hello"`,
		ForEach: false,
	}

	assert.True(t, spec1.Equal(spec2))
	assert.False(t, spec1.Equal(spec3))
}

func TestEqualEdgeCases(t *testing.T) {
	t.Run("both_nil", func(t *testing.T) {
		var spec1 *api.AttributeSpec
		var spec2 *api.AttributeSpec
		assert.True(t, spec1.Equal(spec2))
	})

	t.Run("one_nil_one_not", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
		var spec2 *api.AttributeSpec
		assert.False(t, spec1.Equal(spec2))
		assert.False(t, spec2.Equal(spec1))
	})

	t.Run("different_for_each", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    api.TypeArray,
			ForEach: true,
		}
		spec2 := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    api.TypeArray,
			ForEach: false,
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("different_type", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeNumber,
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("different_default", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role:    api.RoleOptional,
			Type:    api.TypeString,
			Default: `"value1"`,
		}
		spec2 := &api.AttributeSpec{
			Role:    api.RoleOptional,
			Type:    api.TypeString,
			Default: `"value2"`,
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("different_mapping", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role:    api.RoleOutput,
			Type:    api.TypeAny,
			Mapping: "$.foo",
		}
		spec2 := &api.AttributeSpec{
			Role:    api.RoleOutput,
			Type:    api.TypeAny,
			Mapping: "$.bar",
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("all_fields_different", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    api.TypeString,
			Default: "",
			ForEach: false,
		}
		spec2 := &api.AttributeSpec{
			Role:    api.RoleOptional,
			Type:    api.TypeNumber,
			Default: "42",
			ForEach: true,
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("empty_defaults_equal", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    api.TypeString,
			Default: "",
		}
		spec2 := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    api.TypeString,
			Default: "",
		}
		assert.True(t, spec1.Equal(spec2))
	})
}

func TestAttributeSpecsEqual(t *testing.T) {
	specs1 := api.AttributeSpecs{
		"arg1": {Role: api.RoleRequired, Type: api.TypeString},
		"arg2": {Role: api.RoleOptional, Type: api.TypeNumber},
	}
	specs2 := api.AttributeSpecs{
		"arg1": {Role: api.RoleRequired, Type: api.TypeString},
		"arg2": {Role: api.RoleOptional, Type: api.TypeNumber},
	}
	specs3 := api.AttributeSpecs{
		"arg1": {Role: api.RoleRequired, Type: api.TypeString},
		"arg2": {Role: api.RoleOptional, Type: api.TypeBoolean},
	}

	assert.True(t, specs1.Equal(specs2))
	assert.False(t, specs1.Equal(specs3))
}

func TestValidateEdgeCases(t *testing.T) {
	t.Run("for_each_with_type_any", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    api.TypeAny,
			ForEach: true,
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})

	t.Run("for_each_with_empty_type", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    "",
			ForEach: true,
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})

	t.Run("for_each_with_non_array_type", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    api.TypeString,
			ForEach: true,
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrForEachRequiresArray)
	})

	t.Run("for_each_with_output_role", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleOutput,
			Type:    api.TypeArray,
			ForEach: true,
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrForEachNotAllowedOutput)
	})

	t.Run("for_each_with_const_role", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleConst,
			Type:    api.TypeArray,
			ForEach: true,
			Default: `["a", "b"]`,
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})

	t.Run("invalid_role", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: "invalid_role",
			Type: api.TypeString,
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrInvalidAttributeRole)
	})

	t.Run("invalid_type", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: "invalid_type",
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrInvalidAttributeType)
	})

	t.Run("default_with_required_role", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    api.TypeString,
			Default: `"value"`,
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrDefaultNotAllowed)
	})

	t.Run("default_with_output_role", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleOutput,
			Type:    api.TypeString,
			Default: `"value"`,
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrDefaultNotAllowed)
	})

	t.Run("const_requires_default", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleConst,
			Type: api.TypeString,
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrDefaultRequired)
	})

	t.Run("empty_type_allowed", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: "",
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})

	t.Run("default_with_mismatched_type", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleOptional,
			Type:    api.TypeNumber,
			Default: `"not a number"`,
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrInvalidDefaultValue)
	})

	t.Run("valid_optional_with_null_default", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleOptional,
			Type:    api.TypeNull,
			Default: "null",
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})

	t.Run("const_with_mapping_not_allowed", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleConst,
			Type:    api.TypeObject,
			Default: "{}",
			Mapping: "$.foo",
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrMappingNotAllowed)
	})

	t.Run("invalid_mapping_syntax_is_allowed_here", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:    api.RoleRequired,
			Type:    api.TypeObject,
			Mapping: "$[?",
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})
}
