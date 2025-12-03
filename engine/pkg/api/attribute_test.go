package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestValidateDefault(t *testing.T) {
	tests := []struct {
		name      string
		spec      *api.AttributeSpec
		attrName  api.Name
		expectErr bool
	}{
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
