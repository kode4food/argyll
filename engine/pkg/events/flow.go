package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const (
	FlowPrefix          = "flow"
	FlowStatusActive    = "active"
	FlowStatusCompleted = "completed"
	FlowStatusFailed    = "failed"
)

// FlowAppliers contains the event applier functions for flow events
var FlowAppliers = makeFlowAppliers()

// NewFlowState creates an empty flow state with initialized maps for
// attributes, step executions, and timeouts
func NewFlowState() api.FlowState {
	return api.FlowState{
		Attributes: api.AttributeValues{},
		Executions: api.Executions{},
	}
}

// FlowKey returns the aggregate ID for a flow
func FlowKey[T ~string](flowID T) timebox.AggregateID {
	return timebox.NewAggregateID(FlowPrefix, timebox.ID(flowID))
}

func ParseFlowID(id timebox.AggregateID) (api.FlowID, bool) {
	if len(id) < 2 || id[0] != FlowPrefix {
		return "", false
	}
	fid := api.FlowID(id[1])
	if fid == "" {
		return "", false
	}
	return fid, true
}

func FlowIndexer(evs []*timebox.Event) []*timebox.Index {
	res := make([]*timebox.Index, 0, len(evs))

	handleStarted := func(data api.FlowStartedEvent) {
		status := FlowStatusActive
		res = append(res, &timebox.Index{
			Status: &status,
			Labels: data.Labels,
		})
	}

	handleDeactivated := func(data api.FlowDeactivatedEvent) {
		status := string(data.Status)
		res = append(res, &timebox.Index{
			Status: &status,
		})
	}

	for _, ev := range evs {
		switch api.EventType(ev.Type) {
		case api.EventTypeFlowStarted:
			data, err := timebox.GetEventValue[api.FlowStartedEvent](ev)
			if err == nil {
				handleStarted(data)
				continue
			}
			// slog this. very bad

		case api.EventTypeFlowDeactivated:
			data, err := timebox.GetEventValue[api.FlowDeactivatedEvent](ev)
			if err == nil {
				handleDeactivated(data)
				continue
			}
			// slog this. very bad
		}
	}
	return res
}

// IsFlowEvent returns true if the event belongs to a flow aggregate
func IsFlowEvent(ev *timebox.Event) bool {
	return IsFlowEventID(ev.AggregateID)
}

// IsFlowEventID returns true if the ID belongs to a flow aggregate
func IsFlowEventID(id timebox.AggregateID) bool {
	return len(id) == 2 && id[0] == FlowPrefix
}

func makeFlowAppliers() timebox.Appliers[api.FlowState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[api.FlowState]{
		api.EventTypeFlowStarted:      timebox.MakeApplier(flowStarted),
		api.EventTypeFlowCompleted:    timebox.MakeApplier(flowCompleted),
		api.EventTypeFlowDeactivated:  timebox.MakeApplier(flowDeactivated),
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
	_ api.FlowState, ev *timebox.Event, data api.FlowStartedEvent,
) api.FlowState {
	execs := createExecutions(data.Plan)

	attributes := api.AttributeValues{}
	for key, value := range data.Init {
		attributes[key] = &api.AttributeValue{
			Value: value,
			SetAt: ev.Timestamp,
		}
	}

	return api.FlowState{
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
	st api.FlowState, ev *timebox.Event, _ api.FlowCompletedEvent,
) api.FlowState {
	return st.
		SetStatus(api.FlowCompleted).
		SetCompletedAt(ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func flowFailed(
	st api.FlowState, ev *timebox.Event, data api.FlowFailedEvent,
) api.FlowState {
	return st.
		SetStatus(api.FlowFailed).
		SetError(data.Error).
		SetCompletedAt(ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func flowDeactivated(
	st api.FlowState, ev *timebox.Event, _ api.FlowDeactivatedEvent,
) api.FlowState {
	return st.
		SetDeactivatedAt(ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func stepStarted(
	st api.FlowState, ev *timebox.Event, data api.StepStartedEvent,
) api.FlowState {
	workItems := api.WorkItems{}
	for tkn, inputs := range data.WorkItems {
		workItems[tkn] = api.WorkState{
			Status: api.WorkPending,
			Inputs: inputs,
		}
	}

	ex := api.ExecutionState{
		Status:    api.StepPending,
		WorkItems: workItems,
	}

	updated := ex.
		SetStatus(api.StepActive).
		SetStartedAt(ev.Timestamp).
		SetInputs(data.Inputs)

	return st.
		SetExecution(data.StepID, updated).
		SetLastUpdated(ev.Timestamp)
}

func stepCompleted(
	st api.FlowState, ev *timebox.Event, data api.StepCompletedEvent,
) api.FlowState {
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
	st api.FlowState, ev *timebox.Event, data api.StepFailedEvent,
) api.FlowState {
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
	st api.FlowState, ev *timebox.Event, data api.StepSkippedEvent,
) api.FlowState {
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
	st api.FlowState, ev *timebox.Event, data api.AttributeSetEvent,
) api.FlowState {
	return st.
		SetAttribute(data.Key, &api.AttributeValue{
			Value: data.Value,
			Step:  data.StepID,
			SetAt: ev.Timestamp,
		}).
		SetLastUpdated(ev.Timestamp)
}

func createExecutions(p *api.ExecutionPlan) api.Executions {
	ex := api.Executions{}
	for sid := range p.Steps {
		ex[sid] = api.ExecutionState{
			Status:    api.StepPending,
			WorkItems: api.WorkItems{},
		}
	}
	return ex
}

func workStarted(
	st api.FlowState, ev *timebox.Event, data api.WorkStartedEvent,
) api.FlowState {
	ex := st.Executions[data.StepID]
	work := ex.WorkItems[data.Token].
		SetStatus(api.WorkActive).
		SetNodeID(data.NodeID)

	if work.StartedAt.IsZero() {
		work = work.SetStartedAt(ev.Timestamp)
	}

	return st.
		SetExecution(data.StepID, ex.SetWorkItem(data.Token, work)).
		SetLastUpdated(ev.Timestamp)
}

func workSucceeded(
	st api.FlowState, ev *timebox.Event, data api.WorkSucceededEvent,
) api.FlowState {
	ex := st.Executions[data.StepID]
	item, ok := ex.WorkItems[data.Token]
	if !ok {
		return st
	}

	item = item.
		SetStatus(api.WorkSucceeded).
		SetCompletedAt(ev.Timestamp).
		SetOutputs(data.Outputs)

	return st.
		SetExecution(data.StepID, ex.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}

func workFailed(
	st api.FlowState, ev *timebox.Event, data api.WorkFailedEvent,
) api.FlowState {
	ex := st.Executions[data.StepID]
	item, ok := ex.WorkItems[data.Token]
	if !ok {
		return st
	}

	item = item.
		SetStatus(api.WorkFailed).
		SetCompletedAt(ev.Timestamp).
		SetError(data.Error)

	return st.
		SetExecution(data.StepID, ex.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}

func workNotCompleted(
	st api.FlowState, ev *timebox.Event, data api.WorkNotCompletedEvent,
) api.FlowState {
	ex := st.Executions[data.StepID]
	item, ok := ex.WorkItems[data.Token]
	if !ok {
		return st
	}

	item = item.
		SetStatus(api.WorkNotCompleted).
		SetCompletedAt(ev.Timestamp).
		SetError(data.Error)

	var updatedExec api.ExecutionState
	if data.RetryToken != "" && data.RetryToken != data.Token {
		updatedExec = ex.
			RemoveWorkItem(data.Token).
			SetWorkItem(data.RetryToken, item)
	} else {
		updatedExec = ex.SetWorkItem(data.Token, item)
	}

	return st.
		SetExecution(data.StepID, updatedExec).
		SetLastUpdated(ev.Timestamp)
}

func retryScheduled(
	st api.FlowState, ev *timebox.Event, data api.RetryScheduledEvent,
) api.FlowState {
	ex := st.Executions[data.StepID]
	item, ok := ex.WorkItems[data.Token]
	if !ok {
		return st
	}

	item = item.
		SetStatus(api.WorkPending).
		SetNodeID("").
		SetRetryCount(data.RetryCount).
		SetNextRetryAt(data.NextRetryAt).
		SetError(data.Error)

	return st.
		SetExecution(data.StepID, ex.SetWorkItem(data.Token, item)).
		SetLastUpdated(ev.Timestamp)
}
