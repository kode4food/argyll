package api

import (
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
)

type (
	AttributeSpec struct {
		Role    AttributeRole `json:"role"`
		Type    AttributeType `json:"type,omitempty"`
		Default string        `json:"default,omitempty"`
		ForEach bool          `json:"for_each,omitempty"`
	}

	AttributeSpecs map[Name]*AttributeSpec
	AttributeRole  string
	AttributeType  string
)

const (
	RoleRequired AttributeRole = "required"
	RoleOptional AttributeRole = "optional"
	RoleOutput   AttributeRole = "output"
)

const (
	TypeString  AttributeType = "string"
	TypeNumber  AttributeType = "number"
	TypeBoolean AttributeType = "boolean"
	TypeObject  AttributeType = "object"
	TypeArray   AttributeType = "array"
	TypeNull    AttributeType = "null"
	TypeAny     AttributeType = "any"
)

var (
	ErrInvalidAttributeRole = errors.New("invalid attribute role")
	ErrInvalidAttributeType = errors.New("invalid attribute type")
	ErrDefaultNotAllowed    = errors.New(
		"default value requires an optional attribute",
	)
	ErrForEachRequiresArray = errors.New(
		"for_each processing requires an array attribute type",
	)
	ErrForEachNotAllowedOutput = errors.New(
		"for_each processing requires an input attribute type",
	)
	ErrInvalidDefaultValue = errors.New("invalid default value for type")
)

var (
	validAttributeRoles = map[AttributeRole]struct{}{
		RoleRequired: {},
		RoleOptional: {},
		RoleOutput:   {},
	}

	validAttributeTypes = map[AttributeType]struct{}{
		TypeString:  {},
		TypeNumber:  {},
		TypeBoolean: {},
		TypeObject:  {},
		TypeArray:   {},
		TypeNull:    {},
		TypeAny:     {},
	}
)

func (as *AttributeSpec) Validate(name Name) error {
	if _, ok := validAttributeRoles[as.Role]; !ok {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrInvalidAttributeRole, as.Role, name)
	}

	if as.Default != "" && as.Role != RoleOptional {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrDefaultNotAllowed, as.Role, name)
	}

	if as.ForEach {
		if as.Type != TypeArray && as.Type != TypeAny && as.Type != "" {
			return fmt.Errorf("%w: type %s for attribute %q",
				ErrForEachRequiresArray, as.Type, name)
		}
		if as.Role == RoleOutput {
			return fmt.Errorf("%w: %q", ErrForEachNotAllowedOutput, name)
		}
	}

	if as.Type != "" {
		if _, ok := validAttributeTypes[as.Type]; !ok {
			return fmt.Errorf("%w: %s for attribute %q",
				ErrInvalidAttributeType, as.Type, name)
		}
	}

	if as.Default != "" && as.Type != "" {
		if err := validateDefaultValue(as.Default, as.Type); err != nil {
			return fmt.Errorf("%w for attribute %q: %v",
				ErrInvalidDefaultValue, name, err)
		}
	}

	return nil
}

func validateDefaultValue(value string, attrType AttributeType) error {
	if !gjson.Valid(value) {
		return errors.New("must be valid JSON")
	}

	if attrType == TypeAny {
		return nil
	}

	result := gjson.Parse(value)

	switch attrType {
	case TypeString:
		if result.Type != gjson.String {
			return errors.New("must be a valid JSON string")
		}
		return nil

	case TypeNumber:
		if result.Type != gjson.Number {
			return errors.New("must be a valid number")
		}
		return nil

	case TypeBoolean:
		if result.Type != gjson.True && result.Type != gjson.False {
			return errors.New("must be \"true\" or \"false\"")
		}
		return nil

	case TypeObject:
		if !result.IsObject() {
			return errors.New("must be valid JSON object")
		}
		return nil

	case TypeArray:
		if !result.IsArray() {
			return errors.New("must be valid JSON array")
		}
		return nil

	case TypeNull:
		if result.Type != gjson.Null {
			return errors.New("must be \"null\"")
		}
		return nil

	default:
		return nil
	}
}

func (as *AttributeSpec) IsInput() bool {
	return as.Role == RoleRequired || as.Role == RoleOptional
}

func (as *AttributeSpec) IsOutput() bool {
	return as.Role == RoleOutput
}

func (as *AttributeSpec) Equal(other *AttributeSpec) bool {
	if as == nil && other == nil {
		return true
	}
	if as == nil || other == nil {
		return false
	}
	return as.Role == other.Role &&
		as.Type == other.Type &&
		as.ForEach == other.ForEach &&
		as.Default == other.Default
}
