package engine

import (
	"errors"

	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	ErrStepNotInPlan = errors.New("step not in execution plan")
)

// VerifyScript compiles a step's script to check it is valid on this node
func (e *Engine) VerifyScript(step *api.Step) error {
	_, err := e.scripts.Compile(step, step.Script)
	return err
}

// GetCompiledPredicate retrieves the compiled predicate for a flow step.
func (e *Engine) GetCompiledPredicate(fs api.FlowStep) (any, error) {
	step, err := e.getStepFromPlan(fs)
	if err != nil {
		return nil, err
	}
	return e.scripts.Compile(step, step.Predicate)
}

// GetCompiledScript retrieves the compiled script for a step in a flow.
func (e *Engine) GetCompiledScript(fs api.FlowStep) (any, error) {
	step, err := e.getStepFromPlan(fs)
	if err != nil {
		return nil, err
	}
	return e.scripts.Compile(step, step.Script)
}

func (e *Engine) getStepFromPlan(fs api.FlowStep) (*api.Step, error) {
	flow, err := e.GetFlowState(fs.FlowID)
	if err != nil {
		return nil, err
	}

	if step, ok := flow.Plan.Steps[fs.StepID]; ok {
		return step, nil
	}
	return nil, ErrStepNotInPlan
}
