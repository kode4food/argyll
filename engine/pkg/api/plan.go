package api

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

type (
	// ExecutionPlan represents the compiled execution plan for a flow
	ExecutionPlan struct {
		Goals      []StepID                  `json:"goals"`
		Required   []Name                    `json:"required"`
		Steps      Steps                     `json:"steps"`
		Children   map[StepID]*ExecutionPlan `json:"children,omitempty"`
		Attributes AttributeGraph            `json:"attributes"`
		Excluded   ExcludedSteps             `json:"excluded"`
	}

	// ExcludedSteps contains steps encountered during dependency traversal
	// that were not included in the plan. Only steps reachable from the goal
	// set appear here — unrelated catalog steps are not represented
	ExcludedSteps struct {
		// Satisfied maps step ID to output names that were already available
		// from init state, making the step's execution unnecessary
		Satisfied map[StepID][]Name `json:"satisfied,omitempty"`
		// Missing maps step ID to the required input names that could not be
		// satisfied, including alternative providers rejected in favor of a
		// satisfiable one
		Missing map[StepID][]Name `json:"missing,omitempty"`
	}

	// AttributeGraph is a dependency graph of attribute producers/consumers
	AttributeGraph map[Name]*AttributeEdges

	// AttributeEdges tracks which steps provide and consume an attribute
	AttributeEdges struct {
		Providers []StepID `json:"providers"`
		Consumers []StepID `json:"consumers"`
	}
)

var (
	ErrRequiredInputs = errors.New("required inputs not provided")
)

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

// AddStep adds a step's contributions to the graph
func (g AttributeGraph) AddStep(step *Step) AttributeGraph {
	res := maps.Clone(g)

	for name, attr := range step.Attributes {
		edges, ok := res[name]
		if !ok {
			edges = &AttributeEdges{
				Providers: []StepID{},
				Consumers: []StepID{},
			}
		}

		if attr.IsOutput() {
			edges = edges.addProvider(step.ID)
		}

		if attr.IsInput() {
			edges = edges.addConsumer(step.ID)
		}

		res[name] = edges
	}

	return res
}

// RemoveStep removes a step's contributions from the graph
func (g AttributeGraph) RemoveStep(step *Step) AttributeGraph {
	res := maps.Clone(g)

	for name, attr := range step.Attributes {
		if _, ok := g[name]; !ok {
			continue
		}

		edges := res[name]

		if attr.IsOutput() {
			edges = edges.removeProvider(step.ID)
		}

		if attr.IsInput() {
			edges = edges.removeConsumer(step.ID)
		}

		if edges.isEmpty() {
			delete(res, name)
		} else {
			res[name] = edges
		}
	}

	return res
}

func (e *AttributeEdges) addProvider(stepID StepID) *AttributeEdges {
	if slices.Contains(e.Providers, stepID) {
		return e
	}

	return &AttributeEdges{
		Providers: append(slices.Clone(e.Providers), stepID),
		Consumers: e.Consumers,
	}
}

func (e *AttributeEdges) addConsumer(stepID StepID) *AttributeEdges {
	if slices.Contains(e.Consumers, stepID) {
		return e
	}

	return &AttributeEdges{
		Providers: e.Providers,
		Consumers: append(slices.Clone(e.Consumers), stepID),
	}
}

func (e *AttributeEdges) removeProvider(stepID StepID) *AttributeEdges {
	if !slices.Contains(e.Providers, stepID) {
		return e
	}

	return &AttributeEdges{
		Providers: slices.DeleteFunc(
			slices.Clone(e.Providers),
			func(id StepID) bool { return id == stepID },
		),
		Consumers: e.Consumers,
	}
}

func (e *AttributeEdges) removeConsumer(stepID StepID) *AttributeEdges {
	if !slices.Contains(e.Consumers, stepID) {
		return e
	}

	return &AttributeEdges{
		Providers: e.Providers,
		Consumers: slices.DeleteFunc(
			slices.Clone(e.Consumers),
			func(id StepID) bool { return id == stepID },
		),
	}
}

func (e *AttributeEdges) isEmpty() bool {
	return len(e.Providers) == 0 && len(e.Consumers) == 0
}
