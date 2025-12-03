package api

import (
	"errors"
	"fmt"
	"slices"
)

type (
	// ExecutionPlan represents the compiled execution plan for a flow
	ExecutionPlan struct {
		Goals      []StepID       `json:"goals"`
		Required   []Name         `json:"required"`
		Steps      Steps          `json:"steps"`
		Attributes AttributeGraph `json:"attributes"`
	}

	// AttributeGraph is a dependency graph of attribute producers/consumers
	AttributeGraph map[Name]*AttributeEdges

	// AttributeEdges tracks which steps provide and consume an attribute
	AttributeEdges struct {
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

// RemoveStep removes a step's contributions from the graph
func (g AttributeGraph) RemoveStep(stepID StepID, step *Step) {
	for name, attr := range step.Attributes {
		existing, ok := g[name]
		if !ok {
			continue
		}

		updated := &AttributeEdges{
			Providers: existing.Providers,
			Consumers: existing.Consumers,
		}

		if attr.IsOutput() {
			updated.Providers = slices.DeleteFunc(
				slices.Clone(existing.Providers),
				func(id StepID) bool { return id == stepID },
			)
		}

		if attr.IsInput() {
			updated.Consumers = slices.DeleteFunc(
				slices.Clone(existing.Consumers),
				func(id StepID) bool { return id == stepID },
			)
		}

		if len(updated.Providers) == 0 && len(updated.Consumers) == 0 {
			delete(g, name)
		} else {
			g[name] = updated
		}
	}
}

// AddStep adds a step's contributions to the graph
func (g AttributeGraph) AddStep(stepID StepID, step *Step) {
	for name, attr := range step.Attributes {
		existing := g[name]

		var updated *AttributeEdges
		if existing == nil {
			updated = &AttributeEdges{
				Providers: []StepID{},
				Consumers: []StepID{},
			}
		} else {
			updated = &AttributeEdges{
				Providers: slices.Clone(existing.Providers),
				Consumers: slices.Clone(existing.Consumers),
			}
		}

		if attr.IsOutput() {
			updated.Providers = append(updated.Providers, stepID)
		}

		if attr.IsInput() {
			updated.Consumers = append(updated.Consumers, stepID)
		}

		g[name] = updated
	}
}
