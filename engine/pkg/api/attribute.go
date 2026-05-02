package api

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// AttributeSpec defines the specification for a step attribute
	AttributeSpec struct {
		// Common
		Role    AttributeRole     `json:"role"`
		Type    AttributeType     `json:"type"`
		Mapping *AttributeMapping `json:"mapping,omitempty"`

		// Role specific
		Input *InputConfig `json:"input,omitempty"`
	}

	// InputConfig configures runtime input collection and fallback behavior
	InputConfig struct {
		Collect  InputCollect `json:"collect,omitempty"`
		ForEach  bool         `json:"for_each,omitempty"`
		Default  string       `json:"default,omitempty"`
		Deadline int64        `json:"deadline,omitempty"`
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

	// InputCollect defines how an input collects upstream provider outputs
	InputCollect string

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
	InputCollectFirst InputCollect = "first"
	InputCollectLast  InputCollect = "last"
	InputCollectAll   InputCollect = "all"
	InputCollectSome  InputCollect = "some"
	InputCollectNone  InputCollect = "none"
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
		"default value requires optional or const attribute",
	)
	ErrDefaultRequired = errors.New("const value required for const attribute")
	ErrInputNotAllowed = errors.New(
		"input config is only allowed for runtime input attributes",
	)
	ErrForEachRequiresArray = errors.New(
		"for_each processing requires an array attribute type",
	)
	ErrInvalidDefaultValue = errors.New("invalid default value for type")
	ErrMappingNotAllowed   = errors.New(
		"mapping is not allowed for const attributes",
	)
	ErrInvalidAttributeMapping = errors.New("invalid attribute mapping")
	ErrDuplicateInnerName      = errors.New("duplicate mapped parameter name")
	ErrDeadlineNotAllowed      = errors.New(
		"deadline is only allowed on optional input attributes",
	)
	ErrInvalidAttributeDeadline = errors.New(
		"deadline must be between 0 and 1 year in milliseconds",
	)
	ErrInvalidInputCollect = errors.New("invalid input collect")

	ErrDefaultJSON    = errors.New("must be valid JSON")
	ErrDefaultString  = errors.New("must be a valid JSON string")
	ErrDefaultNumber  = errors.New("must be a valid number")
	ErrDefaultBoolean = errors.New(`must be "true" or "false"`)
	ErrDefaultObject  = errors.New("must be valid JSON object")
	ErrDefaultArray   = errors.New("must be valid JSON array")
	ErrDefaultNull    = errors.New(`must be "null"`)
)

const (
	MinAttributeDeadline = 0
	MaxAttributeDeadline = 365 * 24 * 60 * 60 * 1000
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

	validInputCollect = util.SetOf(
		InputCollectFirst,
		InputCollectLast,
		InputCollectAll,
		InputCollectSome,
		InputCollectNone,
	)
)

// Validate checks if the attribute specification is valid
func (s *AttributeSpec) Validate(name Name) error {
	input := s.Input
	def := s.InputDefault()

	if !validAttributeRoles.Contains(s.Role) {
		return fmt.Errorf("%w: %s for attribute %q", ErrInvalidAttributeRole,
			s.Role, name)
	}

	if input != nil && !s.IsRuntimeInput() {
		return fmt.Errorf("%w: %q", ErrInputNotAllowed, name)
	}

	if s.IsConst() && def == "" {
		return fmt.Errorf("%w: %s for attribute %q", ErrDefaultRequired,
			s.Role, name)
	}

	if def != "" && !s.IsOptional() && !s.IsConst() {
		return fmt.Errorf("%w: %s for attribute %q", ErrDefaultNotAllowed,
			s.Role, name)
	}

	if input != nil && input.ForEach {
		if !s.IsInput() && !s.IsConst() {
			return fmt.Errorf("%w: %q", ErrInputNotAllowed, name)
		}
		if s.Type != TypeArray && s.Type != TypeAny && s.Type != "" {
			return fmt.Errorf("%w: type %s for attribute %q",
				ErrForEachRequiresArray, s.Type, name)
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

	if def != "" && s.Type != "" {
		if err := validateDefaultValue(def, s.Type); err != nil {
			return fmt.Errorf("%w for attribute %q: %w",
				ErrInvalidDefaultValue, name, err)
		}
	}

	if input != nil && input.Collect != "" {
		if !validInputCollect.Contains(input.Collect) {
			return fmt.Errorf("%w: %s for attribute %q",
				ErrInvalidInputCollect, input.Collect, name)
		}
		if !s.IsInput() && input.Collect != InputCollectFirst {
			return fmt.Errorf("%w: %q", ErrInputNotAllowed, name)
		}
	}

	deadline := s.InputDeadline()
	if deadline < MinAttributeDeadline || deadline > MaxAttributeDeadline {
		return fmt.Errorf("%w: deadline %d for attribute %q",
			ErrInvalidAttributeDeadline, deadline, name)
	}

	if deadline > 0 && !s.IsOptional() {
		return fmt.Errorf("%w: %q", ErrDeadlineNotAllowed, name)
	}

	return nil
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

func (s *AttributeSpec) InputForEach() bool {
	return s.Input != nil && s.Input.ForEach
}

func (s *AttributeSpec) InputDefault() string {
	if s.Input == nil {
		return ""
	}
	return s.Input.Default
}

func (s *AttributeSpec) InputDeadline() int64 {
	if s.Input == nil {
		return 0
	}
	return s.Input.Deadline
}

func (s *AttributeSpec) InputCollect() InputCollect {
	if s.Input == nil || s.Input.Collect == "" {
		return InputCollectFirst
	}
	return s.Input.Collect
}

// Equal returns true if two attribute specs are equal
func (s *AttributeSpec) Equal(other *AttributeSpec) bool {
	return s.Role == other.Role &&
		s.Type == other.Type &&
		inputsEqual(s.Input, other.Input) &&
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

func inputsEqual(a, b *InputConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Collect == b.Collect &&
		a.ForEach == b.ForEach &&
		a.Default == b.Default &&
		a.Deadline == b.Deadline
}

func validateDefaultValue(data string, attrType AttributeType) error {
	var val any
	if err := json.Unmarshal([]byte(data), &val); err != nil {
		return ErrDefaultJSON
	}

	if attrType == TypeAny {
		return nil
	}

	switch attrType {
	case TypeString:
		if _, ok := val.(string); !ok {
			return ErrDefaultString
		}
		return nil

	case TypeNumber:
		if _, ok := val.(float64); !ok {
			return ErrDefaultNumber
		}
		return nil

	case TypeBoolean:
		if _, ok := val.(bool); !ok {
			return ErrDefaultBoolean
		}
		return nil

	case TypeObject:
		if _, ok := val.(map[string]any); !ok {
			return ErrDefaultObject
		}
		return nil

	case TypeArray:
		if _, ok := val.([]any); !ok {
			return ErrDefaultArray
		}
		return nil

	case TypeNull:
		if val != nil {
			return ErrDefaultNull
		}
		return nil

	default:
		return nil
	}
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
