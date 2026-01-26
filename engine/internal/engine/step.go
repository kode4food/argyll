package engine

import (
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// FlowStep identifies a step execution within a flow
type FlowStep struct {
	FlowID api.FlowID
	StepID api.StepID
}

var stepTransitions = StateTransitions[api.StepStatus]{
	api.StepPending: util.SetOf(
		api.StepActive,
		api.StepSkipped,
		api.StepFailed,
	),
	api.StepActive: util.SetOf(
		api.StepCompleted,
		api.StepFailed,
	),
	api.StepCompleted: {},
	api.StepFailed:    {},
	api.StepSkipped:   {},
}

// Step state checking methods

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
			if _, hasAttr := flow.Attributes[name]; hasAttr {
				continue
			}
			if !e.HasInputProvider(name, flow) {
				return false
			}
		}
	}

	return true
}
