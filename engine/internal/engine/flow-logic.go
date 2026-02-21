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
	step := flow.Plan.Steps[stepID]

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

func (tx *flowTx) canStartStep(stepID api.StepID, flow *api.FlowState) bool {
	exec := flow.Executions[stepID]
	if exec.Status != api.StepPending {
		return false
	}
	if !tx.hasRequired(stepID, flow) {
		return false
	}
	return tx.areOutputsNeeded(stepID, flow)
}

func (tx *flowTx) hasRequired(stepID api.StepID, flow *api.FlowState) bool {
	step := flow.Plan.Steps[stepID]
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
func (tx *flowTx) findInitialSteps(flow *api.FlowState) []api.StepID {
	var ready []api.StepID

	for stepID := range flow.Plan.Steps {
		if tx.canStartStep(stepID, flow) {
			ready = append(ready, stepID)
		}
	}

	return ready
}

// findReadySteps finds ready steps among downstream consumers of a completed
// step
func (tx *flowTx) findReadySteps(
	stepID api.StepID, flow *api.FlowState,
) []api.StepID {
	var ready []api.StepID

	for _, consumerID := range tx.getDownstreamConsumers(stepID, flow) {
		if tx.canStartStep(consumerID, flow) {
			ready = append(ready, consumerID)
		}
	}

	return ready
}

// getDownstreamConsumers returns step IDs that consume any output from the
// given step, using the ExecutionPlan's Attributes dependency map
func (tx *flowTx) getDownstreamConsumers(
	stepID api.StepID, flow *api.FlowState,
) []api.StepID {
	step := flow.Plan.Steps[stepID]

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
