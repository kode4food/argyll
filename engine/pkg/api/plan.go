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

var ErrRequiredInputs = errors.New("required inputs not provided")

// ValidateInputs checks that all required inputs are provided
func (p *ExecutionPlan) ValidateInputs(args Args) error {
	var missing []Name

	for _, required := range p.Required {
		if _, ok := args[required]; !ok {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %v", ErrRequiredInputs, missing)
	}

	return nil
}

// BuildDependencies constructs a dependency graph from all step definitions,
// tracking which steps provide and consume each attribute
func BuildDependencies(steps Steps) map[Name]*Dependencies {
	deps := make(map[Name]*Dependencies)

	for stepID, step := range steps {
		for name, attr := range step.Attributes {
			if deps[name] == nil {
				deps[name] = &Dependencies{
					Providers: []StepID{},
					Consumers: []StepID{},
				}
			}

			if attr.IsOutput() {
				deps[name].Providers = append(deps[name].Providers, stepID)
			}
			if attr.IsInput() {
				deps[name].Consumers = append(deps[name].Consumers, stepID)
			}
		}
	}

	return deps
}
