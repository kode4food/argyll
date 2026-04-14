package events_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestNewClusterState(t *testing.T) {
	st := events.NewClusterState()

	assert.NotNil(t, st)
	assert.NotNil(t, st.Nodes)
	assert.Empty(t, st.Nodes)
}

func TestIsClusterEvent(t *testing.T) {
	clusterEv := &timebox.Event{AggregateID: events.ClusterKey}
	flowEv := &timebox.Event{AggregateID: events.FlowKey("test-flow")}

	assert.True(t, events.IsClusterEvent(clusterEv))
	assert.False(t, events.IsClusterEvent(flowEv))
}

func TestIsClusterEventID(t *testing.T) {
	assert.True(t, events.IsClusterEventID(events.ClusterKey))
	assert.False(t, events.IsClusterEventID(events.FlowKey("test-flow")))
	assert.False(t, events.IsClusterEventID(events.CatalogKey))
}

func TestClusterStepHealthChanged(t *testing.T) {
	st := events.NewClusterState()

	data, err := json.Marshal(api.StepHealthChangedEvent{
		NodeID: "node-1",
		StepID: "test-step",
		Status: api.HealthHealthy,
	})
	assert.NoError(t, err)

	ts := time.Unix(1000, 0)
	ev := &timebox.Event{
		AggregateID: events.ClusterKey,
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Timestamp:   ts,
		Data:        data,
	}

	res := events.ClusterAppliers[ev.Type](st, ev)
	node, ok := res.Nodes[("node-1")]
	assert.True(t, ok)
	assert.Equal(t, api.HealthHealthy, node.Health["test-step"].Status)
	assert.Equal(t, ts, res.LastUpdated)
}

func TestClusterStepHealthChangedWithError(t *testing.T) {
	st := events.NewClusterState()

	data, err := json.Marshal(api.StepHealthChangedEvent{
		NodeID: "node-1",
		StepID: "test-step",
		Status: api.HealthUnhealthy,
		Error:  "connection refused",
	})
	assert.NoError(t, err)

	ev := &timebox.Event{
		AggregateID: events.ClusterKey,
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Data:        data,
	}

	res := events.ClusterAppliers[ev.Type](st, ev)
	node, ok := res.Nodes[("node-1")]
	assert.True(t, ok)
	assert.Equal(t, api.HealthUnhealthy, node.Health["test-step"].Status)
	assert.Equal(t, "connection refused", node.Health["test-step"].Error)
}

func TestClusterMultipleNodes(t *testing.T) {
	et := timebox.EventType(api.EventTypeStepHealthChanged)
	apply := events.ClusterAppliers[et]

	makeEv := func(
		nodeID, stepID string, status api.HealthStatus,
	) *timebox.Event {
		data, _ := json.Marshal(api.StepHealthChangedEvent{
			NodeID: api.NodeID(nodeID),
			StepID: api.StepID(stepID),
			Status: status,
		})
		return &timebox.Event{
			AggregateID: events.ClusterKey,
			Type:        timebox.EventType(api.EventTypeStepHealthChanged),
			Data:        data,
		}
	}

	st := events.NewClusterState()
	st = apply(st, makeEv("node-1", "step-a", api.HealthHealthy))
	st = apply(st, makeEv("node-2", "step-a", api.HealthUnhealthy))

	assert.Len(t, st.Nodes, 2)
	assert.Equal(t,
		api.HealthHealthy, st.Nodes["node-1"].Health["step-a"].Status,
	)
	assert.Equal(t,
		api.HealthUnhealthy, st.Nodes["node-2"].Health["step-a"].Status,
	)
}
