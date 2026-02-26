package engine

import (
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
)

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

func (e *Engine) areOutputsNeeded(stepID api.StepID, flow *api.FlowState) bool {
	plan := flow.Plan
	if slices.Contains(plan.Goals, stepID) {
		return true
	}
	return needsOutputs(plan.Steps[stepID], flow)
}

func (e *Engine) isFlowComplete(flow *api.FlowState) bool {
	for stepID := range flow.Plan.Steps {
		if !e.isStepComplete(stepID, flow) {
			return false
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
