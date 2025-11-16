package api

import (
	"errors"
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/kode4food/spuds/engine/pkg/util"
)

type (
	// AttributeSpec defines the specification for a step attribute
	AttributeSpec struct {
		Role    AttributeRole `json:"role"`
		Type    AttributeType `json:"type,omitempty"`
		Default string        `json:"default,omitempty"`
		ForEach bool          `json:"for_each,omitempty"`
	}

	// AttributeSpecs is a map of attribute names to their specifications
	AttributeSpecs map[Name]*AttributeSpec

	// AttributeRole defines whether an attribute is required, optional, or an
	// output
	AttributeRole string

	// AttributeType defines the data type of an attribute
	AttributeType string
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
	validAttributeRoles = util.SetOf(
		RoleRequired,
		RoleOptional,
		RoleOutput,
	)

	validAttributeTypes = util.SetOf(
		TypeString,
		TypeNumber,
		TypeBoolean,
		TypeObject,
		TypeArray,
		TypeNull,
		TypeAny,
	)
)

// Validate checks if the attribute specification is valid
func (as *AttributeSpec) Validate(name Name) error {
	if !validAttributeRoles.Contains(as.Role) {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrInvalidAttributeRole, as.Role, name)
	}

	if as.Default != "" && !as.IsOptional() {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrDefaultNotAllowed, as.Role, name)
	}

	if as.ForEach {
		if as.Type != TypeArray && as.Type != TypeAny && as.Type != "" {
			return fmt.Errorf("%w: type %s for attribute %q",
				ErrForEachRequiresArray, as.Type, name)
		}
		if as.IsOutput() {
			return fmt.Errorf("%w: %q", ErrForEachNotAllowedOutput, name)
		}
	}

	if as.Type != "" {
		if !validAttributeTypes.Contains(as.Type) {
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

// IsInput returns true if the attribute is an input (required or optional)
func (as *AttributeSpec) IsInput() bool {
	return as.Role == RoleRequired || as.Role == RoleOptional
}

// IsOutput returns true if the attribute is an output
func (as *AttributeSpec) IsOutput() bool {
	return as.Role == RoleOutput
}

// IsRequired returns true if the attribute is required
func (as *AttributeSpec) IsRequired() bool {
	return as.Role == RoleRequired
}

// IsOptional returns true if the attribute is optional
func (as *AttributeSpec) IsOptional() bool {
	return as.Role == RoleOptional
}

// Equal returns true if two attribute specs are equal
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
