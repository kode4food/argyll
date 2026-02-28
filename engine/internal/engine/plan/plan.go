package plan

import (
	"errors"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type builder struct {
	cat         *api.CatalogState
	satisfied   util.Set[api.Name]
	available   util.Set[api.Name]
	satisfiable util.Set[api.StepID]
	visited     util.Set[api.StepID]
	included    util.Set[api.StepID]
	missing     util.Set[api.Name]
	steps       api.Steps
	attributes  api.AttributeGraph
}

var (
	ErrNoGoals      = errors.New("at least one goal step is required")
	ErrStepNotFound = errors.New("step not found")
)

// Create builds an execution plan for the given goal steps, resolving
// dependencies and determining required inputs
func Create(
	cat *api.CatalogState, goalIDs []api.StepID, init api.Args,
) (*api.ExecutionPlan, error) {
	if len(goalIDs) == 0 {
		return nil, ErrNoGoals
	}

	if err := validateGoals(cat, goalIDs); err != nil {
		return nil, err
	}

	pb := newPlanBuilder(cat, init)
	pb.computeSatisfiable()
	if err := pb.collectSteps(goalIDs); err != nil {
		return nil, err
	}
	pb.buildPlan()
	excluded := pb.buildExcluded()

	return &api.ExecutionPlan{
		Goals:      goalIDs,
		Required:   pb.getRequiredInputs(),
		Steps:      pb.steps,
		Attributes: pb.attributes,
		Excluded:   excluded,
	}, nil
}

func newPlanBuilder(st *api.CatalogState, init api.Args) *builder {
	pb := &builder{
		cat:         st,
		satisfied:   util.Set[api.Name]{},
		available:   util.Set[api.Name]{},
		satisfiable: util.Set[api.StepID]{},
		visited:     util.Set[api.StepID]{},
		included:    util.Set[api.StepID]{},
		missing:     util.Set[api.Name]{},
		steps:       api.Steps{},
		attributes:  api.AttributeGraph{},
	}

	for key := range init {
		pb.satisfied.Add(key)
	}

	return pb
}

// Pass 1: determine which steps are satisfiable from init + other satisfiable
// outputs
func (b *builder) computeSatisfiable() {
	for name := range b.satisfied {
		b.available.Add(name)
	}

	progress := true
	for progress {
		progress = false
		for _, step := range b.cat.Steps {
			if b.satisfiable.Contains(step.ID) {
				continue
			}
			if !b.requiredInputsAvailable(step, b.available) {
				continue
			}

			b.satisfiable.Add(step.ID)
			progress = true
			for name, attr := range step.Attributes {
				if attr.IsOutput() && !b.available.Contains(name) {
					b.available.Add(name)
				}
			}
		}
	}
}

func (b *builder) requiredInputsAvailable(
	step *api.Step, available util.Set[api.Name],
) bool {
	for name, attr := range step.Attributes {
		if attr.IsRequired() && !available.Contains(name) {
			return false
		}
	}
	return true
}

// Pass 2: collect steps and build plan from goal traversal
func (b *builder) collectSteps(goalIDs []api.StepID) error {
	for _, goalID := range goalIDs {
		if err := b.collectStep(goalID); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) collectStep(stepID api.StepID) error {
	if b.visited.Contains(stepID) {
		return nil
	}
	b.visited.Add(stepID)

	step := b.cat.Steps[stepID]
	allInputs := step.GetAllInputArgs()
	required := b.buildRequired(step)
	for _, name := range allInputs {
		if b.satisfied.Contains(name) {
			b.markSatisfied(name)
			continue
		}
		hasProvider, err := b.includeProviders(name)
		if err != nil {
			return err
		}
		if required.Contains(name) && !hasProvider {
			b.missing.Add(name)
		}
	}

	if b.shouldInclude(step) {
		b.included.Add(stepID)
	}

	return nil
}

func (b *builder) buildRequired(step *api.Step) util.Set[api.Name] {
	required := util.Set[api.Name]{}
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			required.Add(name)
		}
	}
	return required
}

func (b *builder) markSatisfied(name api.Name) {
	for _, providerID := range b.findProviders(name) {
		step := b.cat.Steps[providerID]
		if b.outputsAvailable(step) {
			b.visited.Add(providerID)
		}
	}
}

func (b *builder) includeProviders(name api.Name) (bool, error) {
	providers := b.findProviders(name)
	if len(providers) == 0 {
		return false, nil
	}

	hasProvider := false
	for _, providerID := range providers {
		if !b.satisfiable.Contains(providerID) {
			continue
		}
		hasProvider = true
		if err := b.collectStep(providerID); err != nil {
			return false, err
		}
	}

	return hasProvider, nil
}

func (b *builder) findProviders(name api.Name) []api.StepID {
	if deps, ok := b.cat.Attributes[name]; ok {
		return deps.Providers
	}
	return nil
}

func (b *builder) shouldInclude(step *api.Step) bool {
	return !b.outputsAvailable(step)
}

func (b *builder) outputsAvailable(step *api.Step) bool {
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

func (b *builder) buildPlan() {
	for id := range b.included {
		step := b.cat.Steps[id]
		b.steps[id] = step
		for name, attr := range step.Attributes {
			if attr.IsOutput() {
				b.addProvider(name, id)
			}
		}
	}

	for id := range b.included {
		step := b.cat.Steps[id]
		for name, attr := range step.Attributes {
			if attr.IsInput() && b.inputSatisfied(name) {
				b.addConsumer(name, id)
			}
		}
	}
}

func (b *builder) inputSatisfied(name api.Name) bool {
	if b.satisfied.Contains(name) {
		return true
	}
	if deps, ok := b.cat.Attributes[name]; ok {
		if slices.ContainsFunc(deps.Providers, b.included.Contains) {
			return true
		}
	}
	return false
}

func (b *builder) missingRequired(step *api.Step) []api.Name {
	var missing []api.Name
	for name, attr := range step.Attributes {
		if attr.IsRequired() && !b.available.Contains(name) {
			missing = append(missing, name)
		}
	}
	return missing
}

func (b *builder) stepOutputNames(step *api.Step) []api.Name {
	var outputs []api.Name
	for name, attr := range step.Attributes {
		if attr.IsOutput() {
			outputs = append(outputs, name)
		}
	}
	return outputs
}

func (b *builder) buildExcluded() api.ExcludedSteps {
	excluded := api.ExcludedSteps{
		Satisfied: map[api.StepID][]api.Name{},
		Missing:   map[api.StepID][]api.Name{},
	}
	for stepID := range b.visited {
		step := b.cat.Steps[stepID]
		if b.included.Contains(stepID) {
			continue
		}
		if b.outputsAvailable(step) {
			excluded.Satisfied[stepID] = b.stepOutputNames(step)
			continue
		}
		missing := b.missingRequired(step)
		if len(missing) > 0 {
			excluded.Missing[stepID] = missing
		}
	}
	return excluded
}

func (b *builder) addProvider(name api.Name, provider api.StepID) {
	edges, ok := b.attributes[name]
	if !ok {
		edges = &api.AttributeEdges{}
	}
	edges.Providers = append(edges.Providers, provider)
	b.attributes[name] = edges
}

func (b *builder) addConsumer(name api.Name, consumer api.StepID) {
	edges, ok := b.attributes[name]
	if !ok {
		edges = &api.AttributeEdges{}
	}
	edges.Consumers = append(edges.Consumers, consumer)
	b.attributes[name] = edges
}

func (b *builder) getRequiredInputs() []api.Name {
	var required []api.Name
	for input := range b.missing {
		required = append(required, input)
	}
	return required
}

func validateGoals(cat *api.CatalogState, goalIDs []api.StepID) error {
	for _, goalID := range goalIDs {
		if _, ok := cat.Steps[goalID]; !ok {
			return ErrStepNotFound
		}
	}
	return nil
}
