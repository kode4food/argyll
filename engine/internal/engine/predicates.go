package engine

import (
	"errors"

	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	ErrStepNotInPlan = errors.New("step not in execution plan")
)

// StepHealth evaluates handler-provided health for a step
func (e *Engine) StepHealth(st *api.Step) (api.HealthState, error) {
	return e.steps.Health(st)
}

// Children returns the child step IDs a step expands into
func (e *Engine) Children(st *api.Step) ([]api.StepID, error) {
	return e.steps.Children(st)
}
