package api

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/util"
)

type (
	StepType string
	Metadata map[string]any

	Step struct {
		Predicate  *ScriptConfig  `json:"predicate,omitempty"`
		HTTP       *HTTPConfig    `json:"http,omitempty"`
		Script     *ScriptConfig  `json:"script,omitempty"`
		WorkConfig *WorkConfig    `json:"work_config,omitempty"`
		ID         timebox.ID     `json:"id"`
		Name       Name           `json:"name"`
		Type       StepType       `json:"type"`
		Version    string         `json:"version"`
		Attributes AttributeSpecs `json:"attributes"`
	}

	HTTPConfig struct {
		Endpoint    string `json:"endpoint"`
		HealthCheck string `json:"health_check,omitempty"`
		Timeout     int64  `json:"timeout"`
	}

	ScriptConfig struct {
		Language string `json:"language"`
		Script   string `json:"script"`
	}

	WorkConfig struct {
		MaxRetries   int    `json:"max_retries,omitempty"`
		BackoffMs    int64  `json:"backoff_ms,omitempty"`
		MaxBackoffMs int64  `json:"max_backoff_ms,omitempty"`
		BackoffType  string `json:"backoff_type,omitempty"`
		Parallelism  int    `json:"parallelism,omitempty"`
	}

	StepRequest struct {
		Arguments Args     `json:"arguments"`
		Metadata  Metadata `json:"metadata"`
	}

	StepResult struct {
		Outputs Args   `json:"outputs,omitempty"`
		Error   string `json:"error,omitempty"`
		Success bool   `json:"success"`
	}

	StepHandler func(context.Context, Args) (StepResult, error)
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
	ErrStepIDEmpty         = errors.New("step ID empty")
	ErrStepNameEmpty       = errors.New("step name empty")
	ErrStepEndpointEmpty   = errors.New("step endpoint empty")
	ErrStepVersionEmpty    = errors.New("step version empty")
	ErrArgNameEmpty        = errors.New("argument name empty")
	ErrInvalidStepType     = errors.New("invalid step type")
	ErrHTTPRequired        = errors.New("http required")
	ErrScriptRequired      = errors.New("script required")
	ErrScriptLanguageEmpty = errors.New("script language empty")
	ErrScriptEmpty         = errors.New("script empty")
	ErrInvalidRetryConfig  = errors.New("invalid retry config")
	ErrInvalidBackoffType  = errors.New("invalid backoff type")
	ErrAttributeNil        = errors.New("attribute has nil definition")
	ErrNegativeBackoff     = errors.New("backoff_ms cannot be negative")
	ErrMaxBackoffTooSmall  = errors.New("max_backoff_ms must be >= backoff_ms")
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
)

func NewResult() *StepResult {
	return &StepResult{
		Success: true,
		Outputs: Args{},
	}
}

func (s *Step) Validate() error {
	if s.ID == "" {
		return ErrStepIDEmpty
	}
	if s.Name == "" {
		return ErrStepNameEmpty
	}
	if s.Version == "" {
		return ErrStepVersionEmpty
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

	if s.WorkConfig.MaxBackoffMs != 0 && s.WorkConfig.MaxBackoffMs < s.WorkConfig.BackoffMs {
		return ErrMaxBackoffTooSmall
	}

	hasRetryConfig := s.WorkConfig.MaxRetries != 0 || s.WorkConfig.BackoffMs != 0 || s.WorkConfig.MaxBackoffMs != 0
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

func (s *Step) GetAllInputArgs() []Name {
	var args []Name
	for name, attr := range s.Attributes {
		if attr.IsInput() {
			args = append(args, name)
		}
	}
	return args
}

func (s *Step) GetRequiredArgs() []Name {
	var args []Name
	for name, attr := range s.Attributes {
		if attr.IsRequired() {
			args = append(args, name)
		}
	}
	return args
}

func (s *Step) GetOptionalArgs() []Name {
	var args []Name
	for name, attr := range s.Attributes {
		if attr.IsOptional() {
			args = append(args, name)
		}
	}
	return args
}

func (s *Step) GetOutputArgs() []Name {
	var args []Name
	for name, attr := range s.Attributes {
		if attr.IsOutput() {
			args = append(args, name)
		}
	}
	return args
}

func (s *Step) IsOptionalArg(argName Name) bool {
	if attr, ok := s.Attributes[argName]; ok {
		return attr.IsOptional()
	}
	return false
}

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

func (sr *StepResult) WithOutput(name Name, value any) *StepResult {
	sr.Outputs[name] = value
	return sr
}

func (sr *StepResult) WithError(err error) *StepResult {
	sr.Success = false
	sr.Error = err.Error()
	return sr
}

func (s *Step) Equal(other *Step) bool {
	if s.ID != other.ID || s.Name != other.Name || s.Type != other.Type {
		return false
	}
	if s.Version != other.Version {
		return false
	}
	if !attributeMapsEqual(s.Attributes, other.Attributes) {
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
	return true
}

func (h *HTTPConfig) Equal(other *HTTPConfig) bool {
	if h == nil && other == nil {
		return true
	}
	if h == nil || other == nil {
		return false
	}
	return h.Endpoint == other.Endpoint &&
		h.HealthCheck == other.HealthCheck &&
		h.Timeout == other.Timeout
}

func (sc *ScriptConfig) Equal(other *ScriptConfig) bool {
	if sc == nil && other == nil {
		return true
	}
	if sc == nil || other == nil {
		return false
	}
	return sc.Language == other.Language && sc.Script == other.Script
}

func (wc *WorkConfig) Equal(other *WorkConfig) bool {
	if wc == nil && other == nil {
		return true
	}
	if wc == nil || other == nil {
		return false
	}
	return wc.Parallelism == other.Parallelism &&
		wc.MaxRetries == other.MaxRetries &&
		wc.BackoffMs == other.BackoffMs &&
		wc.MaxBackoffMs == other.MaxBackoffMs &&
		wc.BackoffType == other.BackoffType
}

func attributeMapsEqual(a, b AttributeSpecs) bool {
	if len(a) != len(b) {
		return false
	}
	for name, attrA := range a {
		attrB, ok := b[name]
		if !ok || !attrA.Equal(attrB) {
			return false
		}
	}
	return true
}
