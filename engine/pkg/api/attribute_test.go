package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDefaultValue(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		attrType  AttributeType
		expectErr bool
	}{
		{
			name:      "valid JSON string",
			value:     `"hello world"`,
			attrType:  TypeString,
			expectErr: false,
		},
		{
			name:      "invalid string - unquoted",
			value:     "hello world",
			attrType:  TypeString,
			expectErr: true,
		},
		{
			name:      "valid number integer",
			value:     "42",
			attrType:  TypeNumber,
			expectErr: false,
		},
		{
			name:      "valid number float",
			value:     "3.14",
			attrType:  TypeNumber,
			expectErr: false,
		},
		{
			name:      "invalid number",
			value:     "not a number",
			attrType:  TypeNumber,
			expectErr: true,
		},
		{
			name:      "valid boolean true",
			value:     "true",
			attrType:  TypeBoolean,
			expectErr: false,
		},
		{
			name:      "valid boolean false",
			value:     "false",
			attrType:  TypeBoolean,
			expectErr: false,
		},
		{
			name:      "invalid boolean",
			value:     "yes",
			attrType:  TypeBoolean,
			expectErr: true,
		},
		{
			name:      "valid object",
			value:     `{"key": "value"}`,
			attrType:  TypeObject,
			expectErr: false,
		},
		{
			name:      "invalid object - array",
			value:     `[1, 2, 3]`,
			attrType:  TypeObject,
			expectErr: true,
		},
		{
			name:      "invalid object - malformed JSON",
			value:     `{key: value}`,
			attrType:  TypeObject,
			expectErr: true,
		},
		{
			name:      "valid array",
			value:     `[1, 2, 3]`,
			attrType:  TypeArray,
			expectErr: false,
		},
		{
			name:      "invalid array - object",
			value:     `{"key": "value"}`,
			attrType:  TypeArray,
			expectErr: true,
		},
		{
			name:      "invalid array - malformed JSON",
			value:     `[1, 2, 3`,
			attrType:  TypeArray,
			expectErr: true,
		},
		{
			name:      "valid null",
			value:     "null",
			attrType:  TypeNull,
			expectErr: false,
		},
		{
			name:      "invalid null",
			value:     "nil",
			attrType:  TypeNull,
			expectErr: true,
		},
		{
			name:      "any type accepts valid JSON string",
			value:     `"whatever"`,
			attrType:  TypeAny,
			expectErr: false,
		},
		{
			name:      "any type accepts valid JSON number",
			value:     "42",
			attrType:  TypeAny,
			expectErr: false,
		},
		{
			name:      "any type accepts valid JSON object",
			value:     `{"key":"value"}`,
			attrType:  TypeAny,
			expectErr: false,
		},
		{
			name:      "any type accepts valid JSON array",
			value:     `[1,2,3]`,
			attrType:  TypeAny,
			expectErr: false,
		},
		{
			name:      "any type rejects invalid JSON",
			value:     "not valid json",
			attrType:  TypeAny,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDefaultValue(tt.value, tt.attrType)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDefault(t *testing.T) {
	tests := []struct {
		name      string
		spec      *AttributeSpec
		attrName  Name
		expectErr bool
	}{
		{
			name: "optional with valid number default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeNumber,
				Default: "42",
			},
			attrName:  "count",
			expectErr: false,
		},
		{
			name: "optional with invalid number default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeNumber,
				Default: "not-a-number",
			},
			attrName:  "count",
			expectErr: true,
		},
		{
			name: "optional with valid boolean true default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeBoolean,
				Default: "true",
			},
			attrName:  "enabled",
			expectErr: false,
		},
		{
			name: "optional with valid boolean false default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeBoolean,
				Default: "false",
			},
			attrName:  "enabled",
			expectErr: false,
		},
		{
			name: "optional with invalid boolean default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeBoolean,
				Default: "yes",
			},
			attrName:  "enabled",
			expectErr: true,
		},
		{
			name: "optional with valid object default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeObject,
				Default: `{"key": "value"}`,
			},
			attrName:  "config",
			expectErr: false,
		},
		{
			name: "optional with invalid object default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeObject,
				Default: `[1, 2, 3]`,
			},
			attrName:  "config",
			expectErr: true,
		},
		{
			name: "optional with valid array default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeArray,
				Default: `[1, 2, 3]`,
			},
			attrName:  "items",
			expectErr: false,
		},
		{
			name: "optional with invalid array default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeArray,
				Default: `{"key": "value"}`,
			},
			attrName:  "items",
			expectErr: true,
		},
		{
			name: "optional with valid string default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeString,
				Default: `"hello"`,
			},
			attrName:  "message",
			expectErr: false,
		},
		{
			name: "optional with invalid string default - unquoted",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeString,
				Default: "hello",
			},
			attrName:  "message",
			expectErr: true,
		},
		{
			name: "optional with valid null default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeNull,
				Default: "null",
			},
			attrName:  "optional_field",
			expectErr: false,
		},
		{
			name: "optional with invalid null default",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeNull,
				Default: "nil",
			},
			attrName:  "optional_field",
			expectErr: true,
		},
		{
			name: "optional with any type - valid number",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeAny,
				Default: "42",
			},
			attrName:  "data",
			expectErr: false,
		},
		{
			name: "optional with any type - valid object",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeAny,
				Default: `{"key":"value"}`,
			},
			attrName:  "data",
			expectErr: false,
		},
		{
			name: "optional with any type - invalid JSON",
			spec: &AttributeSpec{
				Role:    RoleOptional,
				Type:    TypeAny,
				Default: "not json",
			},
			attrName:  "data",
			expectErr: true,
		},
		{
			name: "required with default should fail",
			spec: &AttributeSpec{
				Role:    RoleRequired,
				Type:    TypeString,
				Default: `"value"`,
			},
			attrName:  "name",
			expectErr: true,
		},
		{
			name: "output with default should fail",
			spec: &AttributeSpec{
				Role:    RoleOutput,
				Type:    TypeString,
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
	required := &AttributeSpec{
		Role: RoleRequired,
		Type: TypeString,
	}
	assert.True(t, required.IsRequired())

	optional := &AttributeSpec{
		Role: RoleOptional,
		Type: TypeString,
	}
	assert.False(t, optional.IsRequired())

	output := &AttributeSpec{
		Role: RoleOutput,
		Type: TypeString,
	}
	assert.False(t, output.IsRequired())
}

func TestEqual(t *testing.T) {
	spec1 := &AttributeSpec{
		Role:    RoleRequired,
		Type:    TypeString,
		Default: `"hello"`,
		ForEach: false,
	}

	spec2 := &AttributeSpec{
		Role:    RoleRequired,
		Type:    TypeString,
		Default: `"hello"`,
		ForEach: false,
	}

	spec3 := &AttributeSpec{
		Role:    RoleOptional,
		Type:    TypeString,
		Default: `"hello"`,
		ForEach: false,
	}

	assert.True(t, spec1.Equal(spec2))
	assert.False(t, spec1.Equal(spec3))
}
