package events

import (
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const PartitionPrefix = "partition"

var (
	PartitionKey = timebox.NewAggregateID(PartitionPrefix)

	PartitionAppliers = makePartitionAppliers()
)

// NewPartitionState creates an empty partition state with initialized maps
func NewPartitionState() *api.PartitionState {
	return &api.PartitionState{
		Health:      map[api.StepID]*api.HealthState{},
		Active:      map[api.FlowID]*api.ActiveFlow{},
		Deactivated: []*api.DeactivatedFlow{},
		Archiving:   map[api.FlowID]time.Time{},
		FlowDigests: map[api.FlowID]*api.FlowDigest{},
	}
}

// IsPartitionEvent returns true if the event is for the partition aggregate
func IsPartitionEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 1 && ev.AggregateID[0] == PartitionPrefix
}

func makePartitionAppliers() timebox.Appliers[*api.PartitionState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[*api.PartitionState]{
		api.EventTypeStepHealthChanged: timebox.MakeApplier(stepHealthChanged),
		api.EventTypeFlowActivated:     timebox.MakeApplier(flowActivated),
		api.EventTypeFlowDeactivated:   timebox.MakeApplier(flowDeactivated),
		api.EventTypeFlowArchiving:     timebox.MakeApplier(flowArchiving),
		api.EventTypeFlowArchived:      timebox.MakeApplier(flowArchived),
		api.EventTypeFlowDigestUpdated: timebox.MakeApplier(flowDigestUpdated),
	})
}

func stepHealthChanged(
	st *api.PartitionState, ev *timebox.Event,
	data api.StepHealthChangedEvent,
) *api.PartitionState {
	return st.
		SetHealth(data.StepID, &api.HealthState{
			Status: data.Status,
			Error:  data.Error,
		}).
		SetLastUpdated(ev.Timestamp)
}

func flowActivated(
	st *api.PartitionState, ev *timebox.Event, data api.FlowActivatedEvent,
) *api.PartitionState {
	digest := &api.FlowDigest{
		Status:    api.FlowActive,
		CreatedAt: ev.Timestamp,
		Labels:    data.Labels,
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
	st *api.PartitionState, ev *timebox.Event, data api.FlowDeactivatedEvent,
) *api.PartitionState {
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
	st *api.PartitionState, ev *timebox.Event, data api.FlowArchivingEvent,
) *api.PartitionState {
	return st.
		RemoveDeactivated(data.FlowID).
		AddArchiving(data.FlowID, ev.Timestamp).
		SetLastUpdated(ev.Timestamp)
}

func flowArchived(
	st *api.PartitionState, ev *timebox.Event, data api.FlowArchivedEvent,
) *api.PartitionState {
	return st.
		RemoveArchiving(data.FlowID).
		DeleteFlowDigest(data.FlowID).
		SetLastUpdated(ev.Timestamp)
}

func flowDigestUpdated(
	st *api.PartitionState, ev *timebox.Event, data api.FlowDigestUpdatedEvent,
) *api.PartitionState {
	digest := &api.FlowDigest{
		Status:      data.Status,
		CompletedAt: data.CompletedAt,
		Error:       data.Error,
	}

	if existing, ok := st.FlowDigests[data.FlowID]; ok {
		digest.CreatedAt = existing.CreatedAt
		digest.Labels = existing.Labels
	} else if active, ok := st.Active[data.FlowID]; ok {
		digest.CreatedAt = active.StartedAt
	}

	return st.
		SetFlowDigest(data.FlowID, digest).
		SetLastUpdated(ev.Timestamp)
}
