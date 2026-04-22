package plan

import (
	"errors"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	Planner func(
		api.CatalogState, []api.StepID, api.Args,
	) (*api.ExecutionPlan, error)

	builder struct {
		cat         api.CatalogState
		satisfied   util.Set[api.Name]
		available   util.Set[api.Name]
		satisfiable util.Set[api.StepID]
		visited     util.Set[api.StepID]
		included    util.Set[api.StepID]
		missing     util.Set[api.Name]
		steps       api.Steps
		attributes  api.AttributeGraph
		providers   selectProviders
	}

	selectProviders func(*builder, []api.StepID) []api.StepID
)

var (
	ErrNoGoals      = errors.New("at least one goal step is required")
	ErrStepNotFound = errors.New("step not found")
)

var (
	_ Planner = Create
	_ Planner = Preview
)

// Create builds an execution plan for the given goal steps, resolving
// dependencies and determining required inputs
func Create(
	cat api.CatalogState, goals []api.StepID, init api.Args,
) (*api.ExecutionPlan, error) {
	return create(cat, goals, init, strictProviders)
}

// Preview builds an execution plan for preview purposes. Unlike Create, it
// falls back to unsatisfied provider chains when no satisfiable provider
// exists so the UI can show the full dependency path back to missing init
// inputs
func Preview(
	cat api.CatalogState, goals []api.StepID, init api.Args,
) (*api.ExecutionPlan, error) {
	return create(cat, goals, init, previewProviders)
}

func create(
	cat api.CatalogState, goals []api.StepID, init api.Args,
	providers selectProviders,
) (*api.ExecutionPlan, error) {
	if len(goals) == 0 {
		return nil, ErrNoGoals
	}

	if err := validateGoals(cat, goals); err != nil {
		return nil, err
	}

	pb := newPlanBuilder(cat, init, providers)
	pb.computeSatisfiable()
	if err := pb.collectSteps(goals); err != nil {
		return nil, err
	}
	pb.buildPlan()
	excluded := pb.buildExcluded()

	res := &api.ExecutionPlan{
		Goals:      goals,
		Required:   pb.getRequiredInputs(),
		Steps:      pb.steps,
		Attributes: pb.attributes,
		Excluded:   excluded,
	}
	children, err := buildChildPlans(res, cat, providers)
	if err != nil {
		return nil, err
	}
	res.Children = children
	return res, nil
}

func newPlanBuilder(
	st api.CatalogState, init api.Args, providers selectProviders,
) *builder {
	pb := &builder{
		cat:         st,
		providers:   providers,
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
		for _, st := range b.cat.Steps {
			if b.satisfiable.Contains(st.ID) {
				continue
			}
			if !b.requiredInputsAvailable(st, b.available) {
				continue
			}

			b.satisfiable.Add(st.ID)
			progress = true
			for name, attr := range st.Attributes {
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
func (b *builder) collectSteps(goals []api.StepID) error {
	for _, goalID := range goals {
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

	st := b.cat.Steps[stepID]
	allInputs := st.GetAllInputArgs()
	required := b.buildRequired(st)
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

	if b.shouldInclude(st) {
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
		st := b.cat.Steps[providerID]
		if b.outputsAvailable(st) {
			b.visited.Add(providerID)
		}
	}
}

func (b *builder) includeProviders(name api.Name) (bool, error) {
	providers := b.findProviders(name)
	if len(providers) == 0 {
		return false, nil
	}

	selected := b.providers(b, providers)

	// Mark rejected providers as visited so they appear in excluded
	for _, providerID := range providers {
		if !slices.Contains(selected, providerID) {
			b.visited.Add(providerID)
		}
	}

	if len(selected) == 0 {
		return false, nil
	}

	for _, providerID := range selected {
		if err := b.collectStep(providerID); err != nil {
			return false, err
		}
	}

	return true, nil
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
		st := b.cat.Steps[id]
		b.steps[id] = st
		for name, attr := range st.Attributes {
			if attr.IsOutput() {
				b.addProvider(name, id)
			}
		}
	}

	for id := range b.included {
		st := b.cat.Steps[id]
		for name, attr := range st.Attributes {
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
	for sid := range b.visited {
		st := b.cat.Steps[sid]
		if b.included.Contains(sid) {
			continue
		}
		if b.outputsAvailable(st) {
			excluded.Satisfied[sid] = b.stepOutputNames(st)
			continue
		}
		missing := b.missingRequired(st)
		if len(missing) > 0 {
			excluded.Missing[sid] = missing
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

func validateGoals(cat api.CatalogState, goals []api.StepID) error {
	for _, goalID := range goals {
		if _, ok := cat.Steps[goalID]; !ok {
			return ErrStepNotFound
		}
	}
	return nil
}

func strictProviders(b *builder, providers []api.StepID) []api.StepID {
	var res []api.StepID
	for _, id := range providers {
		if b.satisfiable.Contains(id) {
			res = append(res, id)
		}
	}
	return res
}

func previewProviders(b *builder, providers []api.StepID) []api.StepID {
	if res := strictProviders(b, providers); len(res) > 0 {
		return res
	}
	return providers
}

func buildChildPlans(
	pl *api.ExecutionPlan, cat api.CatalogState, providers selectProviders,
) (map[api.StepID]*api.ExecutionPlan, error) {
	childPlans := map[api.StepID]*api.ExecutionPlan{}
	for sid, st := range pl.Steps {
		if st.Type != api.StepTypeFlow || st.Flow == nil {
			continue
		}
		childPlan, err := create(
			cat, st.Flow.Goals, childPlanInit(st), providers,
		)
		if err != nil {
			return nil, err
		}
		childPlans[sid] = childPlan
	}
	if len(childPlans) == 0 {
		return nil, nil
	}
	return childPlans, nil
}

func childPlanInit(step *api.Step) api.Args {
	res := api.Args{}
	for name, attr := range step.Attributes {
		if !isGuaranteedInput(attr) {
			continue
		}
		mapped, _ := step.MappedName(name)
		res[mapped] = true
	}
	return res
}

func isGuaranteedInput(attr *api.AttributeSpec) bool {
	if attr.IsRequired() || attr.IsConst() {
		return true
	}
	return attr.IsOptional() && attr.Default != ""
}
