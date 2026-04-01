package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const NodePrefix = "node"

var NodeAppliers = makeNodeAppliers()

// NewNodeState creates an empty node state with initialized maps
func NewNodeState() *api.NodeState {
	return &api.NodeState{
		Health: map[api.StepID]*api.HealthState{},
	}
}

// NodeKey returns the node aggregate key for a specific raft node
func NodeKey(nodeID api.NodeID) timebox.AggregateID {
	return timebox.NewAggregateID(NodePrefix, timebox.ID(nodeID))
}

// IsNodeEvent returns true if the event is for a node aggregate
func IsNodeEvent(ev *timebox.Event) bool {
	return IsNodeEventID(ev.AggregateID)
}

// IsNodeEventID returns true if the ID is for a node aggregate
func IsNodeEventID(id timebox.AggregateID) bool {
	return len(id) == 2 && id[0] == NodePrefix
}

func makeNodeAppliers() timebox.Appliers[*api.NodeState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[*api.NodeState]{
		api.EventTypeStepHealthChanged: timebox.MakeApplier(stepHealthChanged),
	})
}

func stepHealthChanged(
	st *api.NodeState, ev *timebox.Event, data api.StepHealthChangedEvent,
) *api.NodeState {
	return st.
		SetID(data.NodeID).
		SetLastSeen(ev.Timestamp).
		SetHealth(data.StepID, &api.HealthState{
			Status: data.Status,
			Error:  data.Error,
		})
}
