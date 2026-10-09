package engine

import (
	"errors"
	"slices"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util"
)

var (
	ErrInvalidFlowStatus = errors.New("invalid indexed flow status")
)

// GetFlowState retrieves the current state of a flow by its ID
func (e *Engine) GetFlowState(fid api.FlowID) (api.FlowState, error) {
	state, _, err := e.GetFlowStateSeq(fid)
	return state, err
}

// GetFlowStatus retrieves the current indexed status of a flow by its ID
func (e *Engine) GetFlowStatus(fid api.FlowID) (api.FlowStatus, error) {
	key := events.FlowKey(fid)
	status, err := e.flowStore.GetAggregateStatus(key)
	if err != nil {
		return "", err
	}
	if status == "" {
		return "", api.ErrFlowNotFound
	}

	switch api.FlowStatus(status) {
	case api.FlowActive, api.FlowCompleted, api.FlowFailed:
		return api.FlowStatus(status), nil
	default:
		return "", ErrInvalidFlowStatus
	}
}

// GetFlowEvents retrieves all events for a flow aggregate
func (e *Engine) GetFlowEvents(fid api.FlowID) ([]*timebox.Event, error) {
	return e.flowStore.GetEvents(events.FlowKey(fid), 0)
}

// GetFlowStateSeq retrieves the current state and next sequence for a flow
func (e *Engine) GetFlowStateSeq(
	fid api.FlowID,
) (api.FlowState, int64, error) {
	var nextSeq int64
	st, err := e.flowExec.Exec(events.FlowKey(fid),
		func(fl api.FlowState, ag *FlowAggregator) error {
			nextSeq = ag.NextSequence()
			return nil
		},
	)
	if err != nil {
		return api.FlowState{}, 0, err
	}

	if st.ID == "" {
		return api.FlowState{}, 0, api.ErrFlowNotFound
	}

	return st, nextSeq, nil
}

// GetAttribute retrieves a specific attribute value from the flow state,
// returning the value, whether it exists, and any error
func (e *Engine) GetAttribute(
	fid api.FlowID, attr api.Name,
) (any, bool, error) {
	fl, err := e.GetFlowState(fid)
	if err != nil {
		return nil, false, err
	}

	if av, ok := fl.Attributes[attr]; ok {
		if len(av) > 0 {
			return av[0].Value, true, nil
		}
	}
	return nil, false, nil
}

// IsFlowFailed determines if a flow has failed by checking whether every one of
// its goal sets has become impossible
func (e *Engine) IsFlowFailed(fl api.FlowState) bool {
	for _, goals := range fl.Plan.Goals.Sets() {
		if !e.goalSetImpossible(goals, fl) {
			return false
		}
	}
	return true
}

// HasInputProvider checks if a required attribute has at least one step that
// can provide it in the flow execution plan
func (e *Engine) HasInputProvider(name api.Name, fl api.FlowState) bool {
	deps, ok := fl.Plan.Attributes[name]
	if !ok {
		return false
	}

	if len(deps.Providers) == 0 {
		return true
	}

	for _, providerID := range deps.Providers {
		if e.canStepComplete(providerID, fl) {
			return true
		}
	}
	return false
}

func (e *Engine) goalSetImpossible(goals []api.StepID, fl api.FlowState) bool {
	return goalSetFailed(goals, fl) || slices.ContainsFunc(goals,
		func(sid api.StepID) bool {
			ex := fl.Executions[sid]
			return !policy.StepPrunedByRequiredMatch(ex.Status, ex.Error) &&
				!e.canStepComplete(sid, fl)
		},
	)
}

func (e *Engine) areOutputsNeeded(
	sid api.StepID, fl api.FlowState, goals []api.StepID,
) bool {
	if slices.Contains(goals, sid) {
		return true
	}
	return e.needsOutputs(fl.Plan.Steps[sid], fl, goalScope(fl.Plan, goals))
}

func (e *Engine) canStepComplete(sid api.StepID, fl api.FlowState) bool {
	ex := fl.Executions[sid]
	if policy.StepTerminal(ex.Status) {
		return policy.StepSucceeded(ex.Status)
	}

	st := fl.Plan.Steps[sid]
	willSkip, _ := e.matchGateWillSkip(st, fl)
	if willSkip {
		return true
	}
	if hasPendingMatchGate(st, fl) {
		return true
	}

	for name, attr := range st.Attributes {
		if attr.IsRequired() {
			if _, ok := fl.FirstAttribute(name); ok {
				continue
			}
			if !e.HasInputProvider(name, fl) {
				return false
			}
		}
	}

	return true
}

func (e *Engine) matchGateWillSkip(
	st *api.Step, fl api.FlowState,
) (bool, error) {
	unsatisfied, err := e.matchGateUnsatisfiedInputs(st, fl)
	if err != nil {
		return false, err
	}
	return len(unsatisfied) > 0, nil
}

func (e *Engine) matchGateUnsatisfiedInputs(
	st *api.Step, fl api.FlowState,
) ([]api.Name, error) {
	var unsatisfied []api.Name
	for name, attr := range st.Attributes {
		if !policy.RequiredInputHasMatch(attr) {
			continue
		}
		providers, _ := providerSummaryFor(fl, name)
		if !providers.Terminal {
			continue
		}
		status, err := policy.RequiredMatchStatus(policy.RequiredMatchSpec{
			Attr:     attr,
			Values:   fl.AttributeValues(name),
			Provider: providers,
			Match:    e.Matcher,
		})
		if err != nil {
			return nil, err
		}
		if policy.MatchAllowsStepSkip(status) {
			unsatisfied = append(unsatisfied, name)
		}
	}
	slices.Sort(unsatisfied)
	return unsatisfied, nil
}

func (e *Engine) needsOutputs(
	st *api.Step, fl api.FlowState, scope util.Set[api.StepID],
) bool {
	for name, attr := range st.Attributes {
		if e.needsOutput(name, attr, fl, scope) {
			return true
		}
	}
	return false
}

// needsOutput reports whether a pending consumer within scope still needs
// the named output
func (e *Engine) needsOutput(
	name api.Name, attr *api.AttributeSpec, fl api.FlowState,
	scope util.Set[api.StepID],
) bool {
	if !attr.IsOutput() {
		return false
	}

	deps, ok := fl.Plan.Attributes[name]
	if !ok || len(deps.Consumers) == 0 {
		return false
	}

	for _, sid := range deps.Consumers {
		ex, ok := fl.Executions[sid]
		if !ok || !policy.StepPending(ex.Status) || !scope.Contains(sid) {
			continue
		}
		consumer := fl.Plan.Steps[sid]
		input := consumer.Attributes[name]
		if input == nil {
			continue
		}
		if willSkip, _ := e.matchGateWillSkip(consumer, fl); willSkip {
			continue
		}
		hasValue := e.inputHasValue(name, input, fl)
		if policy.ProviderOutputNeeded(
			input.Collect(), hasValue, canCollectAll(name, fl),
		) {
			return true
		}
	}
	return false
}

func (e *Engine) inputHasValue(
	name api.Name, attr *api.AttributeSpec, fl api.FlowState,
) bool {
	values := fl.AttributeValues(name)
	if !policy.RequiredInputHasMatch(attr) {
		return len(values) > 0
	}
	matched, _, _ := policy.MatchCandidateValues(attr, values, e.Matcher)
	return len(matched) > 0
}

// isFlowComplete reports whether the active goal set has succeeded and no step
// is still pending or running
func isFlowComplete(fl api.FlowState) bool {
	active := activeGoals(fl)
	if active == nil || !goalsComplete(active.Steps, fl) {
		return false
	}
	for sid := range fl.Plan.Steps {
		if !policy.StepTerminal(fl.Executions[sid].Status) {
			return false
		}
	}
	return true
}

// activeGoals returns the first link in the goal chain that has not failed,
// since checkUnreachable fails impossible pending goals before any start
func activeGoals(fl api.FlowState) *api.Goals {
	for g := &fl.Plan.Goals; g != nil; g = g.Else {
		if !goalSetFailed(g.Steps, fl) {
			return g
		}
	}
	return nil
}

// startGoals returns the goals whose work may start: the active set only
func startGoals(fl api.FlowState) []api.StepID {
	if active := activeGoals(fl); active != nil {
		return active.Steps
	}
	return nil
}

// keptGoals returns the goals whose outputs must be kept: the active set once
// it succeeds, otherwise the active set and every later fallback set
func keptGoals(fl api.FlowState) []api.StepID {
	active := activeGoals(fl)
	if active == nil {
		return fl.Plan.Goals.AllSteps()
	}
	if goalsComplete(active.Steps, fl) {
		return active.Steps
	}
	return active.AllSteps()
}

func hasPendingMatchGate(st *api.Step, fl api.FlowState) bool {
	for name, attr := range st.Attributes {
		if !policy.RequiredInputHasMatch(attr) {
			continue
		}
		providers, _ := providerSummaryFor(fl, name)
		if !providers.Terminal {
			return true
		}
	}
	return false
}

// goalSetFailed reports whether a goal in the set failed, or every goal in it
// was pruned by a required match
func goalSetFailed(goals []api.StepID, fl api.FlowState) bool {
	viableGoal := false
	for _, sid := range goals {
		ex := fl.Executions[sid]
		if policy.StepFailed(ex.Status) {
			return true
		}
		if !policy.StepPrunedByRequiredMatch(ex.Status, ex.Error) {
			viableGoal = true
		}
	}
	return !viableGoal
}

func goalsComplete(goals []api.StepID, fl api.FlowState) bool {
	return !slices.ContainsFunc(goals, func(sid api.StepID) bool {
		return !policy.StepComplete(fl.Executions[sid].Status)
	})
}

// goalScope returns the goals plus every plan step that can feed them
func goalScope(pl *api.ExecutionPlan, goals []api.StepID) util.Set[api.StepID] {
	res := util.Set[api.StepID]{}
	todo := slices.Clone(goals)
	for len(todo) > 0 {
		sid := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		st, ok := pl.Steps[sid]
		if !ok || res.Contains(sid) {
			continue
		}
		res.Add(sid)
		for name, attr := range st.Attributes {
			if deps, ok := pl.Attributes[name]; ok && attr.IsInput() {
				todo = append(todo, deps.Providers...)
			}
		}
	}
	return res
}

func canCollectAll(name api.Name, fl api.FlowState) bool {
	deps, ok := fl.Plan.Attributes[name]
	if !ok {
		return false
	}
	for _, sid := range deps.Providers {
		ex, ok := fl.Executions[sid]
		if !ok || !policy.StepTerminal(ex.Status) {
			continue
		}
		if !policy.StepSucceeded(ex.Status) || !hasValueFrom(fl, name, sid) {
			return false
		}
	}
	return true
}

func hasActiveWork(fl api.FlowState) bool {
	for _, ex := range fl.Executions {
		for _, work := range ex.WorkItems {
			if policy.WorkBlocksFlowDeactivation(work.Status) {
				return true
			}
		}
	}
	return false
}

func isOutputAttribute(st *api.Step, name api.Name) bool {
	attr, ok := st.Attributes[name]
	return ok && attr.IsOutput()
}
