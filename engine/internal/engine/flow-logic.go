package engine

import (
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
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
	deps := flow.Plan.Attributes[name]
	if deps == nil {
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
	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return false
	}

	if isGoalStep(stepID, flow.Plan.Goals) {
		return true
	}

	return needsOutputs(step, flow)
}

func (e *Engine) isFlowComplete(flow *api.FlowState) bool {
	for stepID := range flow.Plan.Steps {
		if !e.isStepComplete(stepID, flow) {
			return false
		}
	}
	return true
}

func (a *flowActor) canStartStep(stepID api.StepID, flow *api.FlowState) bool {
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return false
	}
	if !a.hasRequired(stepID, flow) {
		return false
	}
	return a.areOutputsNeeded(stepID, flow)
}

func (a *flowActor) hasRequired(stepID api.StepID, flow *api.FlowState) bool {
	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return false
	}
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			if _, ok := flow.Attributes[name]; !ok {
				return false
			}
		}
	}
	return true
}

// findInitialSteps finds steps that can start when a flow begins
func (a *flowActor) findInitialSteps(flow *api.FlowState) []api.StepID {
	var ready []api.StepID

	for stepID := range flow.Plan.Steps {
		if a.canStartStep(stepID, flow) {
			ready = append(ready, stepID)
		}
	}

	return ready
}

// findReadySteps finds ready steps among downstream consumers of a completed
// step
func (a *flowActor) findReadySteps(
	stepID api.StepID, flow *api.FlowState,
) []api.StepID {
	var ready []api.StepID

	for _, consumerID := range a.getDownstreamConsumers(stepID, flow) {
		if a.canStartStep(consumerID, flow) {
			ready = append(ready, consumerID)
		}
	}

	return ready
}

// getDownstreamConsumers returns step IDs that consume any output from the
// given step, using the ExecutionPlan's Attributes dependency map
func (a *flowActor) getDownstreamConsumers(
	stepID api.StepID, flow *api.FlowState,
) []api.StepID {
	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return nil
	}

	seen := util.Set[api.StepID]{}
	var consumers []api.StepID

	for name, attr := range step.Attributes {
		if !attr.IsOutput() {
			continue
		}

		deps := flow.Plan.Attributes[name]
		if deps == nil {
			continue
		}

		for _, consumerID := range deps.Consumers {
			if seen.Contains(consumerID) {
				continue
			}
			seen.Add(consumerID)
			consumers = append(consumers, consumerID)
		}
	}

	return consumers
}

// isGoalStep returns true if the step is a goal step
func (a *flowActor) isGoalStep(stepID api.StepID, flow *api.FlowState) bool {
	return slices.Contains(flow.Plan.Goals, stepID)
}

func isGoalStep(stepID api.StepID, goals []api.StepID) bool {
	return slices.Contains(goals, stepID)
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

	if _, alreadySatisfied := flow.Attributes[name]; alreadySatisfied {
		return false
	}

	attrDeps, ok := flow.Plan.Attributes[name]
	if !ok || len(attrDeps.Consumers) == 0 {
		return false
	}

	return hasPendingConsumer(attrDeps.Consumers, flow.Executions)
}

func hasPendingConsumer(
	consumers []api.StepID, executions api.Executions,
) bool {
	for _, consumerID := range consumers {
		consumerExec, ok := executions[consumerID]
		if !ok {
			continue
		}
		if consumerExec.Status == api.StepPending {
			return true
		}
	}
	return false
}

func hasActiveWork(flow *api.FlowState) bool {
	for _, exec := range flow.Executions {
		for _, item := range exec.WorkItems {
			if item.Status == api.WorkPending || item.Status == api.WorkActive {
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
