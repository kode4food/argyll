package engine

import (
	"errors"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type planBuilder struct {
	engState   *api.EngineState
	visited    util.Set[api.StepID]
	included   util.Set[api.StepID]
	satisfied  util.Set[api.Name]
	missing    util.Set[api.Name]
	steps      api.Steps
	attributes api.AttributeGraph
}

var (
	ErrNoGoals = errors.New("at least one goal step is required")
)

// CreateExecutionPlan builds an execution plan for the given goal steps,
// resolving dependencies and determining required inputs
func (e *Engine) CreateExecutionPlan(
	engState *api.EngineState, goalIDs []api.StepID, initState api.Args,
) (*api.ExecutionPlan, error) {
	if len(goalIDs) == 0 {
		return nil, ErrNoGoals
	}

	if err := validateGoals(engState, goalIDs); err != nil {
		return nil, err
	}

	pb := newPlanBuilder(engState, initState)
	if err := pb.collectSteps(goalIDs); err != nil {
		return nil, err
	}
	pb.buildPlan()

	return &api.ExecutionPlan{
		Goals:      goalIDs,
		Required:   pb.getRequiredInputs(),
		Steps:      pb.steps,
		Attributes: pb.attributes,
	}, nil
}

func newPlanBuilder(st *api.EngineState, initState api.Args) *planBuilder {
	pb := &planBuilder{
		engState:   st,
		visited:    util.Set[api.StepID]{},
		included:   util.Set[api.StepID]{},
		satisfied:  util.Set[api.Name]{},
		missing:    util.Set[api.Name]{},
		steps:      api.Steps{},
		attributes: api.AttributeGraph{},
	}

	for key := range initState {
		pb.satisfied.Add(key)
	}

	return pb
}

func validateGoals(engState *api.EngineState, goalIDs []api.StepID) error {
	for _, goalID := range goalIDs {
		if _, ok := engState.Steps[goalID]; !ok {
			return ErrStepNotFound
		}
	}
	return nil
}

func (b *planBuilder) collectSteps(goalIDs []api.StepID) error {
	for _, goalID := range goalIDs {
		if err := b.collectStep(goalID); err != nil {
			return err
		}
	}
	return nil
}

func (b *planBuilder) collectStep(stepID api.StepID) error {
	if b.visited.Contains(stepID) {
		return nil
	}
	b.visited.Add(stepID)

	step, ok := b.engState.Steps[stepID]
	if !ok {
		return ErrStepNotFound
	}

	if err := b.resolveDependencies(step); err != nil {
		return err
	}

	if b.allOutputsAvailable(step) {
		return nil
	}

	b.included.Add(stepID)
	for name, attr := range step.Attributes {
		if attr.IsOutput() {
			b.satisfied.Add(name)
		}
	}
	return nil
}

func (b *planBuilder) buildPlan() {
	for id := range b.included {
		step := b.engState.Steps[id]
		b.steps[id] = step
		for name, attr := range step.Attributes {
			if attr.IsOutput() {
				b.addProvider(name, id)
			}
			if attr.IsInput() && b.satisfied.Contains(name) {
				b.addConsumer(name, id)
			}
		}
	}
}

func (b *planBuilder) resolveDependencies(step *api.Step) error {
	allInputs := step.GetAllInputArgs()
	required := b.buildRequired(step)

	for _, name := range allInputs {
		if b.satisfied.Contains(name) {
			continue
		}
		if err := b.resolveInput(name, required); err != nil {
			return err
		}
	}

	return nil
}

func (b *planBuilder) buildRequired(step *api.Step) util.Set[api.Name] {
	required := util.Set[api.Name]{}
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			required.Add(name)
		}
	}
	return required
}

func (b *planBuilder) resolveInput(
	name api.Name, required util.Set[api.Name],
) error {
	providers := b.findProviders(name)
	if len(providers) == 0 {
		if required.Contains(name) {
			b.missing.Add(name)
		}
		return nil
	}

	for _, providerID := range providers {
		if err := b.collectStep(providerID); err != nil {
			return err
		}
	}
	b.satisfied.Add(name)
	return nil
}

func (b *planBuilder) findProviders(name api.Name) []api.StepID {
	if deps, ok := b.engState.Attributes[name]; ok {
		return deps.Providers
	}
	return nil
}

func (b *planBuilder) allOutputsAvailable(step *api.Step) bool {
	hasOutputs := false
	for _, attr := range step.Attributes {
		if attr.IsOutput() {
			hasOutputs = true
			break
		}
	}
	if !hasOutputs {
		return false
	}

	for name, attr := range step.Attributes {
		if attr.IsOutput() && !b.satisfied.Contains(name) {
			return false
		}
	}
	return true
}

func (b *planBuilder) addProvider(name api.Name, provider api.StepID) {
	edges, ok := b.attributes[name]
	if !ok {
		edges = &api.AttributeEdges{}
	}
	edges.Providers = append(edges.Providers, provider)
	b.attributes[name] = edges
}

func (b *planBuilder) addConsumer(name api.Name, consumer api.StepID) {
	edges, ok := b.attributes[name]
	if !ok {
		edges = &api.AttributeEdges{}
	}
	edges.Consumers = append(edges.Consumers, consumer)
	b.attributes[name] = edges
}

func (b *planBuilder) getRequiredInputs() []api.Name {
	var required []api.Name
	for input := range b.missing {
		required = append(required, input)
	}
	return required
}
