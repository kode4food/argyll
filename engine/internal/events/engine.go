package events

import (
	"encoding/json"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

const enginePrefix = "engine"

var (
	EngineID = timebox.NewAggregateID(enginePrefix)

	EngineAppliers = timebox.Appliers[*api.EngineState]{
		api.EventTypeStepRegistered:    stepRegistered,
		api.EventTypeStepUnregistered:  stepUnregistered,
		api.EventTypeStepHealthChanged: stepHealthChanged,
		api.EventTypeWorkflowStarted:   engineWorkflowStarted,
		api.EventTypeWorkflowCompleted: engineWorkflowCompleted,
		api.EventTypeWorkflowFailed:    engineWorkflowFailed,
	}
)

func NewEngineState() *api.EngineState {
	return &api.EngineState{
		Steps:           map[timebox.ID]*api.Step{},
		Health:          map[timebox.ID]*api.HealthState{},
		ActiveWorkflows: map[timebox.ID]*api.ActiveWorkflowInfo{},
	}
}

func IsEngineEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 1 && ev.AggregateID[0] == enginePrefix
}

func stepRegistered(st *api.EngineState, ev *timebox.Event) *api.EngineState {
	var sr api.StepRegisteredEvent
	if err := json.Unmarshal(ev.Data, &sr); err != nil {
		return st
	}
	return st.
		SetStep(sr.Step.ID, sr.Step).
		SetHealth(sr.Step.ID, &api.HealthState{Status: api.HealthUnknown}).
		SetLastUpdated(ev.Timestamp)
}

func stepUnregistered(st *api.EngineState, ev *timebox.Event) *api.EngineState {
	var su api.StepUnregisteredEvent
	if err := json.Unmarshal(ev.Data, &su); err != nil {
		return st
	}
	return st.
		DeleteStep(su.StepID).
		SetLastUpdated(ev.Timestamp)
}

func stepHealthChanged(
	st *api.EngineState, ev *timebox.Event,
) *api.EngineState {
	var hc api.StepHealthChangedEvent
	if err := json.Unmarshal(ev.Data, &hc); err != nil {
		return st
	}
	return st.
		SetHealth(hc.StepID, &api.HealthState{
			Status: hc.Health,
			Error:  hc.HealthError,
		}).
		SetLastUpdated(ev.Timestamp)
}

func engineWorkflowStarted(
	st *api.EngineState, ev *timebox.Event,
) *api.EngineState {
	var ws api.WorkflowStartedEvent
	if err := json.Unmarshal(ev.Data, &ws); err != nil {
		return st
	}

	return st.
		SetActiveWorkflow(ws.FlowID, &api.ActiveWorkflowInfo{
			FlowID:     ws.FlowID,
			StartedAt:  ev.Timestamp,
			LastActive: ev.Timestamp,
		}).
		SetLastUpdated(ev.Timestamp)
}

func engineWorkflowCompleted(
	st *api.EngineState, ev *timebox.Event,
) *api.EngineState {
	var wc api.WorkflowCompletedEvent
	if err := json.Unmarshal(ev.Data, &wc); err != nil {
		return st
	}

	return st.
		DeleteActiveWorkflow(wc.FlowID).
		SetLastUpdated(ev.Timestamp)
}

func engineWorkflowFailed(
	st *api.EngineState, ev *timebox.Event,
) *api.EngineState {
	var wf api.WorkflowFailedEvent
	if err := json.Unmarshal(ev.Data, &wf); err != nil {
		return st
	}

	return st.
		DeleteActiveWorkflow(wf.FlowID).
		SetLastUpdated(ev.Timestamp)
}
