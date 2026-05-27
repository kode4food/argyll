package step

import (
	"fmt"
	"sync"

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

	// Registry stores step handlers keyed by step type
	Registry struct {
		mu       sync.RWMutex
		handlers map[api.StepType]Handler
	}
)

// NewRegistry creates a step handler registry with the built-in handlers
func NewRegistry(scripts *script.Registry, c client.Client) *Registry {
	r := &Registry{handlers: map[api.StepType]Handler{}}
	RegisterScriptHandler(r, scripts)
	RegisterHTTPHandler(r, c)
	RegisterFlowHandler(r)
	return r
}

// Register registers a handler for a step type
func (r *Registry) Register(stepType api.StepType, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers[stepType] = handler
}

// Lookup returns the handler for a step type
func (r *Registry) Lookup(stepType api.StepType) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, ok := r.handlers[stepType]
	if !ok {
		return Handler{}, fmt.Errorf("%w: %s", api.ErrInvalidStepType, stepType)
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
