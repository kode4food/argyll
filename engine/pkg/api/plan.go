package api

import (
	"errors"
	"fmt"
)

type (
	// ExecutionPlan represents the compiled execution plan for a flow
	ExecutionPlan struct {
		Goals      []StepID               `json:"goals"`
		Required   []Name                 `json:"required"`
		Steps      Steps                  `json:"steps"`
		Attributes map[Name]*Dependencies `json:"attributes"`
	}

	// Dependencies tracks which steps provide and consume an attribute
	Dependencies struct {
		Providers []StepID `json:"providers"`
		Consumers []StepID `json:"consumers"`
	}
)

var (
	ErrRequiredInput  = errors.New("required input not provided")
	ErrRequiredInputs = errors.New("required inputs not provided")
)

// ValidateInputs checks that all required inputs are provided
func (p *ExecutionPlan) ValidateInputs(args Args) error {
	var missing []Name

	for _, requiredInput := range p.Required {
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
