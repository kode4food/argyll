package step

import (
	"fmt"
	"maps"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// Runtime exposes engine services available during work execution
	Runtime interface {
		FlowID() api.FlowID
		StepID() api.StepID
		Metadata() api.Metadata
		WebhookURL(api.Token) string
		CompleteWork(api.Token, api.Args) error
		StartChildFlow(api.Token, api.InitArgs) (api.FlowID, error)
		UpdateHealth(api.HealthStatus, string) error
	}

	// Handler describes step-type-specific behavior. The engine validates the
	// common step shape (scripts, mappings, attributes) before dispatching to
	// Validate and Execute. All capability fields except Execute may
	// be nil
	Handler struct {
		Validate   ValidateFunc
		Execute    ExecuteFunc
		Health     HealthFunc
		Children   ChildrenFunc
		Compensate CompensateFunc
	}

	// ValidateFunc validates step-type-specific configuration
	ValidateFunc func(*api.Step) error

	// ExecuteFunc runs a step's work item
	ExecuteFunc func(Runtime, *api.Step, api.Args, api.Token) error

	// HealthFunc actively evaluates a step's health
	HealthFunc func(*api.Step) api.HealthState

	// ChildrenFunc reports the step IDs a step expands into. Step types that
	// do not expand into child steps leave this nil
	ChildrenFunc func(*api.Step) []api.StepID

	// CompensateFunc reverses a step's effects after it has completed. Step
	// types without compensation semantics leave this nil
	CompensateFunc func(*api.Step, api.Args, api.Args, api.Metadata) error

	// Handlers maps bootstrapped step types to their implementations
	Handlers map[api.StepType]*Handler

	// Registry stores step handlers keyed by step type
	Registry struct {
		handlers Handlers
	}
)

// DefaultHandlers constructs the built-in step handler set
func DefaultHandlers(scripts *script.Registry, c client.Client) Handlers {
	return Handlers{
		api.StepTypeScript: scriptHandler(scripts),
		api.StepTypeSync:   httpHandler(c, false),
		api.StepTypeAsync:  httpHandler(c, true),
		api.StepTypeFlow:   flowHandler(),
	}
}

// NewRegistry freezes a bootstrapped handler set for lock-free lookup
func NewRegistry(handlers Handlers) *Registry {
	return &Registry{
		handlers: maps.Clone(handlers),
	}
}

// Lookup returns the handler for a step type
func (r *Registry) Lookup(stepType api.StepType) (*Handler, error) {
	handler, ok := r.handlers[stepType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", api.ErrInvalidStepType, stepType)
	}
	return handler, nil
}

// Validate dispatches step-type-specific validation to the matching handler
func (r *Registry) Validate(st *api.Step) error {
	handler, err := r.Lookup(st.Type)
	if err != nil {
		return err
	}
	if handler.Validate == nil {
		return nil
	}
	return handler.Validate(st)
}

// Health dispatches step-type-specific health evaluation
func (r *Registry) Health(st *api.Step) (api.HealthState, error) {
	handler, err := r.Lookup(st.Type)
	if err != nil {
		return api.HealthState{}, err
	}
	if handler.Health == nil {
		return api.HealthState{Status: api.HealthUnknown}, nil
	}
	return handler.Health(st), nil
}

// Children returns the child step IDs this step expands into
func (r *Registry) Children(st *api.Step) ([]api.StepID, error) {
	handler, err := r.Lookup(st.Type)
	if err != nil {
		return nil, err
	}
	if handler.Children == nil {
		return nil, nil
	}
	return handler.Children(st), nil
}

// Compensator returns the step's compensation function
func (r *Registry) Compensator(st *api.Step) (CompensateFunc, error) {
	handler, err := r.Lookup(st.Type)
	if err != nil {
		return nil, err
	}
	return handler.Compensate, nil
}
