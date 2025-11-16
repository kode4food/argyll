package events

import (
	"fmt"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

const workflowPrefix = "workflow"

// WorkflowAppliers contains the event applier functions for workflow events
var WorkflowAppliers = makeWorkflowAppliers()

// NewWorkflowState creates an empty workflow state with initialized maps for
// attributes and step executions
func NewWorkflowState() *api.WorkflowState {
	return &api.WorkflowState{
		Attributes: map[api.Name]*api.AttributeValue{},
		Executions: map[timebox.ID]*api.ExecutionState{},
	}
}

// IsWorkflowEvent returns true if the event belongs to a workflow aggregate
func IsWorkflowEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 2 && ev.AggregateID[0] == workflowPrefix
}

func makeWorkflowAppliers() timebox.Appliers[*api.WorkflowState] {
	workflowStartedApplier := timebox.MakeApplier(workflowStarted)
	workflowCompletedApplier := timebox.MakeApplier(workflowCompleted)
	workflowFailedApplier := timebox.MakeApplier(workflowFailed)
	stepStartedApplier := timebox.MakeApplier(stepStarted)
	stepCompletedApplier := timebox.MakeApplier(stepCompleted)
	stepFailedApplier := timebox.MakeApplier(stepFailed)
	stepSkippedApplier := timebox.MakeApplier(stepSkipped)
	attributeSetApplier := timebox.MakeApplier(attributeSet)
	workItemStartedApplier := timebox.MakeApplier(workItemStarted)
	workItemCompletedApplier := timebox.MakeApplier(workItemCompleted)
	workItemFailedApplier := timebox.MakeApplier(workItemFailed)
	retryScheduledApplier := timebox.MakeApplier(retryScheduled)

	return timebox.Appliers[*api.WorkflowState]{
		api.EventTypeWorkflowStarted:   workflowStartedApplier,
		api.EventTypeWorkflowCompleted: workflowCompletedApplier,
		api.EventTypeWorkflowFailed:    workflowFailedApplier,
		api.EventTypeStepStarted:       stepStartedApplier,
		api.EventTypeStepCompleted:     stepCompletedApplier,
		api.EventTypeStepFailed:        stepFailedApplier,
		api.EventTypeStepSkipped:       stepSkippedApplier,
		api.EventTypeAttributeSet:      attributeSetApplier,
		api.EventTypeWorkStarted:       workItemStartedApplier,
		api.EventTypeWorkCompleted:     workItemCompletedApplier,
		api.EventTypeWorkFailed:        workItemFailedApplier,
		api.EventTypeRetryScheduled:    retryScheduledApplier,
	}
}

func workflowStarted(
	_ *api.WorkflowState, ev *timebox.Event, data api.WorkflowStartedEvent,
) *api.WorkflowState {
	exec := createExecutions(data.Plan)

	attributes := map[api.Name]*api.AttributeValue{}
	for key, value := range data.Init {
		attributes[key] = &api.AttributeValue{Value: value}
	}

	return &api.WorkflowState{
		ID:          data.FlowID,
		Status:      api.WorkflowActive,
		Plan:        data.Plan,
		Attributes:  attributes,
		Executions:  exec,
		CreatedAt:   ev.Timestamp,
		LastUpdated: ev.Timestamp,
	}
}

func workflowCompleted(
	st *api.WorkflowState, ev *timebox.Event, _ api.WorkflowCompletedEvent,
) *api.WorkflowState {
	return st.
		SetStatus(api.WorkflowCompleted).
		SetCompletedAt(ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func workflowFailed(
	st *api.WorkflowState, ev *timebox.Event, data api.WorkflowFailedEvent,
) *api.WorkflowState {
	return st.
		SetStatus(api.WorkflowFailed).
		SetError(data.Error).
		SetCompletedAt(ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func stepStarted(
	st *api.WorkflowState, ev *timebox.Event, data api.StepStartedEvent,
) *api.WorkflowState {
	exec := &api.ExecutionState{
		Status:    api.StepPending,
		Inputs:    api.Args{},
		Outputs:   api.Args{},
		StartedAt: time.Time{},
		WorkItems: map[api.Token]*api.WorkState{},
	}

	updated := exec.
		SetStatus(api.StepActive).
		SetStartedAt(ev.Timestamp).
		SetInputs(data.Inputs)

	return st.
		SetExecution(data.StepID, updated).
		SetLastUpdated(ev.Timestamp)
}

func stepCompleted(
	st *api.WorkflowState, ev *timebox.Event, data api.StepCompletedEvent,
) *api.WorkflowState {
	exec := getExecution(st, data.StepID, "stepCompleted")

	updated := exec.
		SetStatus(api.StepCompleted).
		SetCompletedAt(ev.Timestamp).
		SetDuration(data.Duration).
		SetOutputs(data.Outputs)

	return st.
		SetExecution(data.StepID, updated).
		SetLastUpdated(ev.Timestamp)
}

func stepFailed(
	st *api.WorkflowState, ev *timebox.Event, data api.StepFailedEvent,
) *api.WorkflowState {
	exec := getExecution(st, data.StepID, "stepFailed")

	return st.
		SetExecution(data.StepID,
			exec.
				SetStatus(api.StepFailed).
				SetError(data.Error).
				SetCompletedAt(ev.Timestamp),
		).
		SetLastUpdated(ev.Timestamp)
}

func stepSkipped(
	st *api.WorkflowState, ev *timebox.Event, data api.StepSkippedEvent,
) *api.WorkflowState {
	exec := getExecution(st, data.StepID, "stepSkipped")

	return st.
		SetExecution(data.StepID,
			exec.
				SetStatus(api.StepSkipped).
				SetError(data.Reason).
				SetCompletedAt(ev.Timestamp),
		).
		SetLastUpdated(ev.Timestamp)
}

func attributeSet(
	st *api.WorkflowState, ev *timebox.Event, data api.AttributeSetEvent,
) *api.WorkflowState {
	return st.
		SetAttribute(data.Key, &api.AttributeValue{
			Value: data.Value,
			Step:  data.StepID,
		}).
		SetLastUpdated(ev.Timestamp)
}

func createExecutions(p *api.ExecutionPlan) map[timebox.ID]*api.ExecutionState {
	exec := map[timebox.ID]*api.ExecutionState{}
	if p == nil {
		return exec
	}

	for stepID := range p.Steps {
		exec[stepID] = &api.ExecutionState{
			Status:    api.StepPending,
			Inputs:    api.Args{},
			Outputs:   api.Args{},
			StartedAt: time.Time{},
			WorkItems: map[api.Token]*api.WorkState{},
		}
	}
	return exec
}

func getExecution(
	st *api.WorkflowState, stepID timebox.ID, fn string,
) *api.ExecutionState {
	if exec, ok := st.Executions[stepID]; ok {
		return exec
	}
	panic(
		fmt.Errorf("%s: execution does not exist for step %s", fn, stepID),
	)
}

func workItemStarted(
	st *api.WorkflowState, ev *timebox.Event, data api.WorkStartedEvent,
) *api.WorkflowState {
	exec := getExecution(st, data.StepID, "workItemStarted")

	item := &api.WorkState{
		Status:    api.WorkActive,
		StartedAt: ev.Timestamp,
		Inputs:    data.Inputs,
		Outputs:   api.Args{},
	}

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}

func workItemCompleted(
	st *api.WorkflowState, ev *timebox.Event, data api.WorkCompletedEvent,
) *api.WorkflowState {
	exec := getExecution(st, data.StepID, "workItemCompleted")

	if exec.WorkItems == nil || exec.WorkItems[data.Token] == nil {
		return st
	}

	item := exec.WorkItems[data.Token]
	updated := &api.WorkState{
		Status:      api.WorkCompleted,
		StartedAt:   item.StartedAt,
		CompletedAt: ev.Timestamp,
		Inputs:      item.Inputs,
		Outputs:     data.Outputs,
	}

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, updated)).
		SetLastUpdated(ev.Timestamp)
}

func workItemFailed(
	st *api.WorkflowState, ev *timebox.Event, data api.WorkFailedEvent,
) *api.WorkflowState {
	exec := getExecution(st, data.StepID, "workItemFailed")

	if exec.WorkItems == nil || exec.WorkItems[data.Token] == nil {
		return st
	}

	item := exec.WorkItems[data.Token]
	updated := &api.WorkState{
		Status:      api.WorkFailed,
		StartedAt:   item.StartedAt,
		CompletedAt: ev.Timestamp,
		Inputs:      item.Inputs,
		Error:       data.Error,
	}

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, updated)).
		SetLastUpdated(ev.Timestamp)
}

func retryScheduled(
	st *api.WorkflowState, ev *timebox.Event, data api.RetryScheduledEvent,
) *api.WorkflowState {
	exec := getExecution(st, data.StepID, "retryScheduled")

	if exec.WorkItems == nil || exec.WorkItems[data.Token] == nil {
		return st
	}

	item := exec.WorkItems[data.Token]
	updated := item.
		SetStatus(api.WorkPending).
		SetRetryCount(data.RetryCount).
		SetNextRetryAt(data.NextRetryAt).
		SetLastError(data.Error)

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, updated)).
		SetLastUpdated(ev.Timestamp)
}
