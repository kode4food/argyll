package api

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// StepType defines the execution mode for a step (sync, async, or script)
	StepType string

	// Metadata contains additional context passed to step handlers
	Metadata map[string]any

	// Steps contains a map of Steps by their ID
	Steps map[StepID]*Step

	// Labels contains optional step metadata used for discovery and grouping
	Labels map[string]string

	// Step defines a flow step with its configuration, attributes, and
	// execution details
	Step struct {
		Predicate  *ScriptConfig  `json:"predicate,omitempty"`
		HTTP       *HTTPConfig    `json:"http,omitempty"`
		Script     *ScriptConfig  `json:"script,omitempty"`
		WorkConfig *WorkConfig    `json:"work_config,omitempty"`
		Labels     Labels         `json:"labels,omitempty"`
		ID         StepID         `json:"id"`
		Name       Name           `json:"name"`
		Type       StepType       `json:"type"`
		Attributes AttributeSpecs `json:"attributes"`
	}

	// HTTPConfig configures HTTP-based step execution
	HTTPConfig struct {
		Endpoint    string `json:"endpoint"`
		HealthCheck string `json:"health_check,omitempty"`
		Timeout     int64  `json:"timeout"`
	}

	// ScriptConfig configures script-based step execution
	ScriptConfig struct {
		Language string `json:"language"`
		Script   string `json:"script"`
	}

	// WorkConfig configures retry and parallelism behavior for steps with
	// multiple work items
	WorkConfig struct {
		MaxRetries   int    `json:"max_retries,omitempty"`
		BackoffMs    int64  `json:"backoff_ms,omitempty"`
		MaxBackoffMs int64  `json:"max_backoff_ms,omitempty"`
		BackoffType  string `json:"backoff_type,omitempty"`
		Parallelism  int    `json:"parallelism,omitempty"`
	}

	// StepRequest is the request payload sent to step handlers
	StepRequest struct {
		Arguments Args     `json:"arguments"`
		Metadata  Metadata `json:"metadata"`
	}

	// StepResult is the response returned by step handlers
	StepResult struct {
		Outputs Args   `json:"outputs,omitempty"`
		Error   string `json:"error,omitempty"`
		Success bool   `json:"success"`
	}
)

const (
	StepTypeSync   StepType = "sync"
	StepTypeAsync  StepType = "async"
	StepTypeScript StepType = "script"

	ScriptLangAle = "ale"
	ScriptLangLua = "lua"

	BackoffTypeFixed       = "fixed"
	BackoffTypeLinear      = "linear"
	BackoffTypeExponential = "exponential"
)

const (
	Second int64 = 1000
	Minute       = Second * 60
	Hour         = Minute * 60
	Day          = Hour * 24
)

var (
	ErrStepIDEmpty           = errors.New("step ID empty")
	ErrStepNameEmpty         = errors.New("step name empty")
	ErrStepEndpointEmpty     = errors.New("step endpoint empty")
	ErrArgNameEmpty          = errors.New("argument name empty")
	ErrInvalidStepType       = errors.New("invalid step type")
	ErrHTTPRequired          = errors.New("http required")
	ErrScriptRequired        = errors.New("script required")
	ErrScriptLanguageEmpty   = errors.New("script language empty")
	ErrInvalidScriptLanguage = errors.New("invalid script language")
	ErrScriptEmpty           = errors.New("script empty")
	ErrInvalidRetryConfig    = errors.New("invalid retry config")
	ErrInvalidBackoffType    = errors.New("invalid backoff type")
	ErrAttributeNil          = errors.New("attribute has nil definition")
	ErrNegativeBackoff       = errors.New("backoff_ms cannot be negative")
	ErrMaxBackoffTooSmall    = errors.New("max_backoff_ms must be >= backoff_ms")
	ErrWorkNotCompleted      = errors.New("work not completed")
)

var (
	validStepTypes = util.SetOf(
		StepTypeSync,
		StepTypeAsync,
		StepTypeScript,
	)

	validBackoffTypes = util.SetOf(
		BackoffTypeFixed,
		BackoffTypeLinear,
		BackoffTypeExponential,
	)

	validScriptLanguages = util.SetOf(
		ScriptLangAle,
		ScriptLangLua,
	)
)

// NewResult creates a new successful step result with empty outputs
func NewResult() *StepResult {
	return &StepResult{
		Success: true,
	}
}

// Validate checks if the step configuration is valid
func (s *Step) Validate() error {
	if s.ID == "" {
		return ErrStepIDEmpty
	}
	if s.Name == "" {
		return ErrStepNameEmpty
	}

	if !validStepTypes.Contains(s.Type) {
		return fmt.Errorf("%w: %s", ErrInvalidStepType, s.Type)
	}

	switch s.Type {
	case StepTypeSync, StepTypeAsync:
		if err := s.validateHTTPConfig(); err != nil {
			return err
		}
	case StepTypeScript:
		if err := s.validateScriptConfig(); err != nil {
			return err
		}
	}

	if err := s.validateAttributes(); err != nil {
		return err
	}
	return s.validateWorkConfig()
}

func (s *Step) validateAttributes() error {
	if s.Attributes == nil {
		s.Attributes = AttributeSpecs{}
	}

	for name, attr := range s.Attributes {
		if name == "" {
			return ErrArgNameEmpty
		}
		if attr == nil {
			return ErrAttributeNil
		}
		if err := attr.Validate(name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Step) validateHTTPConfig() error {
	if s.HTTP == nil {
		return ErrHTTPRequired
	}
	if s.HTTP.Endpoint == "" {
		return ErrStepEndpointEmpty
	}
	return nil
}

func (s *Step) validateScriptConfig() error {
	if s.Script == nil {
		return ErrScriptRequired
	}
	if s.Script.Language == "" {
		return ErrScriptLanguageEmpty
	}
	if !validScriptLanguages.Contains(s.Script.Language) {
		return fmt.Errorf("%w: %s", ErrInvalidScriptLanguage, s.Script.Language)
	}
	if s.Script.Script == "" {
		return ErrScriptEmpty
	}
	return nil
}

func (s *Step) validateWorkConfig() error {
	if s.WorkConfig == nil {
		return nil
	}

	if s.WorkConfig.BackoffMs < 0 {
		return ErrNegativeBackoff
	}

	if s.WorkConfig.MaxBackoffMs != 0 &&
		s.WorkConfig.MaxBackoffMs < s.WorkConfig.BackoffMs {
		return ErrMaxBackoffTooSmall
	}

	hasRetryConfig := s.WorkConfig.MaxRetries != 0 ||
		s.WorkConfig.BackoffMs != 0 || s.WorkConfig.MaxBackoffMs != 0
	if hasRetryConfig {
		if s.WorkConfig.BackoffType == "" {
			return ErrInvalidRetryConfig
		}
		if !validBackoffTypes.Contains(s.WorkConfig.BackoffType) {
			return ErrInvalidBackoffType
		}
	}

	return nil
}

// IsOptionalArg returns true if the argument is optional
func (s *Step) IsOptionalArg(argName Name) bool {
	if attr, ok := s.Attributes[argName]; ok {
		return attr.IsOptional()
	}
	return false
}

// SortedArgNames returns sorted input argument names
func (s *Step) SortedArgNames() []string {
	var all []string
	for name, attr := range s.Attributes {
		if attr.IsInput() {
			all = append(all, string(name))
		}
	}
	slices.Sort(all)
	return all
}

// MultiArgNames returns names of attributes that support multiple work items
// (for_each)
func (s *Step) MultiArgNames() []Name {
	var names []Name
	for name, attr := range s.Attributes {
		if attr.ForEach {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// GetAllInputArgs returns all input argument names (required and optional)
func (s *Step) GetAllInputArgs() []Name {
	return s.filterAttributes((*AttributeSpec).IsInput)
}

// GetRequiredArgs returns all required argument names
func (s *Step) GetRequiredArgs() []Name {
	return s.filterAttributes((*AttributeSpec).IsRequired)
}

// GetOptionalArgs returns all optional argument names
func (s *Step) GetOptionalArgs() []Name {
	return s.filterAttributes((*AttributeSpec).IsOptional)
}

// GetOutputArgs returns all output argument names
func (s *Step) GetOutputArgs() []Name {
	return s.filterAttributes((*AttributeSpec).IsOutput)
}

// Equal returns true if two steps are equal
func (s *Step) Equal(other *Step) bool {
	if s.ID != other.ID || s.Name != other.Name || s.Type != other.Type {
		return false
	}
	if !s.Attributes.Equal(other.Attributes) {
		return false
	}
	if !s.HTTP.Equal(other.HTTP) {
		return false
	}
	if !s.Script.Equal(other.Script) {
		return false
	}
	if !s.Predicate.Equal(other.Predicate) {
		return false
	}
	if !s.WorkConfig.Equal(other.WorkConfig) {
		return false
	}
	if !s.Labels.Equal(other.Labels) {
		return false
	}
	return true
}

func (s *Step) filterAttributes(predicate func(*AttributeSpec) bool) []Name {
	var args []Name
	for name, attr := range s.Attributes {
		if predicate(attr) {
			args = append(args, name)
		}
	}
	return args
}

// WithOutput adds an output value to the step result
func (r *StepResult) WithOutput(name Name, value any) *StepResult {
	if r.Outputs == nil {
		r.Outputs = Args{name: value}
		return r
	}
	r.Outputs[name] = value
	return r
}

// WithError marks the step result as failed with the given error
func (r *StepResult) WithError(err error) *StepResult {
	r.Success = false
	r.Error = err.Error()
	return r
}

// Equal returns true if two HTTP configs are equal
func (h *HTTPConfig) Equal(other *HTTPConfig) bool {
	return equalWithNilCheck(h, other, func() bool {
		return h.Endpoint == other.Endpoint &&
			h.HealthCheck == other.HealthCheck &&
			h.Timeout == other.Timeout
	})
}

// Equal returns true if two script configs are equal
func (c *ScriptConfig) Equal(other *ScriptConfig) bool {
	return equalWithNilCheck(c, other, func() bool {
		return c.Language == other.Language && c.Script == other.Script
	})
}

// Equal returns true if two work configs are equal
func (c *WorkConfig) Equal(other *WorkConfig) bool {
	return equalWithNilCheck(c, other, func() bool {
		return c.Parallelism == other.Parallelism &&
			c.MaxRetries == other.MaxRetries &&
			c.BackoffMs == other.BackoffMs &&
			c.MaxBackoffMs == other.MaxBackoffMs &&
			c.BackoffType == other.BackoffType
	})
}

func equalWithNilCheck[T any](a, b *T, compare func() bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return compare()
}

// Equal returns true if two label sets are equal
func (l Labels) Equal(other Labels) bool {
	if len(l) != len(other) {
		return false
	}
	for key, val := range l {
		otherVal, ok := other[key]
		if !ok || otherVal != val {
			return false
		}
	}
	return true
}
