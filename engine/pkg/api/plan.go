package api

import (
	"errors"
	"fmt"

	"github.com/kode4food/timebox"
)

type (
	// ExecutionPlan represents the compiled execution plan for a workflow
	ExecutionPlan struct {
		Goals      []timebox.ID             `json:"goals"`
		Required   []Name                   `json:"required"`
		Steps      map[timebox.ID]*StepInfo `json:"steps"`
		Attributes map[Name]*Dependencies   `json:"attributes"`
	}

	// StepInfo contains step metadata and compiled scripts
	StepInfo struct {
		Step      *Step `json:"step"`
		Script    any   `json:"-"`
		Predicate any   `json:"-"`
	}

	// Dependencies tracks which steps provide and consume an attribute
	Dependencies struct {
		Providers []timebox.ID `json:"providers"`
		Consumers []timebox.ID `json:"consumers"`
	}
)

var (
	ErrRequiredInput  = errors.New("required input not provided")
	ErrRequiredInputs = errors.New("required inputs not provided")
)

// GetStep retrieves a step from the plan by ID
func (ep *ExecutionPlan) GetStep(stepID timebox.ID) *Step {
	if info, ok := ep.Steps[stepID]; ok {
		return info.Step
	}
	return nil
}

// ValidateInputs checks that all required inputs are provided
func (ep *ExecutionPlan) ValidateInputs(args Args) error {
	var missing []Name

	for _, requiredInput := range ep.Required {
		if _, ok := args[requiredInput]; !ok {
			missing = append(missing, requiredInput)
		}
	}

	if len(missing) > 0 {
		if len(missing) == 1 {
			return fmt.Errorf("%s: '%s'", ErrRequiredInput, missing[0])
		}
		return fmt.Errorf("%w: %v", ErrRequiredInputs, missing)
	}

	return nil
}

// NeedsCompilation returns true if any scripts need compilation
func (ep *ExecutionPlan) NeedsCompilation() bool {
	for _, info := range ep.Steps {
		if ep.stepNeedsCompile(info) {
			return true
		}
		if ep.predNeedsCompile(info) {
			return true
		}
	}
	return false
}

func (ep *ExecutionPlan) stepNeedsCompile(info *StepInfo) bool {
	step := info.Step
	if step.Type != StepTypeScript || step.Script == nil {
		return false
	}
	return info.Script == nil
}

func (ep *ExecutionPlan) predNeedsCompile(info *StepInfo) bool {
	if info.Step.Predicate == nil {
		return false
	}
	return info.Predicate == nil
}
