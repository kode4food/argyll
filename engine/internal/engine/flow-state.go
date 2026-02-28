package engine

import (
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// GetFlowState retrieves the current state of a flow by its ID.
func (e *Engine) GetFlowState(flowID api.FlowID) (*api.FlowState, error) {
	state, _, err := e.GetFlowStateSeq(flowID)
	return state, err
}

// GetFlowStateSeq retrieves the current state and next sequence for a flow.
func (e *Engine) GetFlowStateSeq(
	flowID api.FlowID,
) (*api.FlowState, int64, error) {
	var nextSeq int64
	state, err := e.execFlow(events.FlowKey(flowID),
		func(st *api.FlowState, ag *FlowAggregator) error {
			nextSeq = ag.NextSequence()
			return nil
		},
	)
	if err != nil {
		return nil, 0, err
	}

	if state.ID == "" {
		return nil, 0, ErrFlowNotFound
	}

	return state, nextSeq, nil
}

// GetAttribute retrieves a specific attribute value from the flow state,
// returning the value, whether it exists, and any error.
func (e *Engine) GetAttribute(
	flowID api.FlowID, attr api.Name,
) (any, bool, error) {
	flow, err := e.GetFlowState(flowID)
	if err != nil {
		return nil, false, err
	}

	if av, ok := flow.Attributes[attr]; ok {
		return av.Value, true, nil
	}
	return nil, false, nil
}

// IsFlowFailed determines if a flow has failed by checking whether any of its
// goal steps cannot be completed
func (e *Engine) IsFlowFailed(flow *api.FlowState) bool {
	for _, goalID := range flow.Plan.Goals {
		if !e.canStepComplete(goalID, flow) {
			return true
		}
	}
	return false
}

// HasInputProvider checks if a required attribute has at least one step that
// can provide it in the flow execution plan
func (e *Engine) HasInputProvider(name api.Name, flow *api.FlowState) bool {
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

func (e *Engine) isFlowComplete(flow *api.FlowState) bool {
	for stepID := range flow.Plan.Steps {
		if !e.isStepComplete(stepID, flow) {
			return false
		}
	}
	return true
}

func (e *Engine) areOutputsNeeded(stepID api.StepID, flow *api.FlowState) bool {
	plan := flow.Plan
	if slices.Contains(plan.Goals, stepID) {
		return true
	}
	return needsOutputs(plan.Steps[stepID], flow)
}

func (e *Engine) isStepComplete(stepID api.StepID, flow *api.FlowState) bool {
	exec := flow.Executions[stepID]
	return exec.Status == api.StepCompleted || exec.Status == api.StepSkipped
}

func (e *Engine) canStepComplete(stepID api.StepID, flow *api.FlowState) bool {
	exec := flow.Executions[stepID]
	if stepTransitions.IsTerminal(exec.Status) {
		return exec.Status == api.StepCompleted
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

func needsOutputs(step *api.Step, flow *api.FlowState) bool {
	for name, attr := range step.Attributes {
		if needsOutput(name, attr, flow) {
			return true
		}
	}
	return false
}

func needsOutput(
	name api.Name, attr *api.AttributeSpec, flow *api.FlowState,
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
		exec, ok := executions[id]
		if !ok {
			continue
		}
		if exec.Status == api.StepPending {
			return true
		}
	}
	return false
}

func hasActiveWork(flow *api.FlowState) bool {
	for _, exec := range flow.Executions {
		for _, work := range exec.WorkItems {
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
