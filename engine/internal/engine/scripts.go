package engine

import (
	"errors"

	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	ErrStepNotInPlan = errors.New("step not in execution plan")
)

// CreatePlan builds an execution plan using the engine's shared script registry
func (e *Engine) CreatePlan(
	cat api.CatalogState, goals []api.StepID, init api.InitArgs,
) (*api.ExecutionPlan, error) {
	return plan.Create(e.Matcher, cat, goals, init)
}

// PreviewPlan builds a preview plan using the engine's shared script registry
func (e *Engine) PreviewPlan(
	cat api.CatalogState, goals []api.StepID, init api.InitArgs,
) (*api.ExecutionPlan, error) {
	return plan.Preview(e.Matcher, cat, goals, init)
}

// VerifyScript compiles a step's script to check it is valid on this node
func (e *Engine) VerifyScript(step *api.Step) error {
	_, err := e.scripts.Compile(step, step.Script)
	return err
}

// GetCompiledPredicate retrieves the compiled predicate for a flow step
func (e *Engine) GetCompiledPredicate(fs api.FlowStep) (any, error) {
	step, err := e.getStepFromPlan(fs)
	if err != nil {
		return nil, err
	}
	return e.scripts.Compile(step, step.Predicate)
}

// GetCompiledScript retrieves the compiled script for a step in a flow
func (e *Engine) GetCompiledScript(fs api.FlowStep) (any, error) {
	step, err := e.getStepFromPlan(fs)
	if err != nil {
		return nil, err
	}
	return e.scripts.Compile(step, step.Script)
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
