package events

import (
	"errors"
	"fmt"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

const flowPrefix = "flow"

var (
	ErrExecutionNotFound = errors.New("execution does not exist for step")
)

// FlowAppliers contains the event applier functions for flow events
var FlowAppliers = makeFlowAppliers()

// NewFlowState creates an empty flow state with initialized maps for
// attributes and step executions
func NewFlowState() *api.FlowState {
	return &api.FlowState{
		Attributes: api.AttributeValues{},
		Executions: api.Executions{},
	}
}

// IsFlowEvent returns true if the event belongs to a flow aggregate
func IsFlowEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 2 && ev.AggregateID[0] == flowPrefix
}

func makeFlowAppliers() timebox.Appliers[*api.FlowState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[*api.FlowState]{
		api.EventTypeFlowStarted:      timebox.MakeApplier(flowStarted),
		api.EventTypeFlowCompleted:    timebox.MakeApplier(flowCompleted),
		api.EventTypeFlowFailed:       timebox.MakeApplier(flowFailed),
		api.EventTypeStepStarted:      timebox.MakeApplier(stepStarted),
		api.EventTypeStepCompleted:    timebox.MakeApplier(stepCompleted),
		api.EventTypeStepFailed:       timebox.MakeApplier(stepFailed),
		api.EventTypeStepSkipped:      timebox.MakeApplier(stepSkipped),
		api.EventTypeAttributeSet:     timebox.MakeApplier(attributeSet),
		api.EventTypeWorkStarted:      timebox.MakeApplier(workStarted),
		api.EventTypeWorkSucceeded:    timebox.MakeApplier(workSucceeded),
		api.EventTypeWorkFailed:       timebox.MakeApplier(workFailed),
		api.EventTypeWorkNotCompleted: timebox.MakeApplier(workNotCompleted),
		api.EventTypeRetryScheduled:   timebox.MakeApplier(retryScheduled),
	})
}

func flowStarted(
	_ *api.FlowState, ev *timebox.Event, data api.FlowStartedEvent,
) *api.FlowState {
	exec := createExecutions(data.Plan)

	attributes := api.AttributeValues{}
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
	workItems := api.WorkItems{}
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
	exec := getExecution(st, data.StepID)

	return st.
		SetExecution(data.StepID,
			exec.
				SetStatus(api.StepCompleted).
				SetCompletedAt(ev.Timestamp).
				SetDuration(data.Duration).
				SetOutputs(data.Outputs),
		).
		SetLastUpdated(ev.Timestamp)
}

func stepFailed(
	st *api.FlowState, ev *timebox.Event, data api.StepFailedEvent,
) *api.FlowState {
	exec := getExecution(st, data.StepID)
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
	exec := getExecution(st, data.StepID)
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

func createExecutions(p *api.ExecutionPlan) api.Executions {
	exec := api.Executions{}
	for stepID := range p.Steps {
		exec[stepID] = &api.ExecutionState{
			Status:    api.StepPending,
			WorkItems: api.WorkItems{},
		}
	}
	return exec
}

func getExecution(st *api.FlowState, stepID api.StepID) *api.ExecutionState {
	if exec, ok := st.Executions[stepID]; ok {
		return exec
	}
	panic(
		fmt.Errorf("%w: %s", ErrExecutionNotFound, stepID),
	)
}

func workStarted(
	st *api.FlowState, ev *timebox.Event, data api.WorkStartedEvent,
) *api.FlowState {
	exec := getExecution(st, data.StepID)
	item := exec.WorkItems[data.Token].SetStatus(api.WorkActive)

	if item.StartedAt.IsZero() {
		item = item.SetStartedAt(ev.Timestamp)
	}

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}

func workSucceeded(
	st *api.FlowState, ev *timebox.Event, data api.WorkSucceededEvent,
) *api.FlowState {
	exec := getExecution(st, data.StepID)
	if exec.WorkItems == nil || exec.WorkItems[data.Token] == nil {
		return st
	}

	item := exec.WorkItems[data.Token].
		SetStatus(api.WorkSucceeded).
		SetCompletedAt(ev.Timestamp).
		SetOutputs(data.Outputs)

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}

func workFailed(
	st *api.FlowState, ev *timebox.Event, data api.WorkFailedEvent,
) *api.FlowState {
	exec := getExecution(st, data.StepID)
	if exec.WorkItems == nil || exec.WorkItems[data.Token] == nil {
		return st
	}

	item := exec.WorkItems[data.Token].
		SetStatus(api.WorkFailed).
		SetCompletedAt(ev.Timestamp).
		SetError(data.Error)

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}

func workNotCompleted(
	st *api.FlowState, ev *timebox.Event, data api.WorkNotCompletedEvent,
) *api.FlowState {
	exec := getExecution(st, data.StepID)
	if exec.WorkItems == nil || exec.WorkItems[data.Token] == nil {
		return st
	}

	item := exec.WorkItems[data.Token].
		SetStatus(api.WorkNotCompleted).
		SetCompletedAt(ev.Timestamp).
		SetError(data.Error)

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}

func retryScheduled(
	st *api.FlowState, ev *timebox.Event, data api.RetryScheduledEvent,
) *api.FlowState {
	exec := getExecution(st, data.StepID)
	if exec.WorkItems == nil || exec.WorkItems[data.Token] == nil {
		return st
	}

	item := exec.WorkItems[data.Token].
		SetStatus(api.WorkPending).
		SetRetryCount(data.RetryCount).
		SetNextRetryAt(data.NextRetryAt).
		SetError(data.Error)

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}
