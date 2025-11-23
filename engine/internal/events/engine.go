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

// NewEngineState creates an empty engine state with initialized maps for
// steps, health status, and active flows
func NewEngineState() *api.EngineState {
	return &api.EngineState{
		Steps:       map[api.StepID]*api.Step{},
		Health:      map[api.StepID]*api.HealthState{},
		ActiveFlows: map[api.FlowID]*api.ActiveFlowInfo{},
	}
}

// IsEngineEvent returns true if the event is for the engine aggregate
func IsEngineEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 1 && ev.AggregateID[0] == enginePrefix
}

func makeEngineAppliers() timebox.Appliers[*api.EngineState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[*api.EngineState]{
		api.EventTypeStepRegistered:    timebox.MakeApplier(stepRegistered),
		api.EventTypeStepUnregistered:  timebox.MakeApplier(stepUnregistered),
		api.EventTypeStepHealthChanged: timebox.MakeApplier(stepHealthChanged),
		api.EventTypeFlowActivated:     timebox.MakeApplier(flowActivated),
		api.EventTypeFlowDeactivated:   timebox.MakeApplier(flowDeactivated),
	})
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

func flowActivated(
	st *api.EngineState, ev *timebox.Event, data api.FlowActivatedEvent,
) *api.EngineState {
	return st.
		SetActiveFlow(data.FlowID, &api.ActiveFlowInfo{
			FlowID:     data.FlowID,
			StartedAt:  ev.Timestamp,
			LastActive: ev.Timestamp,
		}).
		SetLastUpdated(ev.Timestamp)
}

func flowDeactivated(
	st *api.EngineState, ev *timebox.Event, data api.FlowDeactivatedEvent,
) *api.EngineState {
	return st.
		DeleteActiveFlow(data.FlowID).
		SetLastUpdated(ev.Timestamp)
}
