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
		Required    *RequiredConfig `json:"required,omitempty"`
		Optional    *OptionalConfig `json:"optional,omitempty"`
		Const       *ConstConfig    `json:"const,omitempty"`
		Meta        *MetaConfig     `json:"meta,omitempty"`
		Feed        *FeedConfig     `json:"feed,omitempty"`
		Output      *OutputConfig   `json:"output,omitempty"`
		Role        AttributeRole   `json:"role"`
		Compensated bool            `json:"compensated,omitempty"`
	}

	// RequiredConfig configures a required input attribute
	RequiredConfig struct {
		Match   *ScriptConfig  `json:"match,omitempty"`
		Mapping *MappingConfig `json:"mapping,omitempty"`
		Collect InputCollect   `json:"collect,omitempty"`
		ForEach bool           `json:"for_each,omitempty"`
	}

	// OptionalConfig configures an optional input attribute
	OptionalConfig struct {
		Match    *ScriptConfig  `json:"match,omitempty"`
		Mapping  *MappingConfig `json:"mapping,omitempty"`
		Collect  InputCollect   `json:"collect,omitempty"`
		Default  string         `json:"default,omitempty"`
		Deadline int64          `json:"deadline,omitempty"`
		ForEach  bool           `json:"for_each,omitempty"`
	}

	// ConstConfig carries the fixed value for a const attribute
	ConstConfig struct {
		Value string `json:"value"`
	}

	// MetaConfig carries the metadata key for a meta attribute
	MetaConfig struct {
		Key string `json:"key"`
	}

	// FeedConfig names the Feed a feed attribute reads
	FeedConfig struct {
		ID FeedID `json:"id"`
	}

	// OutputConfig configures an output attribute
	OutputConfig struct {
		Mapping *MappingConfig `json:"mapping,omitempty"`
	}

	// MappingConfig defines parameter name mapping and value transformation
	MappingConfig struct {
		Script *ScriptConfig `json:"script,omitempty"`
		Name   string        `json:"name,omitempty"`
	}

	// AttributeSpecs is a map of attribute names to their specifications
	AttributeSpecs Map[Name, *AttributeSpec]

	// AttributeRole defines whether an attribute is required, optional, const,
	// or an output
	AttributeRole string

	// InputCollect defines how an input collects upstream provider outputs
	InputCollect string
)

const (
	RoleRequired AttributeRole = "required"
	RoleOptional AttributeRole = "optional"
	RoleConst    AttributeRole = "const"
	RoleMeta     AttributeRole = "meta"
	RoleFeed     AttributeRole = "feed"
	RoleOutput   AttributeRole = "output"
)

const (
	InputCollectFirst InputCollect = "first"
	InputCollectLast  InputCollect = "last"
	InputCollectAll   InputCollect = "all"
	InputCollectSome  InputCollect = "some"
	InputCollectNone  InputCollect = "none"
)

var (
	ErrInvalidAttributeRole = errors.New("invalid attribute role")
	ErrWrongRoleConfig      = errors.New(
		"config type does not match attribute role",
	)
	ErrConstValueRequired = errors.New(
		"const value required for const attribute",
	)
	ErrMetaKeyRequired = errors.New(
		"meta key required for meta attribute",
	)
	ErrFeedIDRequired = errors.New(
		"feed ID required for feed attribute",
	)
	ErrForEachRequiresCollect = errors.New(
		"for_each processing requires collect all or some",
	)
	ErrInvalidDefaultValue      = errors.New("invalid default value")
	ErrInvalidMappingConfig     = errors.New("invalid attribute mapping")
	ErrDuplicateInnerName       = errors.New("duplicate mapped parameter name")
	ErrInvalidAttributeDeadline = errors.New(
		"deadline must be between 0 and 1 year in milliseconds",
	)
	ErrInvalidInputCollect = errors.New("invalid input collect")

	ErrDefaultJSON = errors.New("must be valid JSON")
)

const (
	MinAttributeDeadline = 0
	MaxAttributeDeadline = Year
)

var (
	validAttributeRoles = util.SetOf(
		RoleRequired,
		RoleOptional,
		RoleConst,
		RoleMeta,
		RoleFeed,
		RoleOutput,
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
	case RoleMeta:
		return s.validateMeta(name)
	case RoleFeed:
		return s.validateFeed(name)
	case RoleOutput:
		return s.validateOutput(name)
	}
	return nil
}

func (s *AttributeSpec) validateRoleConfig(name Name) error {
	hasConfig := map[AttributeRole]bool{
		RoleRequired: s.Required != nil,
		RoleOptional: s.Optional != nil,
		RoleConst:    s.Const != nil,
		RoleMeta:     s.Meta != nil,
		RoleFeed:     s.Feed != nil,
		RoleOutput:   s.Output != nil,
	}
	for role, set := range hasConfig {
		if role != s.Role && set {
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
	if cfg.Collect != "" && !validInputCollect.Contains(cfg.Collect) {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrInvalidInputCollect, cfg.Collect, name)
	}
	if err := validateForEachCollect(
		cfg.ForEach, cfg.Collect, name,
	); err != nil {
		return err
	}
	return validateMapping(cfg.Mapping, name)
}

func (s *AttributeSpec) validateOptional(name Name) error {
	cfg := s.Optional
	if cfg == nil {
		return nil
	}
	if cfg.Default != "" {
		if err := validateJSON(cfg.Default); err != nil {
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
	if err := validateForEachCollect(
		cfg.ForEach, cfg.Collect, name,
	); err != nil {
		return err
	}
	return validateMapping(cfg.Mapping, name)
}

func (s *AttributeSpec) validateConst(name Name) error {
	if s.Const == nil || s.Const.Value == "" {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrConstValueRequired, s.Role, name)
	}
	return nil
}

func (s *AttributeSpec) validateMeta(name Name) error {
	if s.Meta == nil || s.Meta.Key == "" {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrMetaKeyRequired, s.Role, name)
	}
	return nil
}

func (s *AttributeSpec) validateFeed(name Name) error {
	if s.Feed == nil || s.Feed.ID == "" {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrFeedIDRequired, s.Role, name)
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
	return s.IsInput() || s.IsConst() || s.IsMeta() || s.IsFeed()
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

// IsMeta returns true if the attribute is a metadata input
func (s *AttributeSpec) IsMeta() bool {
	return s.Role == RoleMeta
}

// IsFeed returns true if the attribute reads a Feed
func (s *AttributeSpec) IsFeed() bool {
	return s.Role == RoleFeed
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

// Match returns the match predicate configured for the attribute, if any
func (s *AttributeSpec) Match() *ScriptConfig {
	switch s.Role {
	case RoleRequired:
		if s.Required != nil {
			return s.Required.Match
		}
	case RoleOptional:
		if s.Optional != nil {
			return s.Optional.Match
		}
	}
	return nil
}

func (s *AttributeSpec) ConstValue() string {
	if s.Const == nil {
		return ""
	}
	return s.Const.Value
}

func (s *AttributeSpec) MetaKey() string {
	if s.Meta == nil {
		return ""
	}
	return s.Meta.Key
}

// FeedID returns the ID of the Feed a feed attribute reads
func (s *AttributeSpec) FeedID() FeedID {
	if s.Feed == nil {
		return ""
	}
	return s.Feed.ID
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
		s.Compensated == other.Compensated &&
		s.Required.equal(other.Required) &&
		s.Optional.equal(other.Optional) &&
		s.Const.equal(other.Const) &&
		s.Meta.equal(other.Meta) &&
		s.Feed.equal(other.Feed) &&
		s.Output.equal(other.Output)
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

func (c *RequiredConfig) equal(other *RequiredConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.Collect == other.Collect &&
		c.ForEach == other.ForEach &&
		c.Match.equal(other.Match) &&
		c.Mapping.equal(other.Mapping)
}

func (c *OptionalConfig) equal(other *OptionalConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.Collect == other.Collect &&
		c.ForEach == other.ForEach &&
		c.Default == other.Default &&
		c.Deadline == other.Deadline &&
		c.Match.equal(other.Match) &&
		c.Mapping.equal(other.Mapping)
}

func (c *ConstConfig) equal(other *ConstConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.Value == other.Value
}

func (c *MetaConfig) equal(other *MetaConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.Key == other.Key
}

func (c *FeedConfig) equal(other *FeedConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.ID == other.ID
}

func (c *OutputConfig) equal(other *OutputConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.Mapping.equal(other.Mapping)
}

func (c *MappingConfig) equal(other *MappingConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.Name == other.Name && c.Script.equal(other.Script)
}

func (c *ScriptConfig) equal(other *ScriptConfig) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.Language == other.Language && c.Source == other.Source
}

func validateForEachCollect(
	forEach bool, collect InputCollect, name Name,
) error {
	if !forEach {
		return nil
	}
	if collect != InputCollectAll && collect != InputCollectSome {
		return fmt.Errorf("%w: %s for attribute %q",
			ErrForEachRequiresCollect, collect, name)
	}
	return nil
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

func validateJSON(data string) error {
	if !json.Valid([]byte(data)) {
		return ErrDefaultJSON
	}
	return nil
}
