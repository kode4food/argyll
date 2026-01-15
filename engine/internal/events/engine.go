package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const enginePrefix = "engine"

var (
	EngineID = timebox.NewAggregateID(enginePrefix)

	EngineAppliers = makeEngineAppliers()
)

// NewEngineState creates an empty engine state with initialized maps for
// steps, health status, active flows, and deactivated flows
func NewEngineState() *api.EngineState {
	return &api.EngineState{
		Steps:       api.Steps{},
		Health:      map[api.StepID]*api.HealthState{},
		Active:      map[api.FlowID]*api.ActiveFlow{},
		Deactivated: []*api.DeactivatedFlow{},
		Attributes:  api.AttributeGraph{},
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
		api.EventTypeStepUpdated:       timebox.MakeApplier(stepUpdated),
		api.EventTypeStepHealthChanged: timebox.MakeApplier(stepHealthChanged),
		api.EventTypeFlowActivated:     timebox.MakeApplier(flowActivated),
		api.EventTypeFlowDeactivated:   timebox.MakeApplier(flowDeactivated),
		api.EventTypeFlowArchiving:     timebox.MakeApplier(flowArchiving),
		api.EventTypeFlowArchived:      timebox.MakeApplier(flowArchived),
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

func stepUpdated(
	st *api.EngineState, ev *timebox.Event, data api.StepUpdatedEvent,
) *api.EngineState {
	return st.
		SetStep(data.Step.ID, data.Step).
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
		SetActiveFlow(data.FlowID, &api.ActiveFlow{
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
		AddDeactivated(&api.DeactivatedFlow{
			FlowID:        data.FlowID,
			DeactivatedAt: ev.Timestamp,
		}).
		SetLastUpdated(ev.Timestamp)
}

func flowArchiving(
	st *api.EngineState, ev *timebox.Event, data api.FlowArchivingEvent,
) *api.EngineState {
	return st.
		RemoveDeactivated(data.FlowID).
		SetLastUpdated(ev.Timestamp)
}

func flowArchived(
	st *api.EngineState, ev *timebox.Event, data api.FlowArchivedEvent,
) *api.EngineState {
	return st.
		SetLastUpdated(ev.Timestamp)
}
