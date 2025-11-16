package engine

import (
	"errors"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/internal/util"
	"github.com/kode4food/spuds/engine/pkg/api"
)

type planBuilder struct {
	engState   *api.EngineState
	visited    util.Set[timebox.ID]
	available  util.Set[api.Name]
	missing    util.Set[api.Name]
	steps      map[timebox.ID]*api.StepInfo
	attributes map[api.Name]*api.Dependencies
}

var (
	ErrNoGoals = errors.New("at least one goal step is required")
)

func (e *Engine) CreateExecutionPlan(
	engState *api.EngineState, goalIDs []timebox.ID, initState api.Args,
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
		visited:    util.Set[timebox.ID]{},
		available:  util.Set[api.Name]{},
		missing:    util.Set[api.Name]{},
		steps:      map[timebox.ID]*api.StepInfo{},
		attributes: map[api.Name]*api.Dependencies{},
	}

	for key := range initState {
		pb.available.Add(key)
	}

	return pb
}

func (pb *planBuilder) allOutputsAvailable(step *api.Step) bool {
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
		if attr.IsOutput() && !pb.available.Contains(name) {
			return false
		}
	}
	return true
}

func (pb *planBuilder) findProvider(name api.Name) (timebox.ID, bool) {
	for candidateID, candidate := range pb.engState.Steps {
		if pb.stepProvidesOutput(candidate, name) {
			return candidateID, true
		}
	}
	return "", false
}

func (pb *planBuilder) stepProvidesOutput(step *api.Step, name api.Name) bool {
	if attr, ok := step.Attributes[name]; ok {
		return attr.IsOutput()
	}
	return false
}

func (pb *planBuilder) addStepToPlan(stepID timebox.ID, step *api.Step) {
	pb.visited.Add(stepID)

	pb.steps[stepID] = &api.StepInfo{
		Step: step,
	}

	for name, attr := range step.Attributes {
		if attr.IsOutput() {
			pb.available.Add(name)
		}
	}
}

func (pb *planBuilder) buildRequiredSet(step *api.Step) util.Set[api.Name] {
	requiredSet := util.Set[api.Name]{}
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			requiredSet.Add(name)
		}
	}
	return requiredSet
}

func (pb *planBuilder) resolveDependencies(step *api.Step) error {
	allInputs := step.GetAllInputArgs()
	requiredSet := pb.buildRequiredSet(step)

	for _, name := range allInputs {
		if pb.available.Contains(name) {
			pb.trackConsumer(name, step.ID)
			continue
		}
		if err := pb.resolveInput(name, step.ID, requiredSet); err != nil {
			return err
		}
	}

	return nil
}

func (pb *planBuilder) resolveInput(
	name api.Name, consumerID timebox.ID, requiredSet util.Set[api.Name],
) error {
	providerID, found := pb.findProvider(name)
	if !found {
		if requiredSet.Contains(name) {
			pb.missing.Add(name)
		}
		return nil
	}

	pb.trackDependency(name, providerID, consumerID)

	if err := pb.buildPlan(providerID); err != nil {
		return err
	}
	pb.available.Add(name)
	return nil
}

func (pb *planBuilder) buildPlan(stepID timebox.ID) error {
	if pb.visited.Contains(stepID) {
		return nil
	}

	step, ok := pb.engState.Steps[stepID]
	if !ok {
		return ErrStepNotFound
	}

	if pb.allOutputsAvailable(step) {
		pb.visited.Add(stepID)
		return nil
	}

	if err := pb.resolveDependencies(step); err != nil {
		return err
	}

	if !pb.visited.Contains(stepID) {
		pb.addStepToPlan(stepID, step)
	}

	return nil
}

func (pb *planBuilder) trackConsumer(name api.Name, consumerID timebox.ID) {
	if pb.attributes[name] == nil {
		pb.attributes[name] = &api.Dependencies{
			Providers: []timebox.ID{},
			Consumers: []timebox.ID{},
		}
	}
	pb.attributes[name].Consumers = append(pb.attributes[name].Consumers, consumerID)
}

func (pb *planBuilder) trackDependency(
	name api.Name, providerID, consumerID timebox.ID,
) {
	if pb.attributes[name] == nil {
		pb.attributes[name] = &api.Dependencies{
			Providers: []timebox.ID{},
			Consumers: []timebox.ID{},
		}
	}

	pb.attributes[name].Providers = append(pb.attributes[name].Providers, providerID)
	pb.attributes[name].Consumers = append(pb.attributes[name].Consumers, consumerID)
}

func (pb *planBuilder) getRequiredInputs() []api.Name {
	var requiredInputs []api.Name
	for input := range pb.missing {
		requiredInputs = append(requiredInputs, input)
	}
	return requiredInputs
}

func validateGoals(engState *api.EngineState, goalIDs []timebox.ID) error {
	for _, goalID := range goalIDs {
		if _, ok := engState.Steps[goalID]; !ok {
			return ErrStepNotFound
		}
	}
	return nil
}
