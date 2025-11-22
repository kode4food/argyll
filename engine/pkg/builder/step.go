package builder

import (
	"context"
	"errors"
	"maps"
	"regexp"
	"strings"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

// Step is a builder for creating and configuring flow steps. It provides
// an API for defining step attributes, predicates, and execution settings
type Step struct {
	client     *Client
	predicate  *api.ScriptConfig
	http       *api.HTTPConfig
	script     *api.ScriptConfig
	id         StepID
	name       api.Name
	stepType   api.StepType
	version    string
	attributes map[api.Name]*api.AttributeSpec
	timeout    int64
	dirty      bool
}

var (
	camelCaseRegex = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	delimiterRegex = regexp.MustCompile(`[\s_]+`)
)

// NewStep creates a new step builder with the specified name
func (c *Client) NewStep(name api.Name) *Step {
	id := StepID(toSnakeCase(string(name)))
	return &Step{
		client:     c,
		id:         id,
		name:       name,
		version:    "1.0.0",
		stepType:   api.StepTypeSync,
		timeout:    30 * api.Second,
		attributes: map[api.Name]*api.AttributeSpec{},
	}
}

// WithID sets the step ID, overriding the auto-generated ID from the step name
func (s *Step) WithID(id string) *Step {
	res := *s
	res.id = StepID(id)
	return &res
}

// Required declares a required input attribute for the step
func (s *Step) Required(name api.Name, argType api.AttributeType) *Step {
	res := *s
	res.attributes = maps.Clone(res.attributes)
	res.attributes[name] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: argType,
	}
	return &res
}

// Optional declares an optional input attribute with a default value
func (s *Step) Optional(
	name api.Name, argType api.AttributeType, defaultValue string,
) *Step {
	res := *s
	res.attributes = maps.Clone(res.attributes)
	res.attributes[name] = &api.AttributeSpec{
		Role:    api.RoleOptional,
		Type:    argType,
		Default: defaultValue,
	}
	return &res
}

// Output declares an output attribute that the step will produce
func (s *Step) Output(name api.Name, argType api.AttributeType) *Step {
	res := *s
	res.attributes = maps.Clone(res.attributes)
	res.attributes[name] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: argType,
	}
	return &res
}

// WithForEach marks an attribute as supporting multi work items (arrays)
func (s *Step) WithForEach(name api.Name) *Step {
	res := *s
	res.attributes = maps.Clone(res.attributes)
	if attr, ok := res.attributes[name]; ok {
		newAttr := *attr
		newAttr.ForEach = true
		res.attributes[name] = &newAttr
	}
	return &res
}

// WithPredicate sets a predicate script that determines if the step should
// execute
func (s *Step) WithPredicate(language, script string) *Step {
	res := *s
	res.predicate = &api.ScriptConfig{
		Language: language,
		Script:   script,
	}
	return &res
}

// WithAlePredicate sets an Ale language predicate script
func (s *Step) WithAlePredicate(script string) *Step {
	return s.WithPredicate(api.ScriptLangAle, script)
}

// WithLuaPredicate sets a Lua language predicate script
func (s *Step) WithLuaPredicate(script string) *Step {
	return s.WithPredicate(api.ScriptLangLua, script)
}

// WithVersion sets the step version
func (s *Step) WithVersion(version string) *Step {
	res := *s
	res.version = version
	return &res
}

// WithEndpoint sets the HTTP endpoint where the step handler is listening
func (s *Step) WithEndpoint(endpoint string) *Step {
	res := *s
	if res.http == nil {
		res.http = &api.HTTPConfig{}
	} else {
		httpCopy := *res.http
		res.http = &httpCopy
	}
	res.http.Endpoint = endpoint
	if res.stepType == "" {
		res.stepType = api.StepTypeSync
	}
	return &res
}

// WithScript sets an Ale script to execute for this step
func (s *Step) WithScript(script string) *Step {
	res := *s
	res.script = &api.ScriptConfig{
		Language: api.ScriptLangAle,
		Script:   script,
	}
	res.stepType = api.StepTypeScript
	return &res
}

// WithScriptLanguage sets a script with a specific language to execute for
// this step
func (s *Step) WithScriptLanguage(lang, script string) *Step {
	res := *s
	res.script = &api.ScriptConfig{
		Language: lang,
		Script:   script,
	}
	res.stepType = api.StepTypeScript
	return &res
}

// WithHealthCheck sets the HTTP health check endpoint for the step
func (s *Step) WithHealthCheck(endpoint string) *Step {
	res := *s
	if res.http == nil {
		res.http = &api.HTTPConfig{}
	} else {
		httpCopy := *res.http
		res.http = &httpCopy
	}
	res.http.HealthCheck = endpoint
	return &res
}

// WithTimeout sets the execution timeout for the step in milliseconds
func (s *Step) WithTimeout(timeout int64) *Step {
	res := *s
	res.timeout = timeout
	return &res
}

// WithType sets the step execution type (sync, async, or script)
func (s *Step) WithType(stepType api.StepType) *Step {
	res := *s
	res.stepType = stepType
	return &res
}

// WithAsyncExecution configures the step to execute asynchronously
func (s *Step) WithAsyncExecution() *Step {
	res := *s
	res.stepType = api.StepTypeAsync
	return &res
}

// WithSyncExecution configures the step to execute synchronously
func (s *Step) WithSyncExecution() *Step {
	res := *s
	res.stepType = api.StepTypeSync
	return &res
}

// WithScriptExecution configures the step to execute via a script
func (s *Step) WithScriptExecution() *Step {
	res := *s
	res.stepType = api.StepTypeScript
	return &res
}

// Build validates and creates the final Step API object
func (s *Step) Build() (*api.Step, error) {
	var httpConfig *api.HTTPConfig
	if s.http != nil {
		httpCopy := *s.http
		httpCopy.Timeout = s.timeout
		httpConfig = &httpCopy
	}

	step := &api.Step{
		ID:         timebox.ID(s.id),
		Name:       s.name,
		Type:       s.stepType,
		Attributes: s.attributes,
		Predicate:  s.predicate,
		Version:    s.version,
		HTTP:       httpConfig,
		Script:     s.script,
	}

	if err := step.Validate(); err != nil {
		return nil, err
	}

	return step, nil
}

// Register builds and registers the step with the engine
func (s *Step) Register(ctx context.Context) error {
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
func (s *Step) Update() *Step {
	res := *s
	res.dirty = true
	return &res
}

// Start builds the step, registers it with the engine, creates an HTTP server,
// and starts handling requests. Automatically registers the step before
// starting the server
func (s *Step) Start(handler StepHandler) error {
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
