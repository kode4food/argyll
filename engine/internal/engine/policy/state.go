package policy

import "github.com/kode4food/argyll/engine/pkg/api"

// FlowTerminal reports whether a flow has reached an outcome state. Terminal
// flows should not start more steps or work, though active work may still need
// to finish before deactivation
func FlowTerminal(status api.FlowStatus) bool {
	return FlowTransitions.IsTerminal(status)
}

// StepTerminal reports whether a step can no longer change state
func StepTerminal(status api.StepStatus) bool {
	return StepTransitions.IsTerminal(status)
}

// StepComplete reports whether a step counts as done for flow completion
// Skipped steps are complete because the engine intentionally determined their
// outputs are not needed
func StepComplete(status api.StepStatus) bool {
	return status == api.StepCompleted || status == api.StepSkipped
}

// StepSucceeded reports whether a step completed with outputs available for
// downstream consumers
func StepSucceeded(status api.StepStatus) bool {
	return status == api.StepCompleted
}

// StepPending reports whether a step is waiting to be started or skipped
func StepPending(status api.StepStatus) bool {
	return status == api.StepPending
}

// StepActive reports whether a step has started and owns work items
func StepActive(status api.StepStatus) bool {
	return status == api.StepActive
}

// StepFailed reports whether a step reached a failed terminal state
func StepFailed(status api.StepStatus) bool {
	return status == api.StepFailed
}

// WorkCanTransition reports whether a work item status change is valid
func WorkCanTransition(from, to api.WorkStatus) bool {
	return WorkTransitions.CanTransition(from, to)
}

// WorkActive reports whether a work item is currently executing
func WorkActive(status api.WorkStatus) bool {
	return status == api.WorkActive
}

// WorkBlocksFlowDeactivation reports whether a work item still prevents a
// terminal flow from being deactivated. Pending work can still be claimed or
// cancelled, active work may still report back, and compensating work must
// finish before the flow is considered fully settled
func WorkBlocksFlowDeactivation(status api.WorkStatus) bool {
	return status == api.WorkPending ||
		status == api.WorkActive ||
		status == api.WorkCompensating
}

// WorkTerminal reports whether a work item has a final success or failure
// outcome
func WorkTerminal(status api.WorkStatus) bool {
	return status == api.WorkSucceeded || status == api.WorkFailed
}

// WorkSucceeded reports whether a work item produced successful outputs
func WorkSucceeded(status api.WorkStatus) bool {
	return status == api.WorkSucceeded
}

// WorkPending reports whether a work item is waiting to be dispatched
func WorkPending(status api.WorkStatus) bool {
	return status == api.WorkPending
}

// WorkNotCompleted reports whether a work item asked to be retried rather than
// finalized
func WorkNotCompleted(status api.WorkStatus) bool {
	return status == api.WorkNotCompleted
}

// WorkClaimableForRetry reports whether retry task handling may try to claim
// the work item for dispatch ownership
func WorkClaimableForRetry(status api.WorkStatus) bool {
	return status == api.WorkPending || status == api.WorkFailed
}

// WorkCompActive reports whether a work item is in the compensation phase
func WorkCompActive(status api.WorkStatus) bool {
	return status == api.WorkCompensating
}

// WorkCompTerminal reports whether a work item's compensation has settled
func WorkCompTerminal(status api.WorkStatus) bool {
	return status == api.WorkCompensated || status == api.WorkCompFailed
}
