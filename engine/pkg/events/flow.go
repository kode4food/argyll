package events

import (
	"strings"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const FlowPrefix = "flow"

// FlowAppliers contains the event applier functions for flow events
var FlowAppliers = makeFlowAppliers()

// NewFlowState creates an empty flow state with initialized maps for
// attributes, step executions, and timeouts
func NewFlowState() *api.FlowState {
	return &api.FlowState{
		Attributes: api.AttributeValues{},
		Executions: api.Executions{},
	}
}

// FlowKey returns the aggregate ID for a flow
func FlowKey[T ~string](flowID T) timebox.AggregateID {
	return timebox.NewAggregateID(FlowPrefix, timebox.ID(flowID))
}

// FlowJoinKey is a JoinKeyFunc that co-locates parent and child flows in the
// same Redis hash slot. The root flow ID is wrapped in hash slot notation so
// that a parent and its children both resolve to {my-flow} and land in the
// same slot. Produces "flow:{my-flow}" or "flow:{my-flow}:step:token"
func FlowJoinKey(id timebox.AggregateID) string {
	if len(id) < 2 {
		return id.Join(":")
	}
	prefix := string(id[0])
	flowID := string(id[1])
	rootFlowID := flowID
	if before, _, ok := strings.Cut(flowID, ":"); ok {
		rootFlowID = before
	}
	if flowID == rootFlowID {
		return prefix + ":{" + rootFlowID + "}"
	}
	return prefix + ":{" + rootFlowID + "}:" + flowID[len(rootFlowID)+1:]
}

// FlowParseKey is the ParseKeyFunc that reverses FlowJoinKey
func FlowParseKey(str string) timebox.AggregateID {
	before, after, found := strings.Cut(str, ":{")
	if !found {
		return timebox.ParseKey(str)
	}
	slot, remaining, hasRemaining := strings.Cut(after, "}:")
	if !hasRemaining {
		slot = strings.TrimSuffix(after, "}")
		return timebox.AggregateID{timebox.ID(before), timebox.ID(slot)}
	}
	return timebox.AggregateID{
		timebox.ID(before),
		timebox.ID(slot + ":" + remaining),
	}
}

// IsFlowEvent returns true if the event belongs to a flow aggregate
func IsFlowEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 2 && ev.AggregateID[0] == FlowPrefix
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
	execs := createExecutions(data.Plan)

	attributes := api.AttributeValues{}
	for key, value := range data.Init {
		attributes[key] = &api.AttributeValue{
			Value: value,
			SetAt: ev.Timestamp,
		}
	}

	return &api.FlowState{
		ID:          data.FlowID,
		Status:      api.FlowActive,
		Plan:        data.Plan,
		Metadata:    data.Metadata,
		Labels:      data.Labels,
		Attributes:  attributes,
		Executions:  execs,
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
	return st.
		SetExecution(data.StepID,
			st.Executions[data.StepID].
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
	return st.
		SetExecution(data.StepID,
			st.Executions[data.StepID].
				SetStatus(api.StepFailed).
				SetError(data.Error).
				SetCompletedAt(ev.Timestamp),
		).
		SetLastUpdated(ev.Timestamp)
}

func stepSkipped(
	st *api.FlowState, ev *timebox.Event, data api.StepSkippedEvent,
) *api.FlowState {
	return st.
		SetExecution(data.StepID,
			st.Executions[data.StepID].
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
			SetAt: ev.Timestamp,
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

func workStarted(
	st *api.FlowState, ev *timebox.Event, data api.WorkStartedEvent,
) *api.FlowState {
	exec := st.Executions[data.StepID]
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
	exec := st.Executions[data.StepID]
	item, ok := exec.WorkItems[data.Token]
	if !ok {
		return st
	}

	item = item.
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
	exec := st.Executions[data.StepID]
	item, ok := exec.WorkItems[data.Token]
	if !ok {
		return st
	}

	item = item.
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
	exec := st.Executions[data.StepID]
	item, ok := exec.WorkItems[data.Token]
	if !ok {
		return st
	}

	item = item.
		SetStatus(api.WorkNotCompleted).
		SetCompletedAt(ev.Timestamp).
		SetError(data.Error)

	var updatedExec *api.ExecutionState
	if data.RetryToken != "" && data.RetryToken != data.Token {
		updatedExec = exec.
			RemoveWorkItem(data.Token).
			SetWorkItem(data.RetryToken, item)
	} else {
		updatedExec = exec.SetWorkItem(data.Token, item)
	}

	return st.
		SetExecution(data.StepID, updatedExec).
		SetLastUpdated(ev.Timestamp)
}

func retryScheduled(
	st *api.FlowState, ev *timebox.Event, data api.RetryScheduledEvent,
) *api.FlowState {
	exec := st.Executions[data.StepID]
	item, ok := exec.WorkItems[data.Token]
	if !ok {
		return st
	}

	item = item.
		SetStatus(api.WorkPending).
		SetRetryCount(data.RetryCount).
		SetNextRetryAt(data.NextRetryAt).
		SetError(data.Error)

	return st.
		SetExecution(data.StepID, exec.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}
