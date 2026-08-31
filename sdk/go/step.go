package argyll

import (
	"context"
	"errors"
	"maps"
	"regexp"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// Step is a builder for creating and configuring flow steps. It provides an
// API for defining step attributes, predicates, and execution settings
type Step struct {
	client     *Client
	step       *api.Step
	compensate CompensateHandler
	timeout    int64
}

var (
	ErrDetachedStep = errors.New("step not created from client")
)

var (
	camelCaseRegex = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	delimiterRegex = regexp.MustCompile(`[\s_]+`)
)

// NewStep creates a new step builder template
func (c *Client) NewStep() Step {
	return Step{
		client:  c,
		timeout: 30 * api.Second,
		step: &api.Step{
			Type:       api.StepTypeService,
			Labels:     api.Labels{},
			Attributes: api.AttributeSpecs{},
		},
	}
}

// WithID sets the step ID, overriding the auto-generated ID from the step name
func (s Step) WithID(id string) Step {
	s.step = s.step.Copy()
	s.step.ID = api.StepID(id)
	return s
}

// WithName sets the step name. If no ID is set, it will be derived
func (s Step) WithName(name api.Name) Step {
	s.step = s.step.Copy()
	s.step.Name = name
	if s.step.ID == "" && name != "" {
		s.step.ID = api.StepID(toSnakeCase(string(name)))
	}
	return s
}

// Required declares a required input attribute for the step
func (s Step) Required(name api.Name, argType api.AttributeType) Step {
	return s.withAttribute(name, &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: argType,
	})
}

// Optional declares an optional input attribute with a default value
func (s Step) Optional(
	name api.Name, argType api.AttributeType, defaultValue string,
) Step {
	return s.withAttribute(name, &api.AttributeSpec{
		Role:     api.RoleOptional,
		Type:     argType,
		Optional: &api.OptionalConfig{Default: defaultValue},
	})
}

// Const declares a const input attribute with a fixed value
func (s Step) Const(
	name api.Name, argType api.AttributeType, defaultValue string,
) Step {
	return s.withAttribute(name, &api.AttributeSpec{
		Role:  api.RoleConst,
		Type:  argType,
		Const: &api.ConstConfig{Value: defaultValue},
	})
}

// Meta declares a metadata input attribute, injecting the named metadata key
// as a step input at execution time
func (s Step) Meta(name api.Name, metaKey string) Step {
	return s.withAttribute(name, &api.AttributeSpec{
		Role: api.RoleMeta,
		Type: api.TypeAny,
		Meta: &api.MetaConfig{Key: metaKey},
	})
}

// Output declares an output attribute that the step will produce
func (s Step) Output(name api.Name, argType api.AttributeType) Step {
	return s.withAttribute(name, &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: argType,
	})
}

// WithForEach marks attributes as supporting multi work items (arrays)
func (s Step) WithForEach(names ...api.Name) Step {
	for _, name := range names {
		attr, ok := s.step.Attributes[name]
		if !ok {
			continue
		}
		cpy := util.MutableCopy(attr)
		switch cpy.Role {
		case api.RoleRequired:
			cpy.Required = util.MutableCopy(cpy.Required)
			cpy.Required.ForEach = true
		case api.RoleOptional:
			cpy.Optional = util.MutableCopy(cpy.Optional)
			cpy.Optional.ForEach = true
		}
		s = s.withAttribute(name, cpy)
	}
	return s
}

// WithLabels merges the provided labels into the step's labels
func (s Step) WithLabels(labels api.Labels) Step {
	if len(labels) == 0 {
		return s
	}
	s.step = s.step.Copy()
	s.step.Labels = s.step.Labels.Apply(labels)
	return s
}

// WithPredicate sets a predicate script that determines if the step should
// execute
func (s Step) WithPredicate(predicate api.ScriptConfig) Step {
	s.step = s.step.Copy()
	s.step.Predicate = &predicate
	return s
}

// WithRequiredMatch sets a match predicate for a required attribute. The
// predicate receives each candidate attribute value as "value" before collect
// semantics are applied
func (s Step) WithRequiredMatch(name api.Name, match api.ScriptConfig) Step {
	attr, ok := s.step.Attributes[name]
	if !ok || !attr.IsRequired() {
		return s
	}
	cpy := util.MutableCopy(attr)
	cpy.Required = util.MutableCopy(cpy.Required)
	cpy.Required.Match = &match
	return s.withAttribute(name, cpy)
}

// WithEndpoint sets the HTTP endpoint where the step handler is listening
func (s Step) WithEndpoint(endpoint string) Step {
	return s.withHTTP(func(http *api.HTTPConfig) {
		http.Invoke.Endpoint = endpoint
	})
}

// WithMethod sets the HTTP method used to invoke the step endpoint
func (s Step) WithMethod(method string) Step {
	return s.withHTTP(func(http *api.HTTPConfig) {
		http.Invoke.Method = strings.ToUpper(method)
	})
}

// WithFlowGoals configures a flow step with child flow goal IDs
func (s Step) WithFlowGoals(goals ...api.StepID) Step {
	s.step = s.step.Copy()
	if s.step.Flow == nil {
		s.step.Flow = &api.FlowConfig{}
	}
	s.step.Flow = s.step.Flow.WithGoals(goals...)
	s.step.Type = api.StepTypeFlow
	return s
}

// WithFlowSpace restricts a child flow to the specified planning space
func (s Step) WithFlowSpace(spaceID api.SpaceID) Step {
	s.step = s.step.Copy()
	s.step.Flow = util.MutableCopy(s.step.Flow)
	s.step.Flow.SpaceID = spaceID
	s.step.Type = api.StepTypeFlow
	return s
}

// WithScript sets the script to execute for this step
func (s Step) WithScript(script api.ScriptConfig) Step {
	s.step = s.step.Copy()
	s.step.Script = &script
	s.step.Type = api.StepTypeScript
	return s
}

// WithHealthCheck sets the HTTP health check endpoint for the step
func (s Step) WithHealthCheck(endpoint string) Step {
	return s.withHTTP(func(http *api.HTTPConfig) {
		http.Health = endpoint
	})
}

// WithCompensate sets the compensate endpoint and selects compensated handling
func (s Step) WithCompensate(endpoint string) Step {
	return s.withCompensate(func(comp *api.HTTPAction) {
		comp.Endpoint = endpoint
	})
}

// WithCompensateMethod sets the HTTP method used to compensate the step
func (s Step) WithCompensateMethod(method string) Step {
	return s.withCompensate(func(comp *api.HTTPAction) {
		comp.Method = strings.ToUpper(method)
	})
}

// WithCompensateTimeout sets the compensate timeout in milliseconds,
// overriding the step's execution timeout for compensation requests
func (s Step) WithCompensateTimeout(timeout int64) Step {
	return s.withCompensate(func(comp *api.HTTPAction) {
		comp.Timeout = timeout
	})
}

// WithCompensateHandler registers a handler for compensation requests
func (s Step) WithCompensateHandler(handler CompensateHandler) Step {
	s.compensate = handler
	return s
}

// WithTimeout sets the execution timeout for the step in milliseconds
func (s Step) WithTimeout(timeout int64) Step {
	s.timeout = timeout
	return s
}

// WithType sets the step execution type (service or script)
func (s Step) WithType(stepType api.StepType) Step {
	s.step = s.step.Copy()
	s.step.Type = stepType
	return s
}

// WithAsyncExecution configures the step's invoke call to complete via
// webhook rather than in the HTTP response
func (s Step) WithAsyncExecution() Step {
	return s.WithInvokeMode(api.ActionModeAsync)
}

// WithSyncExecution configures the step's invoke call to complete in the
// HTTP response
func (s Step) WithSyncExecution() Step {
	return s.WithInvokeMode(api.ActionModeSync)
}

// WithInvokeMode sets how the step's invoke call reports its result
func (s Step) WithInvokeMode(mode api.ActionMode) Step {
	return s.withHTTP(func(http *api.HTTPConfig) {
		http.Invoke.Mode = mode
	})
}

// WithCompensateMode sets how the step's compensate call reports its result
func (s Step) WithCompensateMode(mode api.ActionMode) Step {
	return s.withCompensate(func(comp *api.HTTPAction) {
		comp.Mode = mode
	})
}

// WithScriptExecution configures the step to execute via a script
func (s Step) WithScriptExecution() Step {
	return s.WithType(api.StepTypeScript)
}

// WithHandling sets how completed work is retained or reversed
func (s Step) WithHandling(handling api.Handling) Step {
	s.step = s.step.Copy()
	s.step.Handling = handling
	return s
}

// WithCompensated includes attributes in compensation requests
func (s Step) WithCompensated(names ...api.Name) Step {
	for _, name := range names {
		attr, ok := s.step.Attributes[name]
		if !ok {
			continue
		}
		cpy := util.MutableCopy(attr)
		cpy.Compensated = true
		s = s.withAttribute(name, cpy)
	}
	return s
}

// Build validates and creates the final Step API object
func (s Step) Build() (*api.Step, error) {
	res := s.step.Copy()
	if res.Name != "" && res.ID == "" {
		res.ID = api.StepID(toSnakeCase(string(res.Name)))
	}
	if res.HTTP != nil {
		res.HTTP = util.MutableCopy(res.HTTP)
		res.HTTP.Invoke.Timeout = s.timeout
	}
	if err := res.Validate(); err != nil {
		return nil, err
	}
	return res, nil
}

// Register builds and registers the step with the engine
func (s Step) Register(ctx context.Context) error {
	st, err := s.Build()
	if err != nil {
		return err
	}

	if s.client == nil {
		return ErrDetachedStep
	}

	return s.client.RegisterStep(ctx, st)
}

// Start builds and registers the step, creates an HTTP server, and starts
// handling requests
func (s Step) Start(handler StepHandler) error {
	if s.client == nil {
		return ErrDetachedStep
	}

	return setupStepServer(s.client, s, handler)
}

func (s Step) withAttribute(name api.Name, attr *api.AttributeSpec) Step {
	s.step = s.step.Copy()
	s.step.Attributes = maps.Clone(s.step.Attributes)
	s.step.Attributes[name] = attr
	return s
}

func (s Step) withHTTP(mutate func(*api.HTTPConfig)) Step {
	s.step = s.step.Copy()
	s.step.HTTP = util.MutableCopy(s.step.HTTP)
	mutate(s.step.HTTP)
	if s.step.Type == "" {
		s.step.Type = api.StepTypeService
	}
	return s
}

func (s Step) withCompensate(mutate func(*api.HTTPAction)) Step {
	s = s.withHTTP(func(http *api.HTTPConfig) {
		comp := util.MutableCopy(http.Compensate)
		mutate(comp)
		http.Compensate = comp
	})
	s.step.Handling = api.HandlingCompensated
	return s
}

func toSnakeCase(s string) string {
	s = camelCaseRegex.ReplaceAllString(s, "$1-$2")
	s = delimiterRegex.ReplaceAllString(s, "-")
	return strings.ToLower(s)
}
