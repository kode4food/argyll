package plan

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// Planner builds an execution plan for the given matcher, children func,
	// catalog state, goals, and init args
	Planner func(
		policy.Matcher, ChildrenFunc, api.CatalogState, []api.StepID,
		api.InitArgs,
	) (*api.ExecutionPlan, error)

	// ChildrenFunc returns the child step IDs a step expands into
	ChildrenFunc func(*api.Step) ([]api.StepID, error)

	builder struct {
		satisfied   util.Set[api.Name]
		available   util.Set[api.Name]
		satisfiable util.Set[api.StepID]
		visited     util.Set[api.StepID]
		included    util.Set[api.StepID]
		needed      util.Set[api.StepID]
		missing     util.Set[api.Name]
		blocked     map[api.StepID][]api.Name
		steps       api.Steps
		attributes  api.AttributeGraph
		planArgs
	}

	planArgs struct {
		cat       api.CatalogState
		match     policy.Matcher
		providers selectProviders
		init      api.InitArgs
		goals     []api.StepID
	}

	selectProviders func(*builder, []api.StepID) []api.StepID
)

var (
	ErrNoGoals            = errors.New("at least one goal step is required")
	ErrStepNotFound       = errors.New("step not found")
	ErrCircularDependency = errors.New("circular dependency detected")
)

// Create builds an execution plan for the given goal steps, resolving
// dependencies and determining required inputs. If children is non-nil it is
// called for each step in the plan to discover child goals, and the result is
// built recursively and attached to ExecutionPlan.Children
func Create(
	match policy.Matcher, children ChildrenFunc, cat api.CatalogState,
	goals []api.StepID, init api.InitArgs,
) (*api.ExecutionPlan, error) {
	return create(planArgs{
		match:     match,
		cat:       cat,
		goals:     goals,
		providers: strictProviders,
		init:      init,
	}, children, util.Set[api.StepID]{})
}

// Preview builds an execution plan for preview purposes. Unlike Create, it
// falls back to unsatisfied provider chains when no satisfiable provider
// exists so the UI can show the full dependency path back to missing init
// inputs
func Preview(
	match policy.Matcher, children ChildrenFunc, cat api.CatalogState,
	goals []api.StepID, init api.InitArgs,
) (*api.ExecutionPlan, error) {
	return create(planArgs{
		match:     match,
		cat:       cat,
		goals:     goals,
		providers: previewProviders,
		init:      init,
	}, children, util.Set[api.StepID]{})
}

// ChildPlanInit derives init args for a child plan from a parent step
func ChildPlanInit(step *api.Step) api.InitArgs {
	res := api.InitArgs{}
	for name, attr := range step.Attributes {
		if !policy.StepInputGuaranteed(attr) {
			continue
		}
		mapped, _ := step.MappedName(name)
		res[mapped] = []any{true}
	}
	return res
}

func create(
	args planArgs, children ChildrenFunc, ancestors util.Set[api.StepID],
) (*api.ExecutionPlan, error) {
	pl, err := build(args)
	if err != nil || children == nil {
		return pl, err
	}
	for sid, st := range pl.Steps {
		childGoals, err := children(st)
		if err != nil {
			return nil, err
		}
		if len(childGoals) == 0 {
			continue
		}
		if ancestors.Contains(sid) {
			return nil, fmt.Errorf("%w: step %s", ErrCircularDependency, sid)
		}
		ancestors.Add(sid)
		childPlan, err := create(planArgs{
			match:     args.match,
			cat:       args.cat,
			goals:     childGoals,
			providers: args.providers,
			init:      ChildPlanInit(st),
		}, children, ancestors)
		ancestors.Remove(sid)
		if err != nil {
			return nil, err
		}
		if pl.Children == nil {
			pl.Children = map[api.StepID]*api.ExecutionPlan{}
		}
		pl.Children[sid] = childPlan
	}
	return pl, nil
}

func build(args planArgs) (*api.ExecutionPlan, error) {
	if len(args.goals) == 0 {
		return nil, ErrNoGoals
	}

	if err := validateGoals(args.cat, args.goals); err != nil {
		return nil, err
	}

	pb := newPlanBuilder(args)
	pb.computeSatisfiable()
	if err := pb.collectSteps(args.goals); err != nil {
		return nil, err
	}
	pb.buildPlan()
	excluded := pb.buildExcluded()

	return &api.ExecutionPlan{
		Goals:      args.goals,
		Required:   pb.getRequiredInputs(),
		Steps:      pb.steps,
		Attributes: pb.attributes,
		Excluded:   excluded,
	}, nil
}

func newPlanBuilder(args planArgs) *builder {
	pb := &builder{
		planArgs:    args,
		satisfied:   util.Set[api.Name]{},
		available:   util.Set[api.Name]{},
		satisfiable: util.Set[api.StepID]{},
		visited:     util.Set[api.StepID]{},
		included:    util.Set[api.StepID]{},
		needed:      util.Set[api.StepID]{},
		missing:     util.Set[api.Name]{},
		blocked:     map[api.StepID][]api.Name{},
		steps:       api.Steps{},
		attributes:  api.AttributeGraph{},
	}

	for key, values := range args.init {
		if len(values) > 0 {
			pb.satisfied.Add(key)
		}
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
			if len(b.blockedInputs(st)) > 0 {
				continue
			}
			if !policy.RequiredInputsAvailable(st, b.available.Contains) {
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
	if blocked := b.blockedInputs(st); len(blocked) > 0 {
		b.blocked[stepID] = blocked
		return nil
	}

	gate := b.stepGateStatus(st)
	gateClosed := policy.MatchAllowsStepSkip(gate)
	gateOpen := policy.MatchAllowsNormalDemand(gate)
	allInputs := st.GetAllInputArgs()
	required := buildRequired(st)
	for _, name := range allInputs {
		attr := st.Attributes[name]
		isGateAttr := policy.RequiredInputHasMatch(attr)
		if gateClosed && !isGateAttr {
			continue
		}
		if b.initSatisfiesInput(name, attr) {
			b.markSatisfied(name)
			continue
		}
		hasProvider, err := b.includeProviders(name, attr)
		if err != nil {
			return err
		}
		if required.Contains(name) && policy.RequiredInputMissing(
			attr.Collect(), hasProvider, b.initHasValues(name),
		) {
			if !gateOpen && (gateClosed || !isGateAttr) {
				continue
			}
			b.missing.Add(name)
		}
	}

	if b.shouldInclude(st) {
		b.included.Add(stepID)
	}

	return nil
}

func (b *builder) stepGateStatus(step *api.Step) policy.MatchStatus {
	status, err := policy.RequiredMatchStepStatus(policy.RequiredMatchStep{
		Step: step,
		Values: func(name api.Name) []*api.AttributeValue {
			return initAttributeValues(b.init[name])
		},
		Providers: func(name api.Name) policy.ProviderSummary {
			hasInit := b.initHasValues(name)
			hasProvider := len(b.findProviders(name)) > 0
			complete := policy.InitProviderComplete(hasInit, hasProvider)
			return policy.ProviderSummary{
				Terminal:     complete,
				AllSucceeded: complete,
			}
		},
		Match: b.match,
	})
	if err != nil {
		return policy.MatchUnknown
	}
	return status
}

func (b *builder) initSatisfiesInput(
	name api.Name, attr *api.AttributeSpec,
) bool {
	return policy.InitSatisfiesRequired(
		attr, b.initHasValues(name), len(b.findProviders(name)) > 0,
		initAttributeValues(b.init[name]), b.match,
	)
}

func (b *builder) blockedInputs(step *api.Step) []api.Name {
	var blocked []api.Name
	for name, attr := range step.Attributes {
		if policy.InitBlocksRuntime(attr, b.initHasValues(name)) {
			blocked = append(blocked, name)
		}
	}
	return blocked
}

func (b *builder) markSatisfied(name api.Name) {
	for _, providerID := range b.findProviders(name) {
		st := b.cat.Steps[providerID]
		if b.outputsAvailable(st) {
			b.visited.Add(providerID)
		}
	}
}

func (b *builder) initHasValues(name api.Name) bool {
	return len(b.init[name]) > 0
}

func (b *builder) includeProviders(
	name api.Name, attr *api.AttributeSpec,
) (bool, error) {
	providers := b.findProviders(name)
	if len(providers) == 0 {
		return false, nil
	}

	selected := b.providers(b, providers)
	if policy.RequiresAllProviders(attr.Collect()) &&
		len(selected) != len(providers) {
		return false, nil
	}

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
		b.needed.Add(providerID)
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
	if b.needed.Contains(step.ID) {
		return true
	}
	return !b.outputsAvailable(step)
}

func (b *builder) outputsAvailable(step *api.Step) bool {
	return policy.StepOutputsSatisfied(step, b.satisfied.Contains)
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

func (b *builder) buildExcluded() api.ExcludedSteps {
	excluded := api.ExcludedSteps{
		Satisfied: map[api.StepID][]api.Name{},
		Blocked:   map[api.StepID][]api.Name{},
		Missing:   map[api.StepID][]api.Name{},
	}
	for sid := range b.visited {
		st := b.cat.Steps[sid]
		if b.included.Contains(sid) {
			continue
		}
		if blocked := b.blocked[sid]; len(blocked) > 0 {
			excluded.Blocked[sid] = blocked
			continue
		}
		if b.outputsAvailable(st) {
			excluded.Satisfied[sid] = stepOutputNames(st)
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

func initAttributeValues(values []any) []*api.AttributeValue {
	res := make([]*api.AttributeValue, 0, len(values))
	for _, value := range values {
		res = append(res, &api.AttributeValue{Value: value})
	}
	return res
}

func buildRequired(step *api.Step) util.Set[api.Name] {
	required := util.Set[api.Name]{}
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			required.Add(name)
		}
	}
	return required
}

func stepOutputNames(step *api.Step) []api.Name {
	var outputs []api.Name
	for name, attr := range step.Attributes {
		if attr.IsOutput() {
			outputs = append(outputs, name)
		}
	}
	return outputs
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
