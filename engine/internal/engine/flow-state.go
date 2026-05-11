package engine

import (
	"errors"
	"slices"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

var (
	ErrFlowNotFound      = errors.New("flow not found")
	ErrInvalidFlowStatus = errors.New("invalid indexed flow status")
)

// GetFlowState retrieves the current state of a flow by its ID
func (e *Engine) GetFlowState(flowID api.FlowID) (api.FlowState, error) {
	state, _, err := e.GetFlowStateSeq(flowID)
	return state, err
}

// GetFlowStatus retrieves the current indexed status of a flow by its ID
func (e *Engine) GetFlowStatus(flowID api.FlowID) (api.FlowStatus, error) {
	key := events.FlowKey(flowID)
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

// GetFlowStateSeq retrieves the current state and next sequence for a flow
func (e *Engine) GetFlowStateSeq(
	flowID api.FlowID,
) (api.FlowState, int64, error) {
	var nextSeq int64
	st, err := e.execFlow(events.FlowKey(flowID),
		func(st api.FlowState, ag *FlowAggregator) error {
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
	flowID api.FlowID, attr api.Name,
) (any, bool, error) {
	fl, err := e.GetFlowState(flowID)
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
func (e *Engine) IsFlowFailed(flow api.FlowState) bool {
	viableGoal := false
	for _, goalID := range flow.Plan.Goals {
		ex := flow.Executions[goalID]
		if policy.StepFailed(ex.Status) {
			return true
		}
		if policy.StepPrunedByRequiredMatch(ex.Status, ex.Error) {
			continue
		}
		if !e.canStepComplete(goalID, flow) {
			return true
		}
		viableGoal = true
	}
	return !viableGoal
}

// HasInputProvider checks if a required attribute has at least one step that
// can provide it in the flow execution plan
func (e *Engine) HasInputProvider(name api.Name, flow api.FlowState) bool {
	deps, ok := flow.Plan.Attributes[name]
	if !ok {
		return false
	}

	if len(deps.Providers) == 0 {
		return true
	}

	for _, providerID := range deps.Providers {
		if e.canStepComplete(providerID, flow) {
			return true
		}
	}
	return false
}

func (e *Engine) isFlowComplete(flow api.FlowState) bool {
	for sid := range flow.Plan.Steps {
		ex := flow.Executions[sid]
		if !policy.StepComplete(ex.Status) {
			return false
		}
	}
	return !allGoalsPruned(flow)
}

func (e *Engine) areOutputsNeeded(stepID api.StepID, flow api.FlowState) bool {
	plan := flow.Plan
	if slices.Contains(plan.Goals, stepID) {
		return true
	}
	return e.needsOutputs(plan.Steps[stepID], flow)
}

func (e *Engine) canStepComplete(stepID api.StepID, flow api.FlowState) bool {
	ex := flow.Executions[stepID]
	if policy.StepTerminal(ex.Status) {
		return policy.StepSucceeded(ex.Status)
	}

	step := flow.Plan.Steps[stepID]
	gate, err := e.stepGateStatus(step, flow)
	if err != nil {
		return false
	}
	if gate == policy.MatchUnknown || gate == policy.MatchUnmatched {
		return true
	}
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			if _, ok := flow.FirstAttribute(name); ok {
				continue
			}
			if !e.HasInputProvider(name, flow) {
				return false
			}
		}
	}

	return true
}

func (e *Engine) needsOutputs(step *api.Step, flow api.FlowState) bool {
	for name, attr := range step.Attributes {
		if e.needsOutput(name, attr, flow) {
			return true
		}
	}
	return false
}

func (e *Engine) needsOutput(
	name api.Name, attr *api.AttributeSpec, flow api.FlowState,
) bool {
	if !attr.IsOutput() {
		return false
	}

	deps, ok := flow.Plan.Attributes[name]
	if !ok || len(deps.Consumers) == 0 {
		return false
	}

	hasValue := len(flow.AttributeValues(name)) > 0
	for _, sid := range deps.Consumers {
		ex, ok := flow.Executions[sid]
		if !ok || !policy.StepPending(ex.Status) {
			continue
		}
		consumer := flow.Plan.Steps[sid]
		input := consumer.Attributes[name]
		if input == nil {
			continue
		}
		if !e.consumerCanDemandInput(consumer, input, flow) {
			continue
		}
		consumerValue := hasValue
		if policy.RequiredInputHasMatch(input) {
			consumerValue = e.inputHasMatchedValue(input, name, flow)
		}
		if policy.ProviderOutputNeeded(
			input.Collect(), consumerValue, canCollectAll(name, flow),
		) {
			return true
		}
	}
	return false
}

func (e *Engine) consumerCanDemandInput(
	consumer *api.Step, input *api.AttributeSpec, flow api.FlowState,
) bool {
	status, err := e.stepGateStatus(consumer, flow)
	if err != nil {
		return false
	}
	if policy.RequiredInputHasMatch(input) {
		return status != policy.MatchUnmatched
	}
	return policy.MatchAllowsNormalDemand(status)
}

func (e *Engine) inputHasMatchedValue(
	input *api.AttributeSpec, name api.Name, flow api.FlowState,
) bool {
	values, _, err := policy.MatchCandidateValues(
		input, flow.AttributeValues(name), e.evaluateRequiredMatch,
	)
	return err == nil && len(values) > 0
}

func canCollectAll(name api.Name, flow api.FlowState) bool {
	deps, ok := flow.Plan.Attributes[name]
	if !ok {
		return false
	}
	for _, sid := range deps.Providers {
		ex, ok := flow.Executions[sid]
		if !ok || !policy.StepTerminal(ex.Status) {
			continue
		}
		if !policy.StepSucceeded(ex.Status) || !hasValueFrom(flow, name, sid) {
			return false
		}
	}
	return true
}

func hasActiveWork(flow api.FlowState) bool {
	for _, ex := range flow.Executions {
		for _, work := range ex.WorkItems {
			if policy.WorkBlocksFlowDeactivation(work.Status) {
				return true
			}
		}
	}
	return false
}

func allGoalsPruned(flow api.FlowState) bool {
	if len(flow.Plan.Goals) == 0 {
		return false
	}
	for _, sid := range flow.Plan.Goals {
		ex := flow.Executions[sid]
		if !policy.StepPrunedByRequiredMatch(ex.Status, ex.Error) {
			return false
		}
	}
	return true
}

func isOutputAttribute(step *api.Step, name api.Name) bool {
	attr, ok := step.Attributes[name]
	return ok && attr.IsOutput()
}
