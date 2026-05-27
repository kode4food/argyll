package engine

import (
	"errors"

	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	ErrStepNotInPlan = errors.New("step not in execution plan")
)

// StepHealth evaluates handler-provided health for a step
func (e *Engine) StepHealth(step *api.Step) (api.HealthState, error) {
	return e.steps.Health(step)
}

// Children returns the child step IDs a step expands into
func (e *Engine) Children(st *api.Step) ([]api.StepID, error) {
	return e.steps.Children(st)
}

// GetCompiledPredicate retrieves the compiled predicate for a flow step
func (e *Engine) GetCompiledPredicate(fs api.FlowStep) (any, error) {
	step, err := e.getStepFromPlan(fs)
	if err != nil {
		return nil, err
	}
	return e.scripts.Compile(step, step.Predicate)
}

func (e *Engine) getStepFromPlan(fs api.FlowStep) (*api.Step, error) {
	fl, err := e.GetFlowState(fs.FlowID)
	if err != nil {
		return nil, err
	}

	if step, ok := fl.Plan.Steps[fs.StepID]; ok {
		return step, nil
	}
	return nil, ErrStepNotInPlan
}
