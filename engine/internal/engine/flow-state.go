package engine

import (
	"errors"
	"slices"

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
		return av.Value, true, nil
	}
	return nil, false, nil
}

// IsFlowFailed determines if a flow has failed by checking whether any of its
// goal steps cannot be completed
func (e *Engine) IsFlowFailed(flow api.FlowState) bool {
	for _, goalID := range flow.Plan.Goals {
		if !e.canStepComplete(goalID, flow) {
			return true
		}
	}
	return false
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
		if !e.isStepComplete(sid, flow) {
			return false
		}
	}
	return true
}

func (e *Engine) areOutputsNeeded(stepID api.StepID, flow api.FlowState) bool {
	plan := flow.Plan
	if slices.Contains(plan.Goals, stepID) {
		return true
	}
	return needsOutputs(plan.Steps[stepID], flow)
}

func (e *Engine) isStepComplete(stepID api.StepID, flow api.FlowState) bool {
	ex := flow.Executions[stepID]
	return ex.Status == api.StepCompleted || ex.Status == api.StepSkipped
}

func (e *Engine) canStepComplete(stepID api.StepID, flow api.FlowState) bool {
	ex := flow.Executions[stepID]
	if stepTransitions.IsTerminal(ex.Status) {
		return ex.Status == api.StepCompleted
	}

	step := flow.Plan.Steps[stepID]
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			if _, ok := flow.Attributes[name]; ok {
				continue
			}
			if !e.HasInputProvider(name, flow) {
				return false
			}
		}
	}

	return true
}

func needsOutputs(step *api.Step, flow api.FlowState) bool {
	for name, attr := range step.Attributes {
		if needsOutput(name, attr, flow) {
			return true
		}
	}
	return false
}

func needsOutput(
	name api.Name, attr *api.AttributeSpec, flow api.FlowState,
) bool {
	if !attr.IsOutput() {
		return false
	}

	if _, ok := flow.Attributes[name]; ok {
		return false
	}

	deps, ok := flow.Plan.Attributes[name]
	if !ok || len(deps.Consumers) == 0 {
		return false
	}

	return hasPendingConsumer(deps.Consumers, flow.Executions)
}

func hasPendingConsumer(
	consumers []api.StepID, executions api.Executions,
) bool {
	for _, id := range consumers {
		ex, ok := executions[id]
		if !ok {
			continue
		}
		if ex.Status == api.StepPending {
			return true
		}
	}
	return false
}

func hasActiveWork(flow api.FlowState) bool {
	for _, ex := range flow.Executions {
		for _, work := range ex.WorkItems {
			if work.Status == api.WorkPending || work.Status == api.WorkActive {
				return true
			}
		}
	}
	return false
}

func isOutputAttribute(step *api.Step, name api.Name) bool {
	attr, ok := step.Attributes[name]
	return ok && attr.IsOutput()
}
