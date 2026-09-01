package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sync"

	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

type (
	// Step defines a flow step with its configuration, attributes, and
	// execution details
	Step struct {
		Script      *ScriptConfig  `json:"script,omitempty"`
		Tags        Tags           `json:"tags,omitempty"`
		Attributes  AttributeSpecs `json:"attributes"`
		Predicate   *ScriptConfig  `json:"predicate,omitempty"`
		WorkConfig  *WorkConfig    `json:"work_config,omitempty"`
		HTTP        *HTTPConfig    `json:"http,omitempty"`
		Flow        *FlowConfig    `json:"flow,omitempty"`
		Type        StepType       `json:"type"`
		ID          StepID         `json:"id"`
		Name        Name           `json:"name"`
		Description string         `json:"description,omitempty"`
		Handling    Handling       `json:"handling,omitempty"`

		hashErr  error
		hashVal  string
		hashOnce sync.Once
	}

	// HTTPConfig configures HTTP-based step execution
	HTTPConfig struct {
		Invoke     HTTPAction  `json:"invoke"`
		Compensate *HTTPAction `json:"compensate,omitempty"`
		Health     string      `json:"health,omitempty"`
	}

	// HTTPAction configures a single callable endpoint on a step's service
	HTTPAction struct {
		Method   string     `json:"method,omitempty"`
		Endpoint string     `json:"endpoint"`
		Timeout  int64      `json:"timeout,omitempty"`
		Mode     ActionMode `json:"mode,omitempty"`
	}

	// ScriptConfig configures script-based step execution
	ScriptConfig struct {
		Language string `json:"language"`
		Script   string `json:"script"`
	}

	// FlowConfig configures flow-based step execution
	FlowConfig struct {
		Goals      []StepID `json:"goals"`
		SpaceID    SpaceID  `json:"space_id,omitempty"`
		Compensate bool     `json:"compensate,omitempty"`
	}

	// WorkConfig configures retry and parallelism behavior for steps with
	// multiple work items
	WorkConfig struct {
		BackoffType string `json:"backoff_type,omitempty"`
		MaxRetries  int    `json:"max_retries,omitempty"`
		InitBackoff int64  `json:"init_backoff,omitempty"`
		MaxBackoff  int64  `json:"max_backoff,omitempty"`
		Parallelism int    `json:"parallelism,omitempty"`
	}

	// StepType defines the execution mode for a step (sync, async, or script)
	StepType string

	// Handling defines how a step's completed work is retained or reversed
	Handling string

	// ActionMode defines how an HTTP action reports its result
	ActionMode string

	// Steps contains a map of Steps by their ID
	Steps map[StepID]*Step

	// Tags contains optional step metadata used for discovery and grouping,
	// held as a set of opaque strings
	Tags []string

	attrPair struct {
		V *AttributeSpec `json:"v"`
		K Name           `json:"k"`
	}

	stepHash struct {
		Flow       *FlowConfig   `json:"flow,omitempty"`
		HTTP       *HTTPConfig   `json:"http,omitempty"`
		Script     *ScriptConfig `json:"script,omitempty"`
		Predicate  *ScriptConfig `json:"predicate,omitempty"`
		WorkConfig *WorkConfig   `json:"work_config,omitempty"`
		Type       StepType      `json:"type"`
		Attributes []attrPair    `json:"attributes"`
		Handling   Handling      `json:"handling"`
	}
)

const (
	StepTypeService StepType = "service"
	StepTypeScript  StepType = "script"
	StepTypeFlow    StepType = "flow"

	ScriptLangJPath = "jpath"
	ScriptLangLua   = "lua"

	BackoffTypeFixed       = "fixed"
	BackoffTypeLinear      = "linear"
	BackoffTypeExponential = "exponential"

	HandlingStandard    Handling = "standard"
	HandlingMemoized    Handling = "memoized"
	HandlingCompensated Handling = "compensated"

	// ActionModeSync returns the result in the invocation response
	ActionModeSync ActionMode = "sync"

	// ActionModeAsync returns the result through a webhook callback
	ActionModeAsync ActionMode = "async"

	// DefaultActionMode is used when an HTTP action omits its mode
	DefaultActionMode = ActionModeSync

	// DefaultHTTPMethod is used when an HTTP action omits its method
	DefaultHTTPMethod = "POST"
)

const (
	Second int64 = 1000
	Minute       = Second * 60
	Hour         = Minute * 60
	Day          = Hour * 24
)

var (
	ErrStepIDEmpty           = errors.New("step ID empty")
	ErrStepIDInvalid         = errors.New("step ID contains invalid characters")
	ErrStepNameEmpty         = errors.New("step name empty")
	ErrTagEmpty              = errors.New("tag empty")
	ErrStepEndpointEmpty     = errors.New("step endpoint empty")
	ErrInvalidHTTPMethod     = errors.New("invalid HTTP method")
	ErrInvalidActionMode     = errors.New("invalid action mode")
	ErrUnknownURLParam       = errors.New("URL contains unknown parameter")
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
	ErrInvalidBackoffType    = errors.New("invalid backoff type")
	ErrInvalidParallelism    = errors.New("parallelism cannot be negative")
	ErrAttributeNil          = errors.New("attribute has nil definition")
	ErrNegativeBackoff       = errors.New("backoff cannot be negative")
	ErrMaxBackoffTooSmall    = errors.New("max_backoff must be >= backoff")
	ErrWorkNotCompleted      = errors.New("work not completed")
	ErrMarshalStep           = errors.New("failed to marshal step definition")
	ErrInvalidHandling       = errors.New("invalid step handling")
	ErrCompensateRequired    = errors.New(
		"compensated handling requires a compensation endpoint",
	)
	ErrCompensateHandling = errors.New(
		"compensation endpoint requires compensated handling",
	)
	ErrAttributeCompensated = errors.New(
		"compensated attribute requires compensated handling",
	)
	ErrCompensateArgConflict = errors.New(
		"conflicting compensation argument",
	)
)

var (
	validBackoffTypes = util.SetOf(
		BackoffTypeFixed,
		BackoffTypeLinear,
		BackoffTypeExponential,
	)

	validScriptLanguages = util.SetOf(
		ScriptLangLua,
	)

	validHTTPMethods = util.SetOf(
		"GET",
		"POST",
		"PUT",
		"DELETE",
	)

	validHandling = util.SetOf(
		HandlingStandard,
		HandlingMemoized,
		HandlingCompensated,
	)

	validActionModes = util.SetOf(
		ActionModeSync,
		ActionModeAsync,
	)

	endpointParamPattern = regexp.MustCompile(`\{([^{}]+)\}`)
)

// Validate checks if the step configuration is valid
func (s *Step) Validate() error {
	return call.Perform(
		s.validateIdentity,
		s.validateHandling,
		s.validateTypeConfig,
		s.validateAttributes,
		s.validateMappingNames,
		s.validateWorkConfig,
	)
}

func (s *Step) validateIdentity() error {
	if s.ID == "" {
		return ErrStepIDEmpty
	}
	if SanitizeID(s.ID) != s.ID {
		return ErrStepIDInvalid
	}
	if s.Name == "" {
		return ErrStepNameEmpty
	}
	if len(s.Tags) > MaxTagCount {
		return fmt.Errorf("%w: maximum is %d", ErrTooManyTags, MaxTagCount)
	}
	if slices.Contains(s.Tags, "") {
		return ErrTagEmpty
	}
	return nil
}

func (s *Step) validateTypeConfig() error {
	switch s.Type {
	case StepTypeService:
		return s.validateHTTPConfig()
	case StepTypeFlow:
		return s.validateFlowConfig()
	case StepTypeScript:
		return s.validateScriptConfig()
	}
	return nil
}

// Copy returns a shallow copy of the step without copying internal cache state
func (s *Step) Copy() *Step {
	return &Step{
		Predicate:   s.Predicate,
		HTTP:        s.HTTP,
		Flow:        s.Flow,
		Script:      s.Script,
		WorkConfig:  s.WorkConfig,
		Tags:        s.Tags,
		ID:          s.ID,
		Name:        s.Name,
		Description: s.Description,
		Type:        s.Type,
		Handling:    s.Handling,
		Attributes:  s.Attributes,
	}
}

// WithWorkDefaults returns a copy of the step with zero-valued work fields
// filled in from defaults
func (s *Step) WithWorkDefaults(defaults *WorkConfig) *Step {
	res := s.Copy()
	work := util.MutableCopy(res.WorkConfig)
	if work.MaxRetries == 0 {
		work.MaxRetries = defaults.MaxRetries
	}
	if work.InitBackoff == 0 {
		work.InitBackoff = defaults.InitBackoff
	}
	if work.MaxBackoff == 0 {
		work.MaxBackoff = defaults.MaxBackoff
	}
	if work.BackoffType == "" {
		work.BackoffType = defaults.BackoffType
	}
	res.WorkConfig = work
	return res
}

// CanCompensate returns true if the step has compensation configured
func (s *Step) CanCompensate() bool {
	return s.DefaultedHandling() == HandlingCompensated &&
		s.HTTP != nil && s.HTTP.Compensate != nil &&
		s.HTTP.Compensate.Endpoint != ""
}

// DefaultedHandling returns the configured handling or standard when unset
func (s *Step) DefaultedHandling() Handling {
	if s.Handling == "" {
		return HandlingStandard
	}
	return s.Handling
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
			mapped, _ := s.MappedName(name)
			all = append(all, string(mapped))
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
		if attr.ForEach() {
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

// MappedName returns the mapped name when present, else the declared name
func (s *Step) MappedName(name Name) (Name, bool) {
	attr := s.Attributes[name]
	if m := attr.Mapping(); m != nil && m.Name != "" {
		return Name(m.Name), true
	}
	return name, false
}

// Equal returns true if two steps are equal
func (s *Step) Equal(other *Step) bool {
	sameIdentity := s.ID == other.ID && s.Name == other.Name &&
		s.Type == other.Type && s.Description == other.Description
	sameAction := s.HTTP.Equal(other.HTTP) && s.Flow.Equal(other.Flow) &&
		s.Script.Equal(other.Script)
	sameWork := s.DefaultedHandling() == other.DefaultedHandling() &&
		s.WorkConfig.Equal(other.WorkConfig) &&
		s.Predicate.Equal(other.Predicate)
	sameContent := s.Attributes.Equal(other.Attributes) &&
		s.Tags.Equal(other.Tags)
	return sameIdentity && sameAction && sameWork && sameContent
}

// HashKey computes a deterministic SHA256 hash key of the functional parts of
// the step definition. Excludes ID, Name, and Tags (non-functional metadata)
func (s *Step) HashKey() (string, error) {
	s.hashOnce.Do(func() {
		s.hashVal, s.hashErr = s.computeHashKey()
	})
	return s.hashVal, s.hashErr
}

func (s *Step) validateAttributes() error {
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

func (s *Step) validateHandling() error {
	handling := s.DefaultedHandling()
	if !validHandling.Contains(handling) {
		return fmt.Errorf("%w: %s", ErrInvalidHandling, handling)
	}

	var compensate *HTTPAction
	if s.HTTP != nil {
		compensate = s.HTTP.Compensate
	}
	if handling == HandlingCompensated &&
		(compensate == nil || compensate.Endpoint == "") {
		return ErrCompensateRequired
	}
	if handling != HandlingCompensated && compensate != nil {
		return ErrCompensateHandling
	}
	for name, attr := range s.Attributes {
		if attr != nil && attr.Compensated && handling != HandlingCompensated {
			return fmt.Errorf("%w: %s", ErrAttributeCompensated, name)
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
	if err := validateAction(&s.HTTP.Invoke); err != nil {
		return err
	}
	if s.HTTP.Compensate == nil {
		return s.validateEndpointParams()
	}
	return call.Perform(
		call.WithArg(validateAction, s.HTTP.Compensate),
		s.validateEndpointParams,
		s.validateCompensateParams,
	)
}

func (s *Step) validateEndpointParams() error {
	params := endpointParams(s.HTTP.Invoke.Endpoint)
	if params.IsEmpty() {
		return nil
	}

	required := util.Set[string]{}
	for name, attr := range s.Attributes {
		if !attr.IsRequired() {
			continue
		}
		mapped, _ := s.MappedName(name)
		required.Add(string(mapped))
	}

	for param := range params {
		if required.Contains(param) {
			continue
		}
		return fmt.Errorf("%w: %q", ErrUnknownURLParam, param)
	}
	return nil
}

func (s *Step) validateCompensateParams() error {
	known, err := s.resolveCompensationNames()
	if err != nil {
		return err
	}

	params := endpointParams(s.HTTP.Compensate.Endpoint)
	for param := range params {
		if known.Contains(param) {
			continue
		}
		return fmt.Errorf("%w: %q", ErrUnknownURLParam, param)
	}
	return nil
}

func (s *Step) resolveCompensationNames() (util.Set[string], error) {
	known := util.Set[string]{}
	for name, attr := range s.Attributes {
		if attr == nil || !attr.Compensated {
			continue
		}
		mapped, _ := s.MappedName(name)
		inner := string(mapped)
		if known.Contains(inner) {
			return nil, fmt.Errorf(
				"%w: %s", ErrCompensateArgConflict, inner,
			)
		}
		known.Add(inner)
	}
	return known, nil
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
	if s.Flow.SpaceID != "" && SanitizeID(s.Flow.SpaceID) != s.Flow.SpaceID {
		return ErrSpaceIDInvalid
	}
	return nil
}

func (s *Step) validateMappingNames() error {
	inputInnerNames := map[string]Name{}
	outputInnerNames := map[string]Name{}

	for name, attr := range s.Attributes {
		mapped, ok := s.MappedName(name)
		if !ok {
			continue
		}

		if attr.IsRuntimeInput() {
			if _, ok := inputInnerNames[string(mapped)]; ok {
				return fmt.Errorf("%w: %q", ErrDuplicateInnerName, mapped)
			}
			inputInnerNames[string(mapped)] = name
		}

		if attr.IsOutput() {
			if _, ok := outputInnerNames[string(mapped)]; ok {
				return fmt.Errorf("%w: %q", ErrDuplicateInnerName, mapped)
			}
			outputInnerNames[string(mapped)] = name
		}
	}

	return nil
}

func (s *Step) validateWorkConfig() error {
	if s.WorkConfig == nil {
		return nil
	}

	if s.WorkConfig.Parallelism < 0 {
		return ErrInvalidParallelism
	}

	if s.WorkConfig.InitBackoff < 0 {
		return ErrNegativeBackoff
	}

	if s.WorkConfig.MaxBackoff != 0 &&
		s.WorkConfig.MaxBackoff < s.WorkConfig.InitBackoff {
		return ErrMaxBackoffTooSmall
	}

	if s.WorkConfig.BackoffType != "" &&
		!validBackoffTypes.Contains(s.WorkConfig.BackoffType) {
		return ErrInvalidBackoffType
	}

	return nil
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

	var httpCfg *HTTPConfig
	if s.HTTP != nil {
		httpCfg = util.MutableCopy(s.HTTP)
		httpCfg.Invoke.Method = s.HTTP.Invoke.DefaultedMethod()
		if s.HTTP.Compensate != nil {
			comp := util.MutableCopy(s.HTTP.Compensate)
			comp.Method = s.HTTP.Compensate.DefaultedMethod()
			httpCfg.Compensate = comp
		}
	}

	h := stepHash{
		Type:       s.Type,
		Handling:   s.DefaultedHandling(),
		Attributes: attrs,
		HTTP:       httpCfg,
		Script:     s.Script,
		Flow:       s.Flow,
		Predicate:  s.Predicate,
		WorkConfig: s.WorkConfig,
	}

	data, err := json.Marshal(h)
	if err != nil {
		return "", errors.Join(ErrMarshalStep, err)
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

// Equal returns true if two HTTP configs are equal
func (h *HTTPConfig) Equal(other *HTTPConfig) bool {
	if h == nil || other == nil {
		return h == other
	}
	return h.Invoke.Equal(&other.Invoke) &&
		h.Compensate.Equal(other.Compensate) &&
		h.Health == other.Health
}

// CompensateTimeout returns the compensate timeout, falling back to the invoke
// timeout when the action does not set its own
func (h *HTTPConfig) CompensateTimeout() int64 {
	if h.Compensate != nil && h.Compensate.Timeout > 0 {
		return h.Compensate.Timeout
	}
	return h.Invoke.Timeout
}

// Equal returns true if two HTTP actions are equal
func (a *HTTPAction) Equal(other *HTTPAction) bool {
	if a == nil || other == nil {
		return a == other
	}
	return a.Endpoint == other.Endpoint &&
		a.DefaultedMethod() == other.DefaultedMethod() &&
		a.DefaultedMode() == other.DefaultedMode() &&
		a.Timeout == other.Timeout
}

// DefaultedMethod returns the configured HTTP method or the default if unset
func (a *HTTPAction) DefaultedMethod() string {
	if a == nil || a.Method == "" {
		return DefaultHTTPMethod
	}
	return a.Method
}

// DefaultedMode returns the configured action mode or the default if unset
func (a *HTTPAction) DefaultedMode() ActionMode {
	if a == nil || a.Mode == "" {
		return DefaultActionMode
	}
	return a.Mode
}

// Async returns true if the action reports its result through a callback
func (a *HTTPAction) Async() bool {
	return a.DefaultedMode() == ActionModeAsync
}

// Equal returns true if two script configs are equal
func (c *ScriptConfig) Equal(other *ScriptConfig) bool {
	if c == nil || other == nil {
		return c == other
	}
	return c.Language == other.Language && c.Script == other.Script
}

// WithGoals returns a copy of the flow config with the provided goals
func (c *FlowConfig) WithGoals(goals ...StepID) *FlowConfig {
	res := util.MutableCopy(c)
	res.Goals = goals
	return res
}

// Equal returns true if two flow configs are equal
func (c *FlowConfig) Equal(other *FlowConfig) bool {
	if c == nil || other == nil {
		return c == other
	}
	return c.SpaceID == other.SpaceID &&
		c.Compensate == other.Compensate &&
		slices.Equal(c.Goals, other.Goals)
}

// Equal returns true if two work configs are equal
func (c *WorkConfig) Equal(other *WorkConfig) bool {
	if c == nil || other == nil {
		return c == other
	}
	sameLimits := c.Parallelism == other.Parallelism &&
		c.MaxRetries == other.MaxRetries
	sameBackoff := c.InitBackoff == other.InitBackoff &&
		c.MaxBackoff == other.MaxBackoff &&
		c.BackoffType == other.BackoffType
	return sameLimits && sameBackoff
}

// Equal returns true if two normalized tag sets are equal
func (t Tags) Equal(other Tags) bool {
	return slices.Equal(t, other)
}

// Normalize returns the tags sorted and deduped, so that equality and digests
// do not depend on declaration order
func (t Tags) Normalize() Tags {
	if len(t) == 0 {
		return t
	}
	return slices.Compact(slices.Sorted(slices.Values(t)))
}

func validateAction(act *HTTPAction) error {
	if act.Endpoint == "" {
		return ErrStepEndpointEmpty
	}
	method := act.DefaultedMethod()
	if !validHTTPMethods.Contains(method) {
		return fmt.Errorf("%w: %s", ErrInvalidHTTPMethod, method)
	}
	mode := act.DefaultedMode()
	if !validActionModes.Contains(mode) {
		return fmt.Errorf("%w: %s", ErrInvalidActionMode, mode)
	}
	return nil
}

func endpointParams(endpoint string) util.Set[string] {
	matches := endpointParamPattern.FindAllStringSubmatch(endpoint, -1)
	res := make(util.Set[string], len(matches))
	for _, m := range matches {
		if len(m) < 2 || m[1] == "" {
			continue
		}
		res.Add(m[1])
	}
	return res
}
