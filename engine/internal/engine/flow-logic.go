package engine

import (
	"slices"
	"time"

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

func (tx *flowTx) canStartStep(stepID api.StepID, flow *api.FlowState) bool {
	ready, _ := tx.canStartStepAt(stepID, flow, time.Now())
	return ready
}

func (tx *flowTx) canStartStepAt(
	stepID api.StepID, flow *api.FlowState, now time.Time,
) (bool, time.Time) {
	exec := flow.Executions[stepID]
	if exec.Status != api.StepPending {
		return false, time.Time{}
	}
	if !tx.hasRequired(stepID, flow) {
		return false, time.Time{}
	}
	optReady, nextDeadline := tx.hasOptionalReady(stepID, flow, now)
	if !optReady {
		return false, nextDeadline
	}
	return tx.areOutputsNeeded(stepID, flow), time.Time{}
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

func (tx *flowTx) hasOptionalReady(
	stepID api.StepID, flow *api.FlowState, now time.Time,
) (bool, time.Time) {
	step := flow.Plan.Steps[stepID]
	blocked := false
	var nextDeadline time.Time

	for name, attr := range step.Attributes {
		if !attr.IsOptional() {
			continue
		}
		ready, deadline := tx.optionalReadyDeadline(name, attr, flow, now)
		if !ready {
			blocked = true
		}
		if !deadline.IsZero() && (nextDeadline.IsZero() ||
			deadline.Before(nextDeadline)) {
			nextDeadline = deadline
		}
	}

	return !blocked, nextDeadline
}

func (tx *flowTx) optionalReadyDeadline(
	name api.Name, attr *api.AttributeSpec, flow *api.FlowState, now time.Time,
) (bool, time.Time) {
	ready, _, deadline := tx.optionalDecision(name, attr, flow, now)
	return ready, deadline
}

func (tx *flowTx) optionalUsesDefault(
	name api.Name, attr *api.AttributeSpec, flow *api.FlowState, now time.Time,
) (bool, bool) {
	ready, useDefault, _ := tx.optionalDecision(name, attr, flow, now)
	return ready, useDefault
}

func (tx *flowTx) optionalDecision(
	name api.Name, attr *api.AttributeSpec, flow *api.FlowState, now time.Time,
) (bool, bool, time.Time) {
	attrVal, hasAttr := flow.Attributes[name]
	deps, ok := flow.Plan.Attributes[name]
	if hasAttr {
		if !ok || len(deps.Providers) == 0 || attr.Timeout <= 0 ||
			attrVal.Step == "" {
			return true, false, time.Time{}
		}

		deadline, ok := tx.optionalDeadline(
			deps.Providers, flow, attr.Timeout,
		)
		if !ok {
			return true, false, time.Time{}
		}

		if !attrVal.SetAt.IsZero() && attrVal.SetAt.After(deadline) {
			return true, true, time.Time{}
		}
		return true, false, time.Time{}
	}

	if !ok || len(deps.Providers) == 0 {
		return true, false, time.Time{}
	}

	activePotential := false
	for _, providerID := range deps.Providers {
		exec, ok := flow.Executions[providerID]
		if !ok || exec == nil {
			continue
		}
		if stepTransitions.IsTerminal(exec.Status) {
			continue
		}
		if !tx.Engine.canStepComplete(providerID, flow) {
			continue
		}
		activePotential = true
	}

	if !activePotential {
		return true, false, time.Time{}
	}
	if attr.Timeout <= 0 {
		return false, false, time.Time{}
	}

	deadline, ok := tx.optionalDeadline(deps.Providers, flow, attr.Timeout)
	if !ok {
		return false, false, time.Time{}
	}
	if !deadline.After(now) {
		return true, true, time.Time{}
	}
	return false, false, deadline
}

func (tx *flowTx) optionalDeadline(
	providers []api.StepID, flow *api.FlowState, timeoutMS int64,
) (time.Time, bool) {
	var startedAt time.Time

	for _, providerID := range providers {
		exec, ok := flow.Executions[providerID]
		if !ok || exec == nil || exec.StartedAt.IsZero() {
			continue
		}
		if startedAt.IsZero() || exec.StartedAt.Before(startedAt) {
			startedAt = exec.StartedAt
		}
	}

	if startedAt.IsZero() {
		return time.Time{}, false
	}

	return startedAt.Add(time.Duration(timeoutMS) * time.Millisecond), true
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
