package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const PartitionPrefix = "partition"

var (
	PartitionAppliers = makePartitionAppliers()
)

// NewPartitionState creates an empty partition state with initialized maps
func NewPartitionState() *api.PartitionState {
	return &api.PartitionState{
		Health: map[api.StepID]*api.HealthState{},
	}
}

// PartitionKey returns the partition aggregate key for a specific raft node.
func PartitionKey(nodeID string) timebox.AggregateID {
	return timebox.NewAggregateID(PartitionPrefix, timebox.ID(nodeID))
}

// IsPartitionEvent returns true if the event is for the partition aggregate
func IsPartitionEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 1 && ev.AggregateID[0] == PartitionPrefix
}

func makePartitionAppliers() timebox.Appliers[*api.PartitionState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[*api.PartitionState]{
		api.EventTypeStepHealthChanged: timebox.MakeApplier(stepHealthChanged),
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
