package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

const enginePrefix = "engine"

var (
	EngineID = timebox.NewAggregateID(enginePrefix)

	EngineAppliers = makeEngineAppliers()
)

func NewEngineState() *api.EngineState {
	return &api.EngineState{
		Steps:           map[timebox.ID]*api.Step{},
		Health:          map[timebox.ID]*api.HealthState{},
		ActiveWorkflows: map[timebox.ID]*api.ActiveWorkflowInfo{},
	}
}

// IsEngineEvent returns true if the event is for the engine aggregate
func IsEngineEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 1 && ev.AggregateID[0] == enginePrefix
}

func makeEngineAppliers() timebox.Appliers[*api.EngineState] {
	stepRegisteredApplier := timebox.MakeApplier(stepRegistered)
	stepUnregisteredApplier := timebox.MakeApplier(stepUnregistered)
	stepHealthChangedApplier := timebox.MakeApplier(stepHealthChanged)
	workflowActivatedApplier := timebox.MakeApplier(workflowActivated)
	workflowDeactivatedApplier := timebox.MakeApplier(workflowDeactivated)

	return timebox.Appliers[*api.EngineState]{
		api.EventTypeStepRegistered:      stepRegisteredApplier,
		api.EventTypeStepUnregistered:    stepUnregisteredApplier,
		api.EventTypeStepHealthChanged:   stepHealthChangedApplier,
		api.EventTypeWorkflowActivated:   workflowActivatedApplier,
		api.EventTypeWorkflowDeactivated: workflowDeactivatedApplier,
	}
}

func stepRegistered(
	st *api.EngineState, ev *timebox.Event, data api.StepRegisteredEvent,
) *api.EngineState {
	return st.
		SetStep(data.Step.ID, data.Step).
		SetHealth(data.Step.ID, &api.HealthState{Status: api.HealthUnknown}).
		SetLastUpdated(ev.Timestamp)
}

func stepUnregistered(
	st *api.EngineState, ev *timebox.Event, data api.StepUnregisteredEvent,
) *api.EngineState {
	return st.
		DeleteStep(data.StepID).
		SetLastUpdated(ev.Timestamp)
}

func stepHealthChanged(
	st *api.EngineState, ev *timebox.Event, data api.StepHealthChangedEvent,
) *api.EngineState {
	return st.
		SetHealth(data.StepID, &api.HealthState{
			Status: data.Status,
			Error:  data.Error,
		}).
		SetLastUpdated(ev.Timestamp)
}

func workflowActivated(
	st *api.EngineState, ev *timebox.Event, data api.WorkflowActivatedEvent,
) *api.EngineState {
	return st.
		SetActiveWorkflow(data.FlowID, &api.ActiveWorkflowInfo{
			FlowID:     data.FlowID,
			StartedAt:  ev.Timestamp,
			LastActive: ev.Timestamp,
		}).
		SetLastUpdated(ev.Timestamp)
}

func workflowDeactivated(
	st *api.EngineState, ev *timebox.Event, data api.WorkflowDeactivatedEvent,
) *api.EngineState {
	return st.
		DeleteActiveWorkflow(data.FlowID).
		SetLastUpdated(ev.Timestamp)
}
