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
		Role     AttributeRole   `json:"role"`
		Type     AttributeType   `json:"type,omitempty"`
		Required *RequiredConfig `json:"required,omitempty"`
		Optional *OptionalConfig `json:"optional,omitempty"`
		Const    *ConstConfig    `json:"const,omitempty"`
		Output   *OutputConfig   `json:"output,omitempty"`
	}

	// RequiredConfig configures a required input attribute
	RequiredConfig struct {
		Collect InputCollect   `json:"collect,omitempty"`
		ForEach bool           `json:"for_each,omitempty"`
		Match   *ScriptConfig  `json:"match,omitempty"`
		Mapping *MappingConfig `json:"mapping,omitempty"`
	}

	// OptionalConfig configures an optional input attribute
	OptionalConfig struct {
		Collect  InputCollect   `json:"collect,omitempty"`
		ForEach  bool           `json:"for_each,omitempty"`
		Default  string         `json:"default,omitempty"`
		Deadline int64          `json:"deadline,omitempty"`
		Mapping  *MappingConfig `json:"mapping,omitempty"`
	}

	// ConstConfig carries the fixed value for a const attribute
	ConstConfig struct {
		Value string `json:"value"`
	}

	// OutputConfig configures an output attribute
	OutputConfig struct {
		Mapping *MappingConfig `json:"mapping,omitempty"`
	}

	// MappingConfig defines parameter name mapping and value transformation
	MappingConfig struct {
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
	ErrWrongRoleConfig      = errors.New(
		"config type does not match attribute role",
	)
	ErrConstValueRequired = errors.New(
		"const value required for const attribute",
	)
	ErrForEachRequiresArray = errors.New(
		"for_each processing requires an array attribute type",
	)
	ErrInvalidDefaultValue      = errors.New("invalid default value for type")
	ErrInvalidMappingConfig     = errors.New("invalid attribute mapping")
	ErrDuplicateInnerName       = errors.New("duplicate mapped parameter name")
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
	if !validAttributeRoles.Contains(s.Role) {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrInvalidAttributeRole, s.Role, name)
	}

	if s.Type != "" && !validAttributeTypes.Contains(s.Type) {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrInvalidAttributeType, s.Type, name)
	}

	if err := s.validateRoleConfig(name); err != nil {
		return err
	}

	switch s.Role {
	case RoleRequired:
		return s.validateRequired(name)
	case RoleOptional:
		return s.validateOptional(name)
	case RoleConst:
		return s.validateConst(name)
	case RoleOutput:
		return s.validateOutput(name)
	}
	return nil
}

func (s *AttributeSpec) validateRoleConfig(name Name) error {
	switch s.Role {
	case RoleRequired:
		if s.Optional != nil || s.Const != nil || s.Output != nil {
			return fmt.Errorf("%w: %q", ErrWrongRoleConfig, name)
		}
	case RoleOptional:
		if s.Required != nil || s.Const != nil || s.Output != nil {
			return fmt.Errorf("%w: %q", ErrWrongRoleConfig, name)
		}
	case RoleConst:
		if s.Required != nil || s.Optional != nil || s.Output != nil {
			return fmt.Errorf("%w: %q", ErrWrongRoleConfig, name)
		}
	case RoleOutput:
		if s.Required != nil || s.Optional != nil || s.Const != nil {
			return fmt.Errorf("%w: %q", ErrWrongRoleConfig, name)
		}
	}
	return nil
}

func (s *AttributeSpec) validateRequired(name Name) error {
	cfg := s.Required
	if cfg == nil {
		return nil
	}
	if cfg.ForEach && s.Type != TypeArray && s.Type != TypeAny && s.Type != "" {
		return fmt.Errorf("%w: type %s for attribute %q",
			ErrForEachRequiresArray, s.Type, name)
	}
	if cfg.Collect != "" && !validInputCollect.Contains(cfg.Collect) {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrInvalidInputCollect, cfg.Collect, name)
	}
	return validateMapping(cfg.Mapping, name)
}

func (s *AttributeSpec) validateOptional(name Name) error {
	cfg := s.Optional
	if cfg == nil {
		return nil
	}
	if cfg.ForEach && s.Type != TypeArray && s.Type != TypeAny && s.Type != "" {
		return fmt.Errorf("%w: type %s for attribute %q",
			ErrForEachRequiresArray, s.Type, name)
	}
	if cfg.Default != "" && s.Type != "" {
		if err := validateDefaultValue(cfg.Default, s.Type); err != nil {
			return fmt.Errorf("%w for attribute %q: %w",
				ErrInvalidDefaultValue, name, err)
		}
	}
	if cfg.Deadline < MinAttributeDeadline ||
		cfg.Deadline > MaxAttributeDeadline {
		return fmt.Errorf("%w: deadline %d for attribute %q",
			ErrInvalidAttributeDeadline, cfg.Deadline, name)
	}
	if cfg.Collect != "" && !validInputCollect.Contains(cfg.Collect) {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrInvalidInputCollect, cfg.Collect, name)
	}
	return validateMapping(cfg.Mapping, name)
}

func (s *AttributeSpec) validateConst(name Name) error {
	if s.Const == nil || s.Const.Value == "" {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrConstValueRequired, s.Role, name)
	}
	if s.Type != "" {
		if err := validateDefaultValue(s.Const.Value, s.Type); err != nil {
			return fmt.Errorf("%w for attribute %q: %w",
				ErrInvalidDefaultValue, name, err)
		}
	}
	return nil
}

func (s *AttributeSpec) validateOutput(name Name) error {
	if s.Output == nil {
		return nil
	}
	return validateMapping(s.Output.Mapping, name)
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

// Mapping returns the mapping for the attribute, if any
func (s *AttributeSpec) Mapping() *MappingConfig {
	switch s.Role {
	case RoleRequired:
		if s.Required != nil {
			return s.Required.Mapping
		}
	case RoleOptional:
		if s.Optional != nil {
			return s.Optional.Mapping
		}
	case RoleOutput:
		if s.Output != nil {
			return s.Output.Mapping
		}
	}
	return nil
}

func (s *AttributeSpec) ForEach() bool {
	switch s.Role {
	case RoleRequired:
		return s.Required != nil && s.Required.ForEach
	case RoleOptional:
		return s.Optional != nil && s.Optional.ForEach
	}
	return false
}

func (s *AttributeSpec) RequiredMatch() string {
	if s.Required == nil || s.Required.Match == nil {
		return ""
	}
	return s.Required.Match.Script
}

func (s *AttributeSpec) ConstValue() string {
	if s.Const == nil {
		return ""
	}
	return s.Const.Value
}

func (s *AttributeSpec) OptionalDefault() string {
	if s.Optional == nil {
		return ""
	}
	return s.Optional.Default
}

func (s *AttributeSpec) OptionalDeadline() int64 {
	if s.Optional == nil {
		return 0
	}
	return s.Optional.Deadline
}

func (s *AttributeSpec) Collect() InputCollect {
	switch s.Role {
	case RoleRequired:
		if s.Required != nil && s.Required.Collect != "" {
			return s.Required.Collect
		}
	case RoleOptional:
		if s.Optional != nil && s.Optional.Collect != "" {
			return s.Optional.Collect
		}
	}
	return InputCollectFirst
}

// Equal returns true if two attribute specs are equal
func (s *AttributeSpec) Equal(other *AttributeSpec) bool {
	return s.Role == other.Role &&
		s.Type == other.Type &&
		requiredConfigsEqual(s.Required, other.Required) &&
		optionalConfigsEqual(s.Optional, other.Optional) &&
		constConfigsEqual(s.Const, other.Const) &&
		outputConfigsEqual(s.Output, other.Output)
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

func requiredConfigsEqual(a, b *RequiredConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Collect == b.Collect &&
		a.ForEach == b.ForEach &&
		scriptsEqual(a.Match, b.Match) &&
		mappingsEqual(a.Mapping, b.Mapping)
}

func optionalConfigsEqual(a, b *OptionalConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Collect == b.Collect &&
		a.ForEach == b.ForEach &&
		a.Default == b.Default &&
		a.Deadline == b.Deadline &&
		mappingsEqual(a.Mapping, b.Mapping)
}

func constConfigsEqual(a, b *ConstConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Value == b.Value
}

func outputConfigsEqual(a, b *OutputConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return mappingsEqual(a.Mapping, b.Mapping)
}

func mappingsEqual(a, b *MappingConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Name == b.Name && scriptsEqual(a.Script, b.Script)
}

func validateMapping(m *MappingConfig, name Name) error {
	if m == nil {
		return nil
	}
	if m.Name == "" && m.Script == nil {
		return fmt.Errorf("%w: %q", ErrInvalidMappingConfig, name)
	}
	return nil
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
