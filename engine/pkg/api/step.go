package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync/atomic"

	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// StepType defines the execution mode for a step (sync, async, or script)
	StepType string

	// Steps contains a map of Steps by their ID
	Steps map[StepID]*Step

	// Labels contains optional step metadata used for discovery and grouping
	Labels map[string]string

	// Step defines a flow step with its configuration, attributes, and
	// execution details
	Step struct {
		Predicate  *ScriptConfig  `json:"predicate,omitempty"`
		HTTP       *HTTPConfig    `json:"http,omitempty"`
		Flow       *FlowConfig    `json:"flow,omitempty"`
		Script     *ScriptConfig  `json:"script,omitempty"`
		WorkConfig *WorkConfig    `json:"work_config,omitempty"`
		Labels     Labels         `json:"labels,omitempty"`
		ID         StepID         `json:"id"`
		Name       Name           `json:"name"`
		Type       StepType       `json:"type"`
		Memoizable bool           `json:"memoizable,omitempty"`
		Attributes AttributeSpecs `json:"attributes"`
		hashKey    atomic.Pointer[string]
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

	// FlowConfig configures flow-based step execution
	FlowConfig struct {
		Goals []StepID `json:"goals"`
	}

	// WorkConfig configures retry and parallelism behavior for steps with
	// multiple work items
	WorkConfig struct {
		MaxRetries  int    `json:"max_retries,omitempty"`
		Backoff     int64  `json:"backoff,omitempty"`
		MaxBackoff  int64  `json:"max_backoff,omitempty"`
		BackoffType string `json:"backoff_type,omitempty"`
		Parallelism int    `json:"parallelism,omitempty"`
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

	attrPair struct {
		K Name           `json:"k"`
		V *AttributeSpec `json:"v"`
	}

	flowCfg struct {
		Goals []StepID `json:"goals"`
	}

	stepHash struct {
		Type       StepType      `json:"type"`
		Memoizable bool          `json:"memoizable,omitempty"`
		Attributes []attrPair    `json:"attributes"`
		HTTP       *HTTPConfig   `json:"http,omitempty"`
		Script     *ScriptConfig `json:"script,omitempty"`
		Flow       any           `json:"flow,omitempty"`
		Predicate  *ScriptConfig `json:"predicate,omitempty"`
		WorkConfig *WorkConfig   `json:"work_config,omitempty"`
	}
)

const (
	StepTypeSync   StepType = "sync"
	StepTypeAsync  StepType = "async"
	StepTypeScript StepType = "script"
	StepTypeFlow   StepType = "flow"

	ScriptLangAle   = "ale"
	ScriptLangJPath = "jpath"
	ScriptLangLua   = "lua"

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
	ErrFlowRequired          = errors.New("flow required")
	ErrFlowGoalsRequired     = errors.New("flow goals required")
	ErrHTTPNotAllowed        = errors.New("http not allowed for step type")
	ErrScriptNotAllowed      = errors.New("script not allowed for step type")
	ErrFlowNotAllowed        = errors.New("flow not allowed for step type")
	ErrScriptLanguageEmpty   = errors.New("script language empty")
	ErrInvalidScriptLanguage = errors.New("invalid script language")
	ErrScriptEmpty           = errors.New("script empty")
	ErrInvalidRetryConfig    = errors.New("invalid retry config")
	ErrInvalidBackoffType    = errors.New("invalid backoff type")
	ErrAttributeNil          = errors.New("attribute has nil definition")
	ErrNegativeBackoff       = errors.New("backoff cannot be negative")
	ErrMaxBackoffTooSmall    = errors.New("max_backoff must be >= backoff")
	ErrWorkNotCompleted      = errors.New("work not completed")
)

var (
	validStepTypes = util.SetOf(
		StepTypeSync,
		StepTypeAsync,
		StepTypeScript,
		StepTypeFlow,
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
	case StepTypeFlow:
		if err := s.validateFlowConfig(); err != nil {
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
	if err := s.validateMappingNames(); err != nil {
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
	if s.Flow != nil {
		return ErrFlowNotAllowed
	}
	if s.Script != nil {
		return ErrScriptNotAllowed
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
	if s.HTTP != nil {
		return ErrHTTPNotAllowed
	}
	if s.Flow != nil {
		return ErrFlowNotAllowed
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

func (s *Step) validateFlowConfig() error {
	if s.Flow == nil {
		return ErrFlowRequired
	}
	if s.HTTP != nil {
		return ErrHTTPNotAllowed
	}
	if s.Script != nil {
		return ErrScriptNotAllowed
	}
	if len(s.Flow.Goals) == 0 {
		return ErrFlowGoalsRequired
	}
	return nil
}

func (s *Step) validateMappingNames() error {
	inputInnerNames := map[string]Name{}
	outputInnerNames := map[string]Name{}

	for name, attr := range s.Attributes {
		if attr.Mapping == nil || attr.Mapping.Name == "" {
			continue
		}

		innerName := attr.Mapping.Name

		if attr.IsRuntimeInput() {
			if _, ok := inputInnerNames[innerName]; ok {
				return fmt.Errorf("%w: %q", ErrDuplicateInnerName, innerName)
			}
			inputInnerNames[innerName] = name
		}

		if attr.IsOutput() {
			if _, ok := outputInnerNames[innerName]; ok {
				return fmt.Errorf("%w: %q", ErrDuplicateInnerName, innerName)
			}
			outputInnerNames[innerName] = name
		}
	}

	return nil
}

func (s *Step) validateWorkConfig() error {
	if s.WorkConfig == nil {
		return nil
	}

	if s.WorkConfig.Backoff < 0 {
		return ErrNegativeBackoff
	}

	if s.WorkConfig.MaxBackoff != 0 &&
		s.WorkConfig.MaxBackoff < s.WorkConfig.Backoff {
		return ErrMaxBackoffTooSmall
	}

	hasRetryConfig := s.WorkConfig.MaxRetries != 0 ||
		s.WorkConfig.Backoff != 0 || s.WorkConfig.MaxBackoff != 0
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

// SortedArgNames returns sorted runtime input argument names
func (s *Step) SortedArgNames() []string {
	var all []string
	for name, attr := range s.Attributes {
		if attr.IsRuntimeInput() {
			if attr.Mapping != nil && attr.Mapping.Name != "" {
				all = append(all, attr.Mapping.Name)
				continue
			}
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
	if s.Memoizable != other.Memoizable {
		return false
	}
	if !s.Attributes.Equal(other.Attributes) {
		return false
	}
	if !s.HTTP.Equal(other.HTTP) {
		return false
	}
	if !s.Flow.Equal(other.Flow) {
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

// HashKey computes a deterministic SHA256 hash key of the functional parts of
// the step definition. Excludes ID, Name, and Labels (non-functional metadata)
func (s *Step) HashKey() (string, error) {
	if cached := s.hashKey.Load(); cached != nil {
		return *cached, nil
	}

	key, err := s.computeHashKey()
	if err != nil {
		return "", err
	}

	s.hashKey.Store(&key)
	return key, nil
}

func (s *Step) computeHashKey() (string, error) {
	names := make([]Name, 0, len(s.Attributes))
	for n := range s.Attributes {
		names = append(names, n)
	}
	slices.Sort(names)

	attrs := make([]attrPair, len(names))
	for i, n := range names {
		attrs[i] = attrPair{K: n, V: s.Attributes[n]}
	}

	var flow any
	if s.Flow != nil {
		flow = flowCfg{
			Goals: s.Flow.Goals,
		}
	}

	h := stepHash{
		Type:       s.Type,
		Memoizable: s.Memoizable,
		Attributes: attrs,
		HTTP:       s.HTTP,
		Script:     s.Script,
		Flow:       flow,
		Predicate:  s.Predicate,
		WorkConfig: s.WorkConfig,
	}

	data, err := json.Marshal(h)
	if err != nil {
		return "", fmt.Errorf("failed to marshal step definition: %w", err)
	}

	return sha256Hex(string(data)), nil
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

// Equal returns true if two flow configs are equal
func (c *FlowConfig) Equal(other *FlowConfig) bool {
	return equalWithNilCheck(c, other, func() bool {
		return slices.Equal(c.Goals, other.Goals)
	})
}

// Equal returns true if two work configs are equal
func (c *WorkConfig) Equal(other *WorkConfig) bool {
	return equalWithNilCheck(c, other, func() bool {
		return c.Parallelism == other.Parallelism &&
			c.MaxRetries == other.MaxRetries &&
			c.Backoff == other.Backoff &&
			c.MaxBackoff == other.MaxBackoff &&
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

// Apply will merge the keys/values of the other label set into this one
func (l Labels) Apply(other Labels) Labels {
	if len(other) == 0 {
		return l
	}
	if l == nil {
		return other
	}
	res := maps.Clone(l)
	maps.Copy(res, other)
	return res
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
