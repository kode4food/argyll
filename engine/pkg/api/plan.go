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
		Excluded   ExcludedSteps             `json:"excluded"`
		Steps      Steps                     `json:"steps"`
		Children   map[StepID]*ExecutionPlan `json:"children,omitempty"`
		Attributes AttributeGraph            `json:"attributes"`
		Goals      Goals                     `json:"goals"`
		Required   []Name                    `json:"required"`
	}

	// Goals is a chain of fallback goal sets. Every one of Steps must succeed,
	// and Else is attempted only when Steps becomes impossible
	Goals struct {
		Else  *Goals   `json:"else,omitempty"`
		Steps []StepID `json:"steps"`
	}

	// ExcludedSteps contains steps encountered during dependency traversal
	// that were not included in the plan. Only steps reachable from the goal
	// set appear here, so unrelated catalog steps are not represented
	ExcludedSteps struct {
		// Satisfied maps step ID to output names that were already available
		// from init state, making the step's execution unnecessary
		Satisfied map[StepID][]Name `json:"satisfied,omitempty"`

		// Blocked maps step ID to input names whose init values prevent the
		// step from ever satisfying its collection policy
		Blocked map[StepID][]Name `json:"blocked,omitempty"`

		// Missing maps step ID to the required input names that could not be
		// satisfied, including alternative providers rejected in favor of a
		// satisfiable one
		Missing map[StepID][]Name `json:"missing,omitempty"`
	}

	// AttributeGraph is a dependency graph of attribute producers/consumers
	AttributeGraph Map[Name, *AttributeEdges]

	// AttributeEdges tracks which steps provide and consume an attribute
	AttributeEdges struct {
		Providers Slice[StepID] `json:"providers"`
		Consumers Slice[StepID] `json:"consumers"`
	}
)

var (
	ErrRequiredInputs = errors.New("required inputs not provided")
)

// ValidateInputs checks that all required inputs are provided
func (p *ExecutionPlan) ValidateInputs(args InitArgs) error {
	var missing []Name

	for _, required := range p.Required {
		if len(args[required]) == 0 {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %v", ErrRequiredInputs, missing)
	}

	return nil
}

// Copy returns a deep copy of the goal chain and its step slices
func (g *Goals) Copy() *Goals {
	if g == nil {
		return nil
	}
	res := &Goals{}
	for src, dst := g, res; src != nil; src = src.Else {
		dst.Steps = slices.Clone(src.Steps)
		if src.Else != nil {
			dst.Else = &Goals{}
			dst = dst.Else
		}
	}
	return res
}

// Sets returns the chain's goal sets in the order they are attempted
func (g *Goals) Sets() [][]StepID {
	var res [][]StepID
	for next := g; next != nil; next = next.Else {
		res = append(res, next.Steps)
	}
	return res
}

// AllSteps returns every goal step in the chain once, in the order the chain
// names them
func (g *Goals) AllSteps() []StepID {
	var res []StepID
	for _, set := range g.Sets() {
		for _, sid := range set {
			if !slices.Contains(res, sid) {
				res = append(res, sid)
			}
		}
	}
	return res
}

// Valid reports whether every goal set in the chain names at least one step
func (g *Goals) Valid() bool {
	return len(g.Steps) > 0 && (g.Else == nil || g.Else.Valid())
}

// Equal reports whether two chains name the same goal sets in the same order
func (g *Goals) Equal(other *Goals) bool {
	return slices.EqualFunc(g.Sets(), other.Sets(), slices.Equal)
}

// AddStep adds a step's contributions to the graph
func (g AttributeGraph) AddStep(st *Step) AttributeGraph {
	res := maps.Clone(g)

	for name, attr := range st.Attributes {
		edges, ok := res[name]
		if !ok {
			edges = &AttributeEdges{
				Providers: []StepID{},
				Consumers: []StepID{},
			}
		}

		if attr.IsOutput() {
			edges = edges.addProvider(st.ID)
		}

		if attr.IsInput() {
			edges = edges.addConsumer(st.ID)
		}

		res[name] = edges
	}

	return res
}

// RemoveStep removes a step's contributions from the graph
func (g AttributeGraph) RemoveStep(st *Step) AttributeGraph {
	res := maps.Clone(g)

	for name, attr := range st.Attributes {
		if _, ok := g[name]; !ok {
			continue
		}

		edges := res[name]

		if attr.IsOutput() {
			edges = edges.removeProvider(st.ID)
		}

		if attr.IsInput() {
			edges = edges.removeConsumer(st.ID)
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
		Providers: e.Providers.Append(stepID),
		Consumers: e.Consumers,
	}
}

func (e *AttributeEdges) addConsumer(stepID StepID) *AttributeEdges {
	if slices.Contains(e.Consumers, stepID) {
		return e
	}

	return &AttributeEdges{
		Providers: e.Providers,
		Consumers: e.Consumers.Append(stepID),
	}
}

func (e *AttributeEdges) removeProvider(stepID StepID) *AttributeEdges {
	if !slices.Contains(e.Providers, stepID) {
		return e
	}

	return &AttributeEdges{
		Providers: e.Providers.Remove(
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
		Consumers: e.Consumers.Remove(
			func(id StepID) bool { return id == stepID },
		),
	}
}

func (e *AttributeEdges) isEmpty() bool {
	return len(e.Providers) == 0 && len(e.Consumers) == 0
}
