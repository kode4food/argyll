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
				Role:  api.RoleConst,
				Type:  api.TypeString,
				Const: &api.ConstConfig{Value: `"fixed"`},
			},
			attrName:  "mode",
			expectErr: false,
		},
		{
			name: "const with invalid string default",
			spec: &api.AttributeSpec{
				Role:  api.RoleConst,
				Type:  api.TypeString,
				Const: &api.ConstConfig{Value: "fixed"},
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
				Role:     api.RoleOptional,
				Type:     api.TypeNumber,
				Optional: &api.OptionalConfig{Default: "42"},
			},
			attrName:  "count",
			expectErr: false,
		},
		{
			name: "optional with invalid number default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeNumber,
				Optional: &api.OptionalConfig{Default: "not-a-number"},
			},
			attrName:  "count",
			expectErr: true,
		},
		{
			name: "optional with valid boolean true default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeBoolean,
				Optional: &api.OptionalConfig{Default: "true"},
			},
			attrName:  "enabled",
			expectErr: false,
		},
		{
			name: "optional with valid boolean false default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeBoolean,
				Optional: &api.OptionalConfig{Default: "false"},
			},
			attrName:  "enabled",
			expectErr: false,
		},
		{
			name: "optional with invalid boolean default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeBoolean,
				Optional: &api.OptionalConfig{Default: "yes"},
			},
			attrName:  "enabled",
			expectErr: true,
		},
		{
			name: "optional with valid object default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeObject,
				Optional: &api.OptionalConfig{Default: `{"key": "value"}`},
			},
			attrName:  "config",
			expectErr: false,
		},
		{
			name: "optional with invalid object default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeObject,
				Optional: &api.OptionalConfig{Default: `[1, 2, 3]`},
			},
			attrName:  "config",
			expectErr: true,
		},
		{
			name: "optional with valid array default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeArray,
				Optional: &api.OptionalConfig{Default: `[1, 2, 3]`},
			},
			attrName:  "items",
			expectErr: false,
		},
		{
			name: "optional with invalid array default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeArray,
				Optional: &api.OptionalConfig{Default: `{"key": "value"}`},
			},
			attrName:  "items",
			expectErr: true,
		},
		{
			name: "optional with valid string default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeString,
				Optional: &api.OptionalConfig{Default: `"hello"`},
			},
			attrName:  "message",
			expectErr: false,
		},
		{
			name: "optional with invalid string default - unquoted",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeString,
				Optional: &api.OptionalConfig{Default: "hello"},
			},
			attrName:  "message",
			expectErr: true,
		},
		{
			name: "optional with valid null default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeNull,
				Optional: &api.OptionalConfig{Default: "null"},
			},
			attrName:  "optional_field",
			expectErr: false,
		},
		{
			name: "optional with invalid null default",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeNull,
				Optional: &api.OptionalConfig{Default: "nil"},
			},
			attrName:  "optional_field",
			expectErr: true,
		},
		{
			name: "optional with any type - valid number",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeAny,
				Optional: &api.OptionalConfig{Default: "42"},
			},
			attrName:  "data",
			expectErr: false,
		},
		{
			name: "optional with any type - valid object",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeAny,
				Optional: &api.OptionalConfig{Default: `{"key":"value"}`},
			},
			attrName:  "data",
			expectErr: false,
		},
		{
			name: "optional with any type - invalid JSON",
			spec: &api.AttributeSpec{
				Role:     api.RoleOptional,
				Type:     api.TypeAny,
				Optional: &api.OptionalConfig{Default: "not json"},
			},
			attrName:  "data",
			expectErr: true,
		},
		{
			name: "required with wrong role config should fail",
			spec: &api.AttributeSpec{
				Role:     api.RoleRequired,
				Type:     api.TypeString,
				Optional: &api.OptionalConfig{Default: `"value"`},
			},
			attrName:  "name",
			expectErr: true,
		},
		{
			name: "output with wrong role config should fail",
			spec: &api.AttributeSpec{
				Role:     api.RoleOutput,
				Type:     api.TypeString,
				Optional: &api.OptionalConfig{Default: `"value"`},
			},
			attrName:  "result",
			expectErr: true,
		},
		{
			name: "input with valid mapping",
			spec: &api.AttributeSpec{
				Role: api.RoleRequired,
				Type: api.TypeObject,
				Required: &api.RequiredConfig{
					Mapping: &api.MappingConfig{
						Script: &api.ScriptConfig{
							Language: api.ScriptLangJPath,
							Script:   "$.foo",
						},
					},
				},
			},
			attrName:  "input",
			expectErr: false,
		},
		{
			name: "output with valid mapping",
			spec: &api.AttributeSpec{
				Role: api.RoleOutput,
				Type: api.TypeAny,
				Output: &api.OutputConfig{
					Mapping: &api.MappingConfig{
						Script: &api.ScriptConfig{
							Language: api.ScriptLangJPath,
							Script:   "$..book",
						},
					},
				},
			},
			attrName:  "result",
			expectErr: false,
		},
		{
			name: "const with wrong role config should fail",
			spec: &api.AttributeSpec{
				Role:  api.RoleConst,
				Type:  api.TypeObject,
				Const: &api.ConstConfig{Value: "{}"},
				Output: &api.OutputConfig{
					Mapping: &api.MappingConfig{
						Script: &api.ScriptConfig{
							Language: api.ScriptLangJPath,
							Script:   "$.foo",
						},
					},
				},
			},
			attrName:  "input",
			expectErr: true,
		},
		{
			name: "invalid mapping syntax is engine-level validation",
			spec: &api.AttributeSpec{
				Role: api.RoleOptional,
				Type: api.TypeObject,
				Optional: &api.OptionalConfig{
					Mapping: &api.MappingConfig{
						Script: &api.ScriptConfig{
							Language: api.ScriptLangJPath,
							Script:   "$[?",
						},
					},
				},
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

func TestValidateDefaultErrorReason(t *testing.T) {
	spec := &api.AttributeSpec{
		Role:     api.RoleOptional,
		Type:     api.TypeString,
		Optional: &api.OptionalConfig{Default: "hello"},
	}

	err := spec.Validate("message")
	assert.ErrorIs(t, err, api.ErrInvalidDefaultValue)
	assert.ErrorIs(t, err, api.ErrDefaultJSON)

	spec = &api.AttributeSpec{
		Role:     api.RoleOptional,
		Type:     api.TypeObject,
		Optional: &api.OptionalConfig{Default: `[1, 2, 3]`},
	}

	err = spec.Validate("config")
	assert.ErrorIs(t, err, api.ErrInvalidDefaultValue)
	assert.ErrorIs(t, err, api.ErrDefaultObject)
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

func TestIsMeta(t *testing.T) {
	spec := &api.AttributeSpec{
		Role: api.RoleMeta,
		Meta: &api.MetaConfig{Key: "flow_id"},
	}
	assert.True(t, spec.IsMeta())
	assert.False(t, spec.IsConst())
	assert.False(t, spec.IsOptional())
	assert.False(t, spec.IsRequired())
	assert.False(t, spec.IsOutput())
}

func TestIsRuntimeInput(t *testing.T) {
	required := &api.AttributeSpec{Role: api.RoleRequired}
	assert.True(t, required.IsRuntimeInput())

	optional := &api.AttributeSpec{Role: api.RoleOptional}
	assert.True(t, optional.IsRuntimeInput())

	constSpec := &api.AttributeSpec{Role: api.RoleConst}
	assert.True(t, constSpec.IsRuntimeInput())

	meta := &api.AttributeSpec{Role: api.RoleMeta}
	assert.True(t, meta.IsRuntimeInput())

	output := &api.AttributeSpec{Role: api.RoleOutput}
	assert.False(t, output.IsRuntimeInput())
}

func TestValidateMetaRole(t *testing.T) {
	t.Run("meta_with_key", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleMeta,
			Meta: &api.MetaConfig{Key: "flow_id"},
		}
		assert.NoError(t, spec.Validate("meta_attr"))
	})

	t.Run("meta_without_config", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleMeta,
		}
		err := spec.Validate("meta_attr")
		assert.ErrorIs(t, err, api.ErrMetaKeyRequired)
	})

	t.Run("meta_with_empty_key", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleMeta,
			Meta: &api.MetaConfig{Key: ""},
		}
		err := spec.Validate("meta_attr")
		assert.ErrorIs(t, err, api.ErrMetaKeyRequired)
	})

	t.Run("meta_with_wrong_role_config", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleMeta,
			Meta:     &api.MetaConfig{Key: "flow_id"},
			Optional: &api.OptionalConfig{Default: `"value"`},
		}
		err := spec.Validate("meta_attr")
		assert.ErrorIs(t, err, api.ErrWrongRoleConfig)
	})
}

func TestAccessors(t *testing.T) {
	t.Run("RequiredMatch_with_match", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleRequired,
			Required: &api.RequiredConfig{
				Match: &api.ScriptConfig{Script: "$.foo"},
			},
		}
		assert.Equal(t, "$.foo", spec.RequiredMatch())
	})

	t.Run("RequiredMatch_no_match", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Required: &api.RequiredConfig{},
		}
		assert.Empty(t, spec.RequiredMatch())
	})

	t.Run("RequiredMatch_no_required", func(t *testing.T) {
		spec := &api.AttributeSpec{Role: api.RoleRequired}
		assert.Empty(t, spec.RequiredMatch())
	})

	t.Run("ConstValue_with_value", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:  api.RoleConst,
			Const: &api.ConstConfig{Value: `"fixed"`},
		}
		assert.Equal(t, `"fixed"`, spec.ConstValue())
	})

	t.Run("ConstValue_no_const", func(t *testing.T) {
		spec := &api.AttributeSpec{Role: api.RoleConst}
		assert.Empty(t, spec.ConstValue())
	})

	t.Run("MetaKey_with_key", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleMeta,
			Meta: &api.MetaConfig{Key: "flow_id"},
		}
		assert.Equal(t, "flow_id", spec.MetaKey())
	})

	t.Run("MetaKey_no_meta", func(t *testing.T) {
		spec := &api.AttributeSpec{Role: api.RoleMeta}
		assert.Empty(t, spec.MetaKey())
	})

	t.Run("OptionalDefault_with_default", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleOptional,
			Optional: &api.OptionalConfig{Default: `"hello"`},
		}
		assert.Equal(t, `"hello"`, spec.OptionalDefault())
	})

	t.Run("OptionalDefault_no_optional", func(t *testing.T) {
		spec := &api.AttributeSpec{Role: api.RoleOptional}
		assert.Empty(t, spec.OptionalDefault())
	})

	t.Run("OptionalDeadline_with_deadline", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleOptional,
			Optional: &api.OptionalConfig{Deadline: 5000},
		}
		assert.EqualValues(t, 5000, spec.OptionalDeadline())
	})

	t.Run("OptionalDeadline_no_optional", func(t *testing.T) {
		spec := &api.AttributeSpec{Role: api.RoleOptional}
		assert.EqualValues(t, 0, spec.OptionalDeadline())
	})

	t.Run("Collect_required_explicit", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Required: &api.RequiredConfig{Collect: api.InputCollectAll},
		}
		assert.Equal(t, api.InputCollectAll, spec.Collect())
	})

	t.Run("Collect_required_default", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Required: &api.RequiredConfig{},
		}
		assert.Equal(t, api.InputCollectFirst, spec.Collect())
	})

	t.Run("Collect_optional_explicit", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleOptional,
			Optional: &api.OptionalConfig{Collect: api.InputCollectLast},
		}
		assert.Equal(t, api.InputCollectLast, spec.Collect())
	})

	t.Run("Collect_other_role_defaults_to_first", func(t *testing.T) {
		spec := &api.AttributeSpec{Role: api.RoleOutput}
		assert.Equal(t, api.InputCollectFirst, spec.Collect())
	})
}

func TestMetaConfigEqual(t *testing.T) {
	t.Run("both_nil", func(t *testing.T) {
		spec1 := &api.AttributeSpec{Role: api.RoleMeta}
		spec2 := &api.AttributeSpec{Role: api.RoleMeta}
		assert.True(t, spec1.Equal(spec2))
	})

	t.Run("one_nil", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleMeta,
			Meta: &api.MetaConfig{Key: "flow_id"},
		}
		spec2 := &api.AttributeSpec{Role: api.RoleMeta}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("same_key", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleMeta,
			Meta: &api.MetaConfig{Key: "flow_id"},
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleMeta,
			Meta: &api.MetaConfig{Key: "flow_id"},
		}
		assert.True(t, spec1.Equal(spec2))
	})

	t.Run("different_key", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleMeta,
			Meta: &api.MetaConfig{Key: "flow_id"},
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleMeta,
			Meta: &api.MetaConfig{Key: "step_id"},
		}
		assert.False(t, spec1.Equal(spec2))
	})
}

func TestConstConfigEqual(t *testing.T) {
	t.Run("one_nil", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role:  api.RoleConst,
			Const: &api.ConstConfig{Value: `"x"`},
		}
		spec2 := &api.AttributeSpec{Role: api.RoleConst}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("same_value", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role:  api.RoleConst,
			Const: &api.ConstConfig{Value: `"x"`},
		}
		spec2 := &api.AttributeSpec{
			Role:  api.RoleConst,
			Const: &api.ConstConfig{Value: `"x"`},
		}
		assert.True(t, spec1.Equal(spec2))
	})

	t.Run("different_value", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role:  api.RoleConst,
			Const: &api.ConstConfig{Value: `"x"`},
		}
		spec2 := &api.AttributeSpec{
			Role:  api.RoleConst,
			Const: &api.ConstConfig{Value: `"y"`},
		}
		assert.False(t, spec1.Equal(spec2))
	})
}

func TestEqual(t *testing.T) {
	spec1 := &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}

	spec2 := &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}

	spec3 := &api.AttributeSpec{
		Role: api.RoleOptional,
		Type: api.TypeString,
	}

	assert.True(t, spec1.Equal(spec2))
	assert.False(t, spec1.Equal(spec3))
}

func TestEqualEdgeCases(t *testing.T) {
	t.Run("different_for_each", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Type:     api.TypeArray,
			Required: &api.RequiredConfig{ForEach: true},
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeArray,
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
			Role:     api.RoleOptional,
			Type:     api.TypeString,
			Optional: &api.OptionalConfig{Default: `"value1"`},
		}
		spec2 := &api.AttributeSpec{
			Role:     api.RoleOptional,
			Type:     api.TypeString,
			Optional: &api.OptionalConfig{Default: `"value2"`},
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("different_mapping", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeAny,
			Output: &api.OutputConfig{
				Mapping: &api.MappingConfig{
					Script: &api.ScriptConfig{
						Language: api.ScriptLangJPath,
						Script:   "$.foo",
					},
				},
			},
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeAny,
			Output: &api.OutputConfig{
				Mapping: &api.MappingConfig{
					Script: &api.ScriptConfig{
						Language: api.ScriptLangJPath,
						Script:   "$.bar",
					},
				},
			},
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("all_fields_different", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeNumber,
			Optional: &api.OptionalConfig{
				Default: "42",
				ForEach: true,
			},
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("empty_defaults_equal", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
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
	t.Run("valid_input_collect_values", func(t *testing.T) {
		for _, collect := range []api.InputCollect{
			api.InputCollectFirst,
			api.InputCollectLast,
			api.InputCollectAll,
			api.InputCollectSome,
			api.InputCollectNone,
		} {
			spec := &api.AttributeSpec{
				Role:     api.RoleRequired,
				Type:     api.TypeString,
				Required: &api.RequiredConfig{Collect: collect},
			}
			err := spec.Validate(api.Name(collect))
			assert.NoError(t, err)
		}
	})

	t.Run("invalid_input_collect", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Type:     api.TypeString,
			Required: &api.RequiredConfig{Collect: "invalid"},
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrInvalidInputCollect)
	})

	t.Run("for_each_with_type_any", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Type:     api.TypeAny,
			Required: &api.RequiredConfig{ForEach: true},
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})

	t.Run("for_each_with_empty_type", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Type:     "",
			Required: &api.RequiredConfig{ForEach: true},
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})

	t.Run("for_each_with_non_array_type", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Type:     api.TypeString,
			Required: &api.RequiredConfig{ForEach: true},
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrForEachRequiresArray)
	})

	t.Run("for_each_with_output_role", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleOutput,
			Type:     api.TypeArray,
			Required: &api.RequiredConfig{ForEach: true},
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrWrongRoleConfig)
	})

	t.Run("for_each_with_const_not_allowed", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleConst,
			Type:     api.TypeArray,
			Const:    &api.ConstConfig{Value: `["a", "b"]`},
			Required: &api.RequiredConfig{ForEach: true},
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrWrongRoleConfig)
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
			Role:     api.RoleRequired,
			Type:     api.TypeString,
			Optional: &api.OptionalConfig{Default: `"value"`},
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrWrongRoleConfig)
	})

	t.Run("default_with_output_role", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleOutput,
			Type:     api.TypeString,
			Optional: &api.OptionalConfig{Default: `"value"`},
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrWrongRoleConfig)
	})

	t.Run("const_requires_value", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleConst,
			Type: api.TypeString,
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrConstValueRequired)
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
			Role:     api.RoleOptional,
			Type:     api.TypeNumber,
			Optional: &api.OptionalConfig{Default: `"not a number"`},
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrInvalidDefaultValue)
	})

	t.Run("valid_optional_with_null_default", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleOptional,
			Type:     api.TypeNull,
			Optional: &api.OptionalConfig{Default: "null"},
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})

	t.Run("const_with_wrong_role_config", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:  api.RoleConst,
			Type:  api.TypeObject,
			Const: &api.ConstConfig{Value: "{}"},
			Output: &api.OutputConfig{
				Mapping: &api.MappingConfig{
					Script: &api.ScriptConfig{
						Language: api.ScriptLangJPath,
						Script:   "$.foo",
					},
				},
			},
		}
		err := spec.Validate("test_arg")
		assert.ErrorIs(t, err, api.ErrWrongRoleConfig)
	})

	t.Run("invalid_mapping_syntax_is_allowed_here", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeObject,
			Required: &api.RequiredConfig{
				Mapping: &api.MappingConfig{
					Script: &api.ScriptConfig{
						Language: api.ScriptLangJPath,
						Script:   "$[?",
					},
				},
			},
		}
		err := spec.Validate("test_arg")
		assert.NoError(t, err)
	})
}

func TestMappingConfigValidation(t *testing.T) {
	t.Run("name_only", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
			Required: &api.RequiredConfig{
				Mapping: &api.MappingConfig{Name: "service_param"},
			},
		}
		assert.NoError(t, spec.Validate("test"))
	})

	t.Run("script_only", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
			Required: &api.RequiredConfig{
				Mapping: &api.MappingConfig{
					Script: &api.ScriptConfig{
						Language: api.ScriptLangJPath,
						Script:   "$.value",
					},
				},
			},
		}
		assert.NoError(t, spec.Validate("test"))
	})

	t.Run("both_name_and_script", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
			Required: &api.RequiredConfig{
				Mapping: &api.MappingConfig{
					Name: "api_field",
					Script: &api.ScriptConfig{
						Language: api.ScriptLangJPath,
						Script:   "$.data.value",
					},
				},
			},
		}
		assert.NoError(t, spec.Validate("test"))
	})

	t.Run("empty_mapping_fails", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Type:     api.TypeString,
			Required: &api.RequiredConfig{Mapping: &api.MappingConfig{}},
		}
		err := spec.Validate("test")
		assert.ErrorIs(t, err, api.ErrInvalidMappingConfig)
	})
}

func TestDeadlineValidation(t *testing.T) {
	t.Run("optional_with_valid_deadline", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"default"`,
				Deadline: 5000,
			},
		}
		err := spec.Validate("test_attr")
		assert.NoError(t, err)
	})

	t.Run("optional_with_zero_deadline_allowed", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"default"`,
				Deadline: 0,
			},
		}
		err := spec.Validate("test_attr")
		assert.NoError(t, err)
	})

	t.Run("optional_with_max_deadline", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"default"`,
				Deadline: api.MaxAttributeDeadline,
			},
		}
		err := spec.Validate("test_attr")
		assert.NoError(t, err)
	})

	t.Run("optional_with_deadline_below_min", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"default"`,
				Deadline: -1,
			},
		}
		err := spec.Validate("test_attr")
		assert.ErrorIs(t, err, api.ErrInvalidAttributeDeadline)
	})

	t.Run("optional_with_deadline_above_max", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"default"`,
				Deadline: api.MaxAttributeDeadline + 1,
			},
		}
		err := spec.Validate("test_attr")
		assert.ErrorIs(t, err, api.ErrInvalidAttributeDeadline)
	})

	t.Run("required_with_deadline_not_allowed", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleRequired,
			Type:     api.TypeString,
			Optional: &api.OptionalConfig{Deadline: 5000},
		}
		err := spec.Validate("test_attr")
		assert.ErrorIs(t, err, api.ErrWrongRoleConfig)
	})

	t.Run("const_with_wrong_role_config", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleConst,
			Type:     api.TypeString,
			Const:    &api.ConstConfig{Value: `"const"`},
			Optional: &api.OptionalConfig{Deadline: 5000},
		}
		err := spec.Validate("test_attr")
		assert.ErrorIs(t, err, api.ErrWrongRoleConfig)
	})

	t.Run("output_with_deadline_not_allowed", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role:     api.RoleOutput,
			Type:     api.TypeString,
			Optional: &api.OptionalConfig{Deadline: 5000},
		}
		err := spec.Validate("test_attr")
		assert.ErrorIs(t, err, api.ErrWrongRoleConfig)
	})

	t.Run("optional_without_deadline", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"default"`,
				Deadline: 0,
			},
		}
		err := spec.Validate("test_attr")
		assert.NoError(t, err)
	})

	t.Run("deadline_one_year_max", func(t *testing.T) {
		spec := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"default"`,
				Deadline: 365 * 24 * 60 * 60 * 1000,
			},
		}
		err := spec.Validate("test")
		assert.NoError(t, err)
	})
}

func TestDeadlineInEqual(t *testing.T) {
	t.Run("same_deadline_equal", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"hello"`,
				Deadline: 5000,
			},
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"hello"`,
				Deadline: 5000,
			},
		}
		assert.True(t, spec1.Equal(spec2))
	})

	t.Run("different_deadline_not_equal", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"hello"`,
				Deadline: 5000,
			},
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"hello"`,
				Deadline: 10000,
			},
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("one_with_deadline_not_equal", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"hello"`,
				Deadline: 5000,
			},
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"hello"`,
				Deadline: 0,
			},
		}
		assert.False(t, spec1.Equal(spec2))
	})

	t.Run("both_without_deadline_equal", func(t *testing.T) {
		spec1 := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"hello"`,
				Deadline: 0,
			},
		}
		spec2 := &api.AttributeSpec{
			Role: api.RoleOptional,
			Type: api.TypeString,
			Optional: &api.OptionalConfig{
				Default:  `"hello"`,
				Deadline: 0,
			},
		}
		assert.True(t, spec1.Equal(spec2))
	})
}
