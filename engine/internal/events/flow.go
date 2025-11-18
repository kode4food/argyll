package events

import (
	"fmt"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

const flowPrefix = "flow"

// FlowAppliers contains the event applier functions for flow events
var FlowAppliers = makeFlowAppliers()

// NewFlowState creates an empty flow state with initialized maps for
// attributes and step executions
func NewFlowState() *api.FlowState {
	return &api.FlowState{
		Attributes: map[api.Name]*api.AttributeValue{},
		Executions: map[timebox.ID]*api.ExecutionState{},
	}
}

// IsFlowEvent returns true if the event belongs to a flow aggregate
func IsFlowEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 2 && ev.AggregateID[0] == flowPrefix
}

func makeFlowAppliers() timebox.Appliers[*api.FlowState] {
	flowStartedApplier := timebox.MakeApplier(flowStarted)
	flowCompletedApplier := timebox.MakeApplier(flowCompleted)
	flowFailedApplier := timebox.MakeApplier(flowFailed)
	stepStartedApplier := timebox.MakeApplier(stepStarted)
	stepCompletedApplier := timebox.MakeApplier(stepCompleted)
	stepFailedApplier := timebox.MakeApplier(stepFailed)
	stepSkippedApplier := timebox.MakeApplier(stepSkipped)
	attributeSetApplier := timebox.MakeApplier(attributeSet)
	workItemStartedApplier := timebox.MakeApplier(workItemStarted)
	workItemCompletedApplier := timebox.MakeApplier(workItemCompleted)
	workItemFailedApplier := timebox.MakeApplier(workItemFailed)
	retryScheduledApplier := timebox.MakeApplier(retryScheduled)

	return timebox.Appliers[*api.FlowState]{
		api.EventTypeFlowStarted:    flowStartedApplier,
		api.EventTypeFlowCompleted:  flowCompletedApplier,
		api.EventTypeFlowFailed:     flowFailedApplier,
		api.EventTypeStepStarted:    stepStartedApplier,
		api.EventTypeStepCompleted:  stepCompletedApplier,
		api.EventTypeStepFailed:     stepFailedApplier,
		api.EventTypeStepSkipped:    stepSkippedApplier,
		api.EventTypeAttributeSet:   attributeSetApplier,
		api.EventTypeWorkStarted:    workItemStartedApplier,
		api.EventTypeWorkCompleted:  workItemCompletedApplier,
		api.EventTypeWorkFailed:     workItemFailedApplier,
		api.EventTypeRetryScheduled: retryScheduledApplier,
	}
}

func flowStarted(
	_ *api.FlowState, ev *timebox.Event, data api.FlowStartedEvent,
) *api.FlowState {
	exec := createExecutions(data.Plan)

	attributes := map[api.Name]*api.AttributeValue{}
	for key, value := range data.Init {
		attributes[key] = &api.AttributeValue{Value: value}
	}

	return &api.FlowState{
		ID:          data.FlowID,
		Status:      api.FlowActive,
		Plan:        data.Plan,
		Attributes:  attributes,
		Executions:  exec,
		CreatedAt:   ev.Timestamp,
		LastUpdated: ev.Timestamp,
	}
}

func flowCompleted(
	st *api.FlowState, ev *timebox.Event, _ api.FlowCompletedEvent,
) *api.FlowState {
	return st.
		SetStatus(api.FlowCompleted).
		SetCompletedAt(ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func flowFailed(
	st *api.FlowState, ev *timebox.Event, data api.FlowFailedEvent,
) *api.FlowState {
	return st.
		SetStatus(api.FlowFailed).
		SetError(data.Error).
		SetCompletedAt(ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func stepStarted(
	st *api.FlowState, ev *timebox.Event, data api.StepStartedEvent,
) *api.FlowState {
	exec := &api.ExecutionState{
		Status:    api.StepPending,
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
	st *api.FlowState, ev *timebox.Event, data api.StepCompletedEvent,
) *api.FlowState {
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
	st *api.FlowState, ev *timebox.Event, data api.StepFailedEvent,
) *api.FlowState {
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
	st *api.FlowState, ev *timebox.Event, data api.StepSkippedEvent,
) *api.FlowState {
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
	st *api.FlowState, ev *timebox.Event, data api.AttributeSetEvent,
) *api.FlowState {
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
			WorkItems: map[api.Token]*api.WorkState{},
		}
	}
	return exec
}

func getExecution(
	st *api.FlowState, stepID timebox.ID, fn string,
) *api.ExecutionState {
	if exec, ok := st.Executions[stepID]; ok {
		return exec
	}
	panic(
		fmt.Errorf("%s: execution does not exist for step %s", fn, stepID),
	)
}

func workItemStarted(
	st *api.FlowState, ev *timebox.Event, data api.WorkStartedEvent,
) *api.FlowState {
	exec := getExecution(st, data.StepID, "workItemStarted")

	item := &api.WorkState{
		Status:    api.WorkActive,
		StartedAt: ev.Timestamp,
		Inputs:    data.Inputs,
	}

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}

func workItemCompleted(
	st *api.FlowState, ev *timebox.Event, data api.WorkCompletedEvent,
) *api.FlowState {
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
	st *api.FlowState, ev *timebox.Event, data api.WorkFailedEvent,
) *api.FlowState {
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
	st *api.FlowState, ev *timebox.Event, data api.RetryScheduledEvent,
) *api.FlowState {
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
