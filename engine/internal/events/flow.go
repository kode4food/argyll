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
		Executions: map[api.StepID]*api.ExecutionState{},
	}
}

// IsFlowEvent returns true if the event belongs to a flow aggregate
func IsFlowEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 2 && ev.AggregateID[0] == flowPrefix
}

func makeFlowAppliers() timebox.Appliers[*api.FlowState] {
	flowStartedEvent := timebox.EventType(api.EventTypeFlowStarted)
	flowStartedApplier := timebox.MakeApplier(flowStarted)
	flowCompletedEvent := timebox.EventType(api.EventTypeFlowCompleted)
	flowCompletedApplier := timebox.MakeApplier(flowCompleted)
	flowFailedEvent := timebox.EventType(api.EventTypeFlowFailed)
	flowFailedApplier := timebox.MakeApplier(flowFailed)
	stepStartedEvent := timebox.EventType(api.EventTypeStepStarted)
	stepStartedApplier := timebox.MakeApplier(stepStarted)
	stepCompletedEvent := timebox.EventType(api.EventTypeStepCompleted)
	stepCompletedApplier := timebox.MakeApplier(stepCompleted)
	stepFailedEvent := timebox.EventType(api.EventTypeStepFailed)
	stepFailedApplier := timebox.MakeApplier(stepFailed)
	stepSkippedEvent := timebox.EventType(api.EventTypeStepSkipped)
	stepSkippedApplier := timebox.MakeApplier(stepSkipped)
	attributeSetEvent := timebox.EventType(api.EventTypeAttributeSet)
	attributeSetApplier := timebox.MakeApplier(attributeSet)
	workItemStartedEvent := timebox.EventType(api.EventTypeWorkStarted)
	workItemStartedApplier := timebox.MakeApplier(workItemStarted)
	workItemCompletedEvent := timebox.EventType(api.EventTypeWorkCompleted)
	workItemCompletedApplier := timebox.MakeApplier(workItemCompleted)
	workItemFailedEvent := timebox.EventType(api.EventTypeWorkFailed)
	workItemFailedApplier := timebox.MakeApplier(workItemFailed)
	retryScheduledEvent := timebox.EventType(api.EventTypeRetryScheduled)
	retryScheduledApplier := timebox.MakeApplier(retryScheduled)

	return timebox.Appliers[*api.FlowState]{
		flowStartedEvent:      flowStartedApplier,
		flowCompletedEvent:    flowCompletedApplier,
		flowFailedEvent:       flowFailedApplier,
		stepStartedEvent:      stepStartedApplier,
		stepCompletedEvent:    stepCompletedApplier,
		stepFailedEvent:       stepFailedApplier,
		stepSkippedEvent:      stepSkippedApplier,
		attributeSetEvent:     attributeSetApplier,
		workItemStartedEvent:  workItemStartedApplier,
		workItemCompletedEvent: workItemCompletedApplier,
		workItemFailedEvent:   workItemFailedApplier,
		retryScheduledEvent:   retryScheduledApplier,
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
	workItems := map[api.Token]*api.WorkState{}
	for token, inputs := range data.WorkItems {
		workItems[token] = &api.WorkState{
			Status: api.WorkPending,
			Inputs: inputs,
		}
	}

	exec := &api.ExecutionState{
		Status:    api.StepPending,
		WorkItems: workItems,
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

func createExecutions(p *api.ExecutionPlan) map[api.StepID]*api.ExecutionState {
	exec := map[api.StepID]*api.ExecutionState{}
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
	st *api.FlowState, stepID api.StepID, fn string,
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
