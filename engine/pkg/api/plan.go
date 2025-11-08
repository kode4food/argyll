package api

import (
	"errors"
	"fmt"

	"github.com/kode4food/timebox"
)

type ExecutionPlan struct {
	Scripts        map[timebox.ID]any `json:"-"`
	Predicates     map[timebox.ID]any `json:"-"`
	GoalSteps      []timebox.ID       `json:"goal_steps"`
	RequiredInputs []Name             `json:"required_inputs"`
	Steps          []*Step            `json:"steps"`
}

var (
	ErrRequiredInput  = errors.New("required input not provided")
	ErrRequiredInputs = errors.New("required inputs not provided")
)

func (ep *ExecutionPlan) GetStep(stepID timebox.ID) *Step {
	for _, step := range ep.Steps {
		if step.ID == stepID {
			return step
		}
	}
	return nil
}

func (ep *ExecutionPlan) ValidateInputs(args Args) error {
	var missing []Name

	for _, requiredInput := range ep.RequiredInputs {
		if _, ok := args[requiredInput]; !ok {
			missing = append(missing, requiredInput)
		}
	}

	if len(missing) > 0 {
		if len(missing) == 1 {
			return fmt.Errorf("%s: '%s'", ErrRequiredInput, missing[0])
		}
		return fmt.Errorf("%s: %v", ErrRequiredInputs, missing)
	}

	return nil
}

func (ep *ExecutionPlan) NeedsCompilation() bool {
	for _, step := range ep.Steps {
		if ep.stepNeedsCompile(step) {
			return true
		}
		if ep.predNeedsCompile(step) {
			return true
		}
	}
	return false
}

func (ep *ExecutionPlan) stepNeedsCompile(step *Step) bool {
	if step.Type != StepTypeScript || step.Script == nil {
		return false
	}
	if ep.Scripts == nil {
		return true
	}
	_, comp := ep.Scripts[step.ID]
	return !comp
}

func (ep *ExecutionPlan) predNeedsCompile(step *Step) bool {
	if step.Predicate == nil {
		return false
	}
	if ep.Predicates == nil {
		return true
	}
	_, comp := ep.Predicates[step.ID]
	return !comp
}
