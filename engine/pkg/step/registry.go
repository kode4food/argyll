// Package step defines the contract between Argyll and step implementations
package step

import (
	"errors"
	"fmt"
	"maps"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// Runtime exposes engine services available during work execution
	Runtime interface {
		FlowID() api.FlowID
		StepID() api.StepID
		Metadata() api.Metadata
		CompleteWork(api.Token, api.Args) error
		UpdateHealth(api.HealthStatus, string) error
	}

	// Handler describes the capabilities supplied by a step implementation
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

	// HealthFunc reports whether a step can currently run
	HealthFunc func(*api.Step) api.HealthState

	// ChildrenFunc reports the step IDs a step expands into
	ChildrenFunc func(*api.Step) []api.StepID

	// CompensateFunc reverses a completed work item. True reports completion;
	// false with no error leaves it awaiting an asynchronous callback
	CompensateFunc func(CompensateRequest) (bool, error)

	// CompensateRequest carries the work item being reversed
	CompensateRequest struct {
		Step     *api.Step
		Inputs   api.Args
		Outputs  api.Args
		Metadata api.Metadata
		FlowID   api.FlowID
		Token    api.Token
	}

	// Handlers maps each step type to the implementation that runs it
	Handlers map[api.StepType]*Handler

	// Registry answers which handler runs a step
	Registry struct {
		handlers Handlers
	}
)

var (
	ErrCompensatorRequired = errors.New("compensator required")
)

// NewRegistry freezes a handler set for lock-free lookup
func NewRegistry(handlers Handlers) *Registry {
	return &Registry{handlers: maps.Clone(handlers)}
}

// Lookup returns the handler registered for a step type
func (r *Registry) Lookup(stepType api.StepType) (*Handler, error) {
	handler, ok := r.handlers[stepType]
	if !ok {
		return nil, fmt.Errorf("%w: %s", api.ErrInvalidStepType, stepType)
	}
	return handler, nil
}

// Validate dispatches step-type-specific validation to the handler
func (r *Registry) Validate(st *api.Step) error {
	handler, err := r.Lookup(st.Type)
	if err != nil {
		return err
	}
	if st.CanCompensate() && handler.Compensate == nil {
		return ErrCompensatorRequired
	}
	if handler.Validate == nil {
		return nil
	}
	return handler.Validate(st)
}

// Health reports a step's health, unknown when the handler evaluates none
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

// Children returns the step IDs a step expands into
func (r *Registry) Children(st *api.Step) ([]api.StepID, error) {
	handler, err := r.Lookup(st.Type)
	if err != nil || handler.Children == nil {
		return nil, err
	}
	return handler.Children(st), nil
}

// Compensator returns the step's compensation, nil when it has none
func (r *Registry) Compensator(st *api.Step) (CompensateFunc, error) {
	handler, err := r.Lookup(st.Type)
	if err != nil || !st.CanCompensate() {
		return nil, err
	}
	return handler.Compensate, nil
}
