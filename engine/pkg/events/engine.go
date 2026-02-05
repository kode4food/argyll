package events

import (
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const EnginePrefix = "engine"

var (
	EngineID = timebox.NewAggregateID(EnginePrefix)

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
		Archiving:   map[api.FlowID]time.Time{},
		FlowDigests: map[api.FlowID]*api.FlowDigest{},
		Attributes:  api.AttributeGraph{},
	}
}

// IsEngineEvent returns true if the event is for the engine aggregate
func IsEngineEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 1 && ev.AggregateID[0] == EnginePrefix
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
		api.EventTypeFlowDigestUpdated: timebox.MakeApplier(flowDigestUpdated),
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
	digest := &api.FlowDigest{
		Status:    api.FlowActive,
		CreatedAt: ev.Timestamp,
	}
	return st.
		SetActiveFlow(data.FlowID, &api.ActiveFlow{
			ParentFlowID: data.ParentFlowID,
			StartedAt:    ev.Timestamp,
			LastActive:   ev.Timestamp,
		}).
		SetFlowDigest(data.FlowID, digest).
		SetLastUpdated(ev.Timestamp)
}

func flowDeactivated(
	st *api.EngineState, ev *timebox.Event, data api.FlowDeactivatedEvent,
) *api.EngineState {
	var parentID api.FlowID
	if active, ok := st.Active[data.FlowID]; ok {
		parentID = active.ParentFlowID
	}
	return st.
		DeleteActiveFlow(data.FlowID).
		AddDeactivated(&api.DeactivatedFlow{
			FlowID:        data.FlowID,
			ParentFlowID:  parentID,
			DeactivatedAt: ev.Timestamp,
		}).
		SetLastUpdated(ev.Timestamp)
}

func flowArchiving(
	st *api.EngineState, ev *timebox.Event, data api.FlowArchivingEvent,
) *api.EngineState {
	return st.
		RemoveDeactivated(data.FlowID).
		AddArchiving(data.FlowID, ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func flowArchived(
	st *api.EngineState, ev *timebox.Event, data api.FlowArchivedEvent,
) *api.EngineState {
	return st.
		RemoveArchiving(data.FlowID).
		DeleteFlowDigest(data.FlowID).
		SetLastUpdated(ev.Timestamp)
}

func flowDigestUpdated(
	st *api.EngineState, ev *timebox.Event, data api.FlowDigestUpdatedEvent,
) *api.EngineState {
	digest := &api.FlowDigest{
		Status:      data.Status,
		CompletedAt: data.CompletedAt,
		Error:       data.Error,
	}

	if existing, ok := st.FlowDigests[data.FlowID]; ok {
		digest.CreatedAt = existing.CreatedAt
	} else if active, ok := st.Active[data.FlowID]; ok {
		digest.CreatedAt = active.StartedAt
	}

	return st.
		SetFlowDigest(data.FlowID, digest).
		SetLastUpdated(ev.Timestamp)
}
