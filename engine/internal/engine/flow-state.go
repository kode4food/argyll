package engine

import (
	"errors"
	"slices"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

var (
	ErrFlowNotFound      = errors.New("flow not found")
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
	status, err := e.flowExec.GetStore().GetAggregateStatus(key)
	if err != nil {
		return "", err
	}
	if status == "" {
		return "", ErrFlowNotFound
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
	return e.flowExec.GetStore().GetEvents(events.FlowKey(fid), 0)
}

// GetFlowStateSeq retrieves the current state and next sequence for a flow
func (e *Engine) GetFlowStateSeq(
	fid api.FlowID,
) (api.FlowState, int64, error) {
	var nextSeq int64
	st, err := e.execFlow(events.FlowKey(fid),
		func(fl api.FlowState, ag *FlowAggregator) error {
			nextSeq = ag.NextSequence()
			return nil
		},
	)
	if err != nil {
		return api.FlowState{}, 0, err
	}

	if st.ID == "" {
		return api.FlowState{}, 0, ErrFlowNotFound
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

// IsFlowFailed determines if a flow has failed by checking whether any of its
// goal steps cannot be completed
func (e *Engine) IsFlowFailed(fl api.FlowState) bool {
	viableGoal := false
	for _, goalID := range fl.Plan.Goals {
		ex := fl.Executions[goalID]
		if policy.StepFailed(ex.Status) {
			return true
		}
		if policy.StepPrunedByRequiredMatch(ex.Status, ex.Error) {
			continue
		}
		if !e.canStepComplete(goalID, fl) {
			return true
		}
		viableGoal = true
	}
	return !viableGoal
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

func (e *Engine) areOutputsNeeded(sid api.StepID, fl api.FlowState) bool {
	pl := fl.Plan
	if slices.Contains(pl.Goals, sid) {
		return true
	}
	return e.needsOutputs(pl.Steps[sid], fl)
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

func (e *Engine) needsOutputs(st *api.Step, fl api.FlowState) bool {
	for name, attr := range st.Attributes {
		if e.needsOutput(name, attr, fl) {
			return true
		}
	}
	return false
}

func (e *Engine) needsOutput(
	name api.Name, attr *api.AttributeSpec, fl api.FlowState,
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
		if !ok || !policy.StepPending(ex.Status) {
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

func isFlowComplete(fl api.FlowState) bool {
	for sid := range fl.Plan.Steps {
		ex := fl.Executions[sid]
		if !policy.StepComplete(ex.Status) {
			return false
		}
	}
	return !allGoalsPruned(fl)
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

func allGoalsPruned(fl api.FlowState) bool {
	if len(fl.Plan.Goals) == 0 {
		return false
	}
	for _, sid := range fl.Plan.Goals {
		ex := fl.Executions[sid]
		if !policy.StepPrunedByRequiredMatch(ex.Status, ex.Error) {
			return false
		}
	}
	return true
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
