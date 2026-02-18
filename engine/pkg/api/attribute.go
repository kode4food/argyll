package api

import (
	"errors"
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// AttributeSpec defines the specification for a step attribute
	AttributeSpec struct {
		Role    AttributeRole     `json:"role"`
		Type    AttributeType     `json:"type"`
		Default string            `json:"default,omitempty"`
		Mapping *AttributeMapping `json:"mapping,omitempty"`
		ForEach bool              `json:"for_each,omitempty"`
	}

	// AttributeMapping defines parameter name mapping and value transformation
	AttributeMapping struct {
		Name   string        `json:"name,omitempty"`
		Script *ScriptConfig `json:"script,omitempty"`
	}

	// AttributeSpecs is a map of attribute names to their specifications
	AttributeSpecs map[Name]*AttributeSpec

	// AttributeTypes is a map of attribute names to their data types
	AttributeTypes map[Name]AttributeType

	// AttributeRole defines whether an attribute is required, optional, const,
	// or an output
	AttributeRole string

	// AttributeType defines the data type of an attribute
	AttributeType string
)

const (
	RoleRequired AttributeRole = "required"
	RoleOptional AttributeRole = "optional"
	RoleConst    AttributeRole = "const"
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
	ErrDefaultRequired = errors.New(
		"default value required for const attribute",
	)
	ErrForEachRequiresArray = errors.New(
		"for_each processing requires an array attribute type",
	)
	ErrForEachNotAllowedOutput = errors.New(
		"for_each processing requires an input attribute type",
	)
	ErrInvalidDefaultValue = errors.New("invalid default value for type")
	ErrMappingNotAllowed   = errors.New(
		"mapping is not allowed for const attributes",
	)
	ErrInvalidAttributeMapping = errors.New("invalid attribute mapping")
	ErrDuplicateInnerName      = errors.New("duplicate mapped parameter name")
)

var (
	validAttributeRoles = util.SetOf(
		RoleRequired,
		RoleOptional,
		RoleConst,
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
func (s *AttributeSpec) Validate(name Name) error {
	if !validAttributeRoles.Contains(s.Role) {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrInvalidAttributeRole, s.Role, name)
	}

	if s.IsConst() && s.Default == "" {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrDefaultRequired, s.Role, name)
	}

	if s.Default != "" && !s.IsOptional() && !s.IsConst() {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrDefaultNotAllowed, s.Role, name)
	}

	if s.ForEach {
		if s.Type != TypeArray && s.Type != TypeAny && s.Type != "" {
			return fmt.Errorf("%w: type %s for attribute %q",
				ErrForEachRequiresArray, s.Type, name)
		}
		if s.IsOutput() {
			return fmt.Errorf("%w: %q", ErrForEachNotAllowedOutput, name)
		}
	}

	if s.Mapping != nil {
		if s.IsConst() {
			return fmt.Errorf("%w: %q", ErrMappingNotAllowed, name)
		}
		if s.Mapping.Name == "" && s.Mapping.Script == nil {
			return fmt.Errorf("%w: %q", ErrInvalidAttributeMapping, name)
		}
	}

	if s.Type != "" {
		if !validAttributeTypes.Contains(s.Type) {
			return fmt.Errorf("%w: %s for attribute %q",
				ErrInvalidAttributeType, s.Type, name)
		}
	}

	if s.Default != "" && s.Type != "" {
		if err := validateDefaultValue(s.Default, s.Type); err != nil {
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
func (s *AttributeSpec) IsInput() bool {
	return s.Role == RoleRequired || s.Role == RoleOptional
}

// IsRuntimeInput returns true if the attribute is passed into a step
func (s *AttributeSpec) IsRuntimeInput() bool {
	return s.IsInput() || s.IsConst()
}

// IsOutput returns true if the attribute is an output
func (s *AttributeSpec) IsOutput() bool {
	return s.Role == RoleOutput
}

// IsRequired returns true if the attribute is required
func (s *AttributeSpec) IsRequired() bool {
	return s.Role == RoleRequired
}

// IsOptional returns true if the attribute is optional
func (s *AttributeSpec) IsOptional() bool {
	return s.Role == RoleOptional
}

// IsConst returns true if the attribute is a constant input
func (s *AttributeSpec) IsConst() bool {
	return s.Role == RoleConst
}

// Equal returns true if two attribute specs are equal
func (s *AttributeSpec) Equal(other *AttributeSpec) bool {
	if s == nil && other == nil {
		return true
	}
	if s == nil || other == nil {
		return false
	}
	return s.Role == other.Role &&
		s.Type == other.Type &&
		s.ForEach == other.ForEach &&
		s.Default == other.Default &&
		mappingsEqual(s.Mapping, other.Mapping)
}

// Equal returns true if two attribute spec maps are equal
func (a AttributeSpecs) Equal(other AttributeSpecs) bool {
	if len(a) != len(other) {
		return false
	}
	for name, spec := range a {
		otherSpec, ok := other[name]
		if !ok || !spec.Equal(otherSpec) {
			return false
		}
	}
	return true
}

func mappingsEqual(a, b *AttributeMapping) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Name == b.Name && scriptsEqual(a.Script, b.Script)
}

func scriptsEqual(a, b *ScriptConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Language == b.Language && a.Script == b.Script
}
