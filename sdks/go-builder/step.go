package builder

import (
	"context"
	"errors"
	"maps"
	"regexp"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// Step is a builder for creating and configuring flow steps. It provides an
// API for defining step attributes, predicates, and execution settings
type Step struct {
	client     *Client
	predicate  *api.ScriptConfig
	http       *api.HTTPConfig
	flow       *api.FlowConfig
	script     *api.ScriptConfig
	id         api.StepID
	name       api.Name
	stepType   api.StepType
	labels     api.Labels
	attributes api.AttributeSpecs
	timeout    int64
	memoizable bool
	dirty      bool
}

var (
	camelCaseRegex = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	delimiterRegex = regexp.MustCompile(`[\s_]+`)
)

// NewStep creates a new step builder template
func (c *Client) NewStep() Step {
	return Step{
		client:     c,
		stepType:   api.StepTypeSync,
		labels:     api.Labels{},
		timeout:    30 * api.Second,
		attributes: api.AttributeSpecs{},
	}
}

// WithID sets the step ID, overriding the auto-generated ID from the step name
func (s Step) WithID(id string) Step {
	s.id = api.StepID(id)
	return s
}

// WithName sets the step name. If no ID is set, it will be derived
func (s Step) WithName(name api.Name) Step {
	s.name = name
	if s.id == "" && name != "" {
		s.id = api.StepID(toSnakeCase(string(name)))
	}
	return s
}

// Required declares a required input attribute for the step
func (s Step) Required(name api.Name, argType api.AttributeType) Step {
	s.attributes = maps.Clone(s.attributes)
	s.attributes[name] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: argType,
	}
	return s
}

// Optional declares an optional input attribute with a default value
func (s Step) Optional(
	name api.Name, argType api.AttributeType, defaultValue string,
) Step {
	s.attributes = maps.Clone(s.attributes)
	s.attributes[name] = &api.AttributeSpec{
		Role:    api.RoleOptional,
		Type:    argType,
		Default: defaultValue,
	}
	return s
}

// Const declares a const input attribute with a default value
func (s Step) Const(
	name api.Name, argType api.AttributeType, defaultValue string,
) Step {
	s.attributes = maps.Clone(s.attributes)
	s.attributes[name] = &api.AttributeSpec{
		Role:    api.RoleConst,
		Type:    argType,
		Default: defaultValue,
	}
	return s
}

// Output declares an output attribute that the step will produce
func (s Step) Output(name api.Name, argType api.AttributeType) Step {
	s.attributes = maps.Clone(s.attributes)
	s.attributes[name] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: argType,
	}
	return s
}

// WithForEach marks an attribute as supporting multi work items (arrays)
func (s Step) WithForEach(name api.Name) Step {
	s.attributes = maps.Clone(s.attributes)
	if attr, ok := s.attributes[name]; ok {
		newAttr := *attr
		newAttr.ForEach = true
		s.attributes[name] = &newAttr
	}
	return s
}

// WithLabel sets a single label for the step
func (s Step) WithLabel(key, value string) Step {
	return s.WithLabels(api.Labels{key: value})
}

// WithLabels merges the provided labels into the step's labels
func (s Step) WithLabels(labels api.Labels) Step {
	if len(labels) == 0 {
		return s
	}
	s.labels = s.labels.Apply(labels)
	return s
}

// WithPredicate sets a predicate script that determines if the step should
// execute
func (s Step) WithPredicate(language, script string) Step {
	s.predicate = &api.ScriptConfig{
		Language: language,
		Script:   script,
	}
	return s
}

// WithAlePredicate sets an Ale language predicate script
func (s Step) WithAlePredicate(script string) Step {
	return s.WithPredicate(api.ScriptLangAle, script)
}

// WithLuaPredicate sets a Lua language predicate script
func (s Step) WithLuaPredicate(script string) Step {
	return s.WithPredicate(api.ScriptLangLua, script)
}

// WithEndpoint sets the HTTP endpoint where the step handler is listening
func (s Step) WithEndpoint(endpoint string) Step {
	if s.http == nil {
		s.http = &api.HTTPConfig{}
	} else {
		httpCopy := *s.http
		s.http = &httpCopy
	}
	s.http.Endpoint = endpoint
	if s.stepType == "" {
		s.stepType = api.StepTypeSync
	}
	return s
}

// WithFlowGoals configures a flow step with child flow goal IDs
func (s Step) WithFlowGoals(goals ...api.StepID) Step {
	if s.flow == nil {
		s.flow = &api.FlowConfig{}
	}
	s.flow = s.flow.WithGoals(goals...)
	s.stepType = api.StepTypeFlow
	return s
}

// WithScript sets an Ale script to execute for this step
func (s Step) WithScript(script string) Step {
	s.script = &api.ScriptConfig{
		Language: api.ScriptLangAle,
		Script:   script,
	}
	s.stepType = api.StepTypeScript
	return s
}

// WithScriptLanguage sets a script with a specific language to execute for
// this step
func (s Step) WithScriptLanguage(lang, script string) Step {
	s.script = &api.ScriptConfig{
		Language: lang,
		Script:   script,
	}
	s.stepType = api.StepTypeScript
	return s
}

// WithHealthCheck sets the HTTP health check endpoint for the step
func (s Step) WithHealthCheck(endpoint string) Step {
	if s.http == nil {
		s.http = &api.HTTPConfig{}
	} else {
		httpCopy := *s.http
		s.http = &httpCopy
	}
	s.http.HealthCheck = endpoint
	return s
}

// WithTimeout sets the execution timeout for the step in milliseconds
func (s Step) WithTimeout(timeout int64) Step {
	s.timeout = timeout
	return s
}

// WithType sets the step execution type (sync, async, or script)
func (s Step) WithType(stepType api.StepType) Step {
	s.stepType = stepType
	return s
}

// WithAsyncExecution configures the step to execute asynchronously
func (s Step) WithAsyncExecution() Step {
	s.stepType = api.StepTypeAsync
	return s
}

// WithSyncExecution configures the step to execute synchronously
func (s Step) WithSyncExecution() Step {
	s.stepType = api.StepTypeSync
	return s
}

// WithScriptExecution configures the step to execute via a script
func (s Step) WithScriptExecution() Step {
	s.stepType = api.StepTypeScript
	return s
}

// WithMemoizable marks the step as eligible for result memoization
func (s Step) WithMemoizable() Step {
	s.memoizable = true
	return s
}

// Build validates and creates the final Step API object
func (s Step) Build() (*api.Step, error) {
	if s.name != "" && s.id == "" {
		return s.WithName(s.name).Build()
	}
	var httpConfig *api.HTTPConfig
	if s.http != nil {
		httpCopy := *s.http
		httpCopy.Timeout = s.timeout
		httpConfig = &httpCopy
	}

	st := &api.Step{
		ID:         s.id,
		Name:       s.name,
		Type:       s.stepType,
		Attributes: s.attributes,
		Labels:     s.labels,
		Predicate:  s.predicate,
		HTTP:       httpConfig,
		Flow:       s.flow,
		Script:     s.script,
		Memoizable: s.memoizable,
	}

	if err := st.Validate(); err != nil {
		return nil, err
	}

	return st, nil
}

// Register builds and registers the step with the engine
func (s Step) Register(ctx context.Context) error {
	step, err := s.Build()
	if err != nil {
		return err
	}

	if s.client == nil {
		return errors.New("step not created from client")
	}

	return s.client.registerStep(ctx, step)
}

// Update marks this step as modified, so the next Start() will update the
// existing step registration rather than creating a new one
func (s Step) Update() Step {
	s.dirty = true
	return s
}

// Start builds and registers the step, creates an HTTP server, and starts
// handling requests
func (s Step) Start(handler StepHandler) error {
	if s.client == nil {
		return errors.New("step not created from client")
	}

	return setupStepServer(s.client, s, handler)
}

func toSnakeCase(s string) string {
	s = camelCaseRegex.ReplaceAllString(s, "$1-$2")
	s = delimiterRegex.ReplaceAllString(s, "-")
	return strings.ToLower(s)
}
