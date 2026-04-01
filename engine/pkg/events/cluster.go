package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const ClusterPrefix = "cluster"

var (
	ClusterKey = timebox.NewAggregateID(ClusterPrefix)

	ClusterAppliers = makeClusterAppliers()
)

// NewClusterState creates an empty cluster state with initialized maps
func NewClusterState() *api.ClusterState {
	return &api.ClusterState{
		Nodes: map[api.NodeID]*api.NodeState{},
	}
}

// IsClusterEvent returns true if the event is for the cluster aggregate
func IsClusterEvent(ev *timebox.Event) bool {
	return IsClusterEventID(ev.AggregateID)
}

// IsClusterEventID returns true if the ID is for the cluster aggregate
func IsClusterEventID(id timebox.AggregateID) bool {
	return len(id) == 1 && id[0] == ClusterPrefix
}

func makeClusterAppliers() timebox.Appliers[*api.ClusterState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[*api.ClusterState]{
		api.EventTypeStepHealthChanged: timebox.MakeApplier(stepHealthChanged),
	})
}

func stepHealthChanged(
	st *api.ClusterState, ev *timebox.Event, data api.StepHealthChangedEvent,
) *api.ClusterState {
	node := st.Nodes[data.NodeID]
	if node == nil {
		node = &api.NodeState{Health: map[api.StepID]*api.HealthState{}}
	}
	node = node.
		SetLastSeen(ev.Timestamp).
		SetHealth(data.StepID, &api.HealthState{
			Status: data.Status,
			Error:  data.Error,
		})

	return st.
		SetNode(data.NodeID, node).
		SetLastUpdated(ev.Timestamp)
}
