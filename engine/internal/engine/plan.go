package engine

import (
	"errors"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

type planBuilder struct {
	engState   *api.EngineState
	visited    util.Set[api.StepID]
	available  util.Set[api.Name]
	missing    util.Set[api.Name]
	steps      map[api.StepID]*api.StepInfo
	attributes map[api.Name]*api.Dependencies
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

	for _, goalID := range goalIDs {
		if err := pb.buildPlan(goalID); err != nil {
			return nil, err
		}
	}

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
		available:  util.Set[api.Name]{},
		missing:    util.Set[api.Name]{},
		steps:      map[api.StepID]*api.StepInfo{},
		attributes: map[api.Name]*api.Dependencies{},
	}

	for key := range initState {
		pb.available.Add(key)
	}

	return pb
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
		if attr.IsOutput() && !b.available.Contains(name) {
			return false
		}
	}
	return true
}

func (b *planBuilder) findProvider(name api.Name) (api.StepID, bool) {
	for candidateID, candidate := range b.engState.Steps {
		if b.stepProvidesOutput(candidate, name) {
			return candidateID, true
		}
	}
	return "", false
}

func (b *planBuilder) stepProvidesOutput(step *api.Step, name api.Name) bool {
	if attr, ok := step.Attributes[name]; ok {
		return attr.IsOutput()
	}
	return false
}

func (b *planBuilder) addStepToPlan(stepID api.StepID, step *api.Step) {
	b.visited.Add(stepID)

	b.steps[stepID] = &api.StepInfo{
		Step: step,
	}

	for name, attr := range step.Attributes {
		if attr.IsOutput() {
			b.available.Add(name)
		}
	}
}

func (b *planBuilder) buildRequiredSet(step *api.Step) util.Set[api.Name] {
	requiredSet := util.Set[api.Name]{}
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			requiredSet.Add(name)
		}
	}
	return requiredSet
}

func (b *planBuilder) resolveDependencies(step *api.Step) error {
	allInputs := step.GetAllInputArgs()
	requiredSet := b.buildRequiredSet(step)

	for _, name := range allInputs {
		if b.available.Contains(name) {
			b.trackConsumer(name, step.ID)
			continue
		}
		if err := b.resolveInput(name, step.ID, requiredSet); err != nil {
			return err
		}
	}

	return nil
}

func (b *planBuilder) resolveInput(
	name api.Name, consumerID api.StepID, requiredSet util.Set[api.Name],
) error {
	providerID, found := b.findProvider(name)
	if !found {
		if requiredSet.Contains(name) {
			b.missing.Add(name)
		}
		return nil
	}

	b.trackDependency(name, providerID, consumerID)

	if err := b.buildPlan(providerID); err != nil {
		return err
	}
	b.available.Add(name)
	return nil
}

func (b *planBuilder) buildPlan(stepID api.StepID) error {
	if b.visited.Contains(stepID) {
		return nil
	}

	step, ok := b.engState.Steps[stepID]
	if !ok {
		return ErrStepNotFound
	}

	if b.allOutputsAvailable(step) {
		b.visited.Add(stepID)
		return nil
	}

	if err := b.resolveDependencies(step); err != nil {
		return err
	}

	if !b.visited.Contains(stepID) {
		b.addStepToPlan(stepID, step)
	}

	return nil
}

func (b *planBuilder) trackConsumer(name api.Name, consumerID api.StepID) {
	if b.attributes[name] == nil {
		b.attributes[name] = &api.Dependencies{
			Providers: []api.StepID{},
			Consumers: []api.StepID{},
		}
	}
	b.attributes[name].Consumers = append(
		b.attributes[name].Consumers, consumerID,
	)
}

func (b *planBuilder) trackDependency(
	name api.Name, providerID, consumerID api.StepID,
) {
	if b.attributes[name] == nil {
		b.attributes[name] = &api.Dependencies{
			Providers: []api.StepID{},
			Consumers: []api.StepID{},
		}
	}

	b.attributes[name].Providers = append(
		b.attributes[name].Providers, providerID,
	)
	b.attributes[name].Consumers = append(
		b.attributes[name].Consumers, consumerID,
	)
}

func (b *planBuilder) getRequiredInputs() []api.Name {
	var requiredInputs []api.Name
	for input := range b.missing {
		requiredInputs = append(requiredInputs, input)
	}
	return requiredInputs
}

func validateGoals(engState *api.EngineState, goalIDs []api.StepID) error {
	for _, goalID := range goalIDs {
		if _, ok := engState.Steps[goalID]; !ok {
			return ErrStepNotFound
		}
	}
	return nil
}
