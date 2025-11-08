package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

const workflowPrefix = "workflow"

var WorkflowAppliers = timebox.Appliers[*api.WorkflowState]{
	api.EventTypeWorkflowStarted:   workflowStarted,
	api.EventTypeWorkflowCompleted: workflowCompleted,
	api.EventTypeWorkflowFailed:    workflowFailed,
	api.EventTypeStepStarted:       stepStarted,
	api.EventTypeStepCompleted:     stepCompleted,
	api.EventTypeStepFailed:        stepFailed,
	api.EventTypeStepSkipped:       stepSkipped,
	api.EventTypeAttributeSet:      attributeSet,
	api.EventTypeWorkStarted:       workItemStarted,
	api.EventTypeWorkCompleted:     workItemCompleted,
	api.EventTypeWorkFailed:        workItemFailed,
	api.EventTypeRetryScheduled:    retryScheduled,
}

func NewWorkflowState() *api.WorkflowState {
	return &api.WorkflowState{
		Attributes: map[api.Name]*api.AttributeValue{},
		Executions: map[timebox.ID]*api.ExecutionState{},
	}
}

func IsWorkflowEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 2 && ev.AggregateID[0] == workflowPrefix
}

func workflowStarted(
	st *api.WorkflowState, ev *timebox.Event,
) *api.WorkflowState {
	var ws api.WorkflowStartedEvent
	if err := json.Unmarshal(ev.Data, &ws); err != nil {
		return st
	}

	exec := createExecutions(ws.ExecutionPlan)

	attributes := map[api.Name]*api.AttributeValue{}
	for key, value := range ws.InitialState {
		attributes[key] = &api.AttributeValue{Value: value}
	}

	return &api.WorkflowState{
		ID:            ws.WorkflowID,
		Status:        api.WorkflowActive,
		ExecutionPlan: ws.ExecutionPlan,
		Attributes:    attributes,
		Executions:    exec,
		CreatedAt:     ev.Timestamp,
		LastUpdated:   ev.Timestamp,
	}
}

func workflowCompleted(
	st *api.WorkflowState, ev *timebox.Event,
) *api.WorkflowState {
	return st.
		SetStatus(api.WorkflowCompleted).
		SetCompletedAt(ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func workflowFailed(
	st *api.WorkflowState, ev *timebox.Event,
) *api.WorkflowState {
	var wf api.WorkflowFailedEvent
	if err := json.Unmarshal(ev.Data, &wf); err != nil {
		return st
	}
	return st.
		SetStatus(api.WorkflowFailed).
		SetError(wf.Error).
		SetCompletedAt(ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func stepStarted(st *api.WorkflowState, ev *timebox.Event) *api.WorkflowState {
	var ss api.StepStartedEvent
	if err := json.Unmarshal(ev.Data, &ss); err != nil {
		return st
	}

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
		SetInputs(ss.Inputs)

	return st.
		SetExecution(ss.StepID, updated).
		SetLastUpdated(ev.Timestamp)
}

func stepCompleted(
	st *api.WorkflowState, ev *timebox.Event,
) *api.WorkflowState {
	var sc api.StepCompletedEvent
	if err := json.Unmarshal(ev.Data, &sc); err != nil {
		return st
	}
	exec := getExecution(st, sc.StepID, "stepCompleted")

	updated := exec.
		SetStatus(api.StepCompleted).
		SetCompletedAt(ev.Timestamp).
		SetDuration(sc.Duration).
		SetOutputs(sc.Outputs)

	return st.
		SetExecution(sc.StepID, updated).
		SetLastUpdated(ev.Timestamp)
}

func stepFailed(st *api.WorkflowState, ev *timebox.Event) *api.WorkflowState {
	var sf api.StepFailedEvent
	if err := json.Unmarshal(ev.Data, &sf); err != nil {
		return st
	}
	exec := getExecution(st, sf.StepID, "stepFailed")

	return st.
		SetExecution(sf.StepID,
			exec.
				SetStatus(api.StepFailed).
				SetError(sf.Error).
				SetCompletedAt(ev.Timestamp),
		).
		SetLastUpdated(ev.Timestamp)
}

func stepSkipped(st *api.WorkflowState, ev *timebox.Event) *api.WorkflowState {
	var ss api.StepSkippedEvent
	if err := json.Unmarshal(ev.Data, &ss); err != nil {
		return st
	}
	exec := getExecution(st, ss.StepID, "stepSkipped")

	return st.
		SetExecution(ss.StepID,
			exec.
				SetStatus(api.StepSkipped).
				SetError(ss.Reason).
				SetCompletedAt(ev.Timestamp),
		).
		SetLastUpdated(ev.Timestamp)
}

func attributeSet(st *api.WorkflowState, ev *timebox.Event) *api.WorkflowState {
	var as api.AttributeSetEvent
	if err := json.Unmarshal(ev.Data, &as); err != nil {
		return st
	}

	return st.
		SetAttribute(as.Key, &api.AttributeValue{
			Value: as.Value,
			Step:  as.StepID,
		}).
		SetLastUpdated(ev.Timestamp)
}

func createExecutions(p *api.ExecutionPlan) map[timebox.ID]*api.ExecutionState {
	exec := map[timebox.ID]*api.ExecutionState{}
	if p == nil {
		return exec
	}

	for _, step := range p.Steps {
		exec[step.ID] = &api.ExecutionState{
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

func workItemStarted(st *api.WorkflowState, ev *timebox.Event) *api.WorkflowState {
	var wi api.WorkStartedEvent
	if err := json.Unmarshal(ev.Data, &wi); err != nil {
		return st
	}

	exec := getExecution(st, wi.StepID, "workItemStarted")

	item := &api.WorkState{
		Status:    api.WorkActive,
		StartedAt: ev.Timestamp,
		Inputs:    wi.Inputs,
		Outputs:   api.Args{},
	}

	return st.
		SetExecution(wi.StepID, exec.SetWorkItem(wi.Token, item)).
		SetLastUpdated(ev.Timestamp)
}

func workItemCompleted(st *api.WorkflowState, ev *timebox.Event) *api.WorkflowState {
	var wi api.WorkCompletedEvent
	if err := json.Unmarshal(ev.Data, &wi); err != nil {
		return st
	}

	exec := getExecution(st, wi.StepID, "workItemCompleted")

	if exec.WorkItems == nil || exec.WorkItems[wi.Token] == nil {
		return st
	}

	item := exec.WorkItems[wi.Token]
	updated := &api.WorkState{
		Status:      api.WorkCompleted,
		StartedAt:   item.StartedAt,
		CompletedAt: ev.Timestamp,
		Inputs:      item.Inputs,
		Outputs:     wi.Outputs,
	}

	return st.
		SetExecution(wi.StepID, exec.SetWorkItem(wi.Token, updated)).
		SetLastUpdated(ev.Timestamp)
}

func workItemFailed(st *api.WorkflowState, ev *timebox.Event) *api.WorkflowState {
	var wi api.WorkFailedEvent
	if err := json.Unmarshal(ev.Data, &wi); err != nil {
		return st
	}

	exec := getExecution(st, wi.StepID, "workItemFailed")

	if exec.WorkItems == nil || exec.WorkItems[wi.Token] == nil {
		return st
	}

	item := exec.WorkItems[wi.Token]
	updated := &api.WorkState{
		Status:      api.WorkFailed,
		StartedAt:   item.StartedAt,
		CompletedAt: ev.Timestamp,
		Inputs:      item.Inputs,
		Error:       wi.Error,
	}

	return st.
		SetExecution(wi.StepID, exec.SetWorkItem(wi.Token, updated)).
		SetLastUpdated(ev.Timestamp)
}

func retryScheduled(
	st *api.WorkflowState, ev *timebox.Event,
) *api.WorkflowState {
	var rs api.RetryScheduledEvent
	if err := json.Unmarshal(ev.Data, &rs); err != nil {
		return st
	}

	exec := getExecution(st, rs.StepID, "retryScheduled")

	if exec.WorkItems == nil || exec.WorkItems[rs.Token] == nil {
		return st
	}

	item := exec.WorkItems[rs.Token]
	updated := item.
		SetStatus(api.WorkPending).
		SetRetryCount(rs.RetryCount).
		SetNextRetryAt(rs.NextRetryAt).
		SetLastError(rs.Error)

	return st.
		SetExecution(rs.StepID, exec.SetWorkItem(rs.Token, updated)).
		SetLastUpdated(ev.Timestamp)
}
