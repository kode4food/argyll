package events_test

import (
	"encoding/json"
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestNewNodeState(t *testing.T) {
	st := events.NewNodeState()

	assert.NotNil(t, st)
	assert.NotNil(t, st.Health)
	assert.Empty(t, st.Health)
}

func TestIsNodeEvent(t *testing.T) {
	nodeEv := &timebox.Event{
		AggregateID: events.NodeKey("node-1"),
	}
	shortEv := &timebox.Event{
		AggregateID: timebox.NewAggregateID(events.NodePrefix),
	}
	longEv := &timebox.Event{
		AggregateID: timebox.NewAggregateID(
			events.NodePrefix, "node-1", "extra",
		),
	}
	flowEv := &timebox.Event{AggregateID: events.FlowKey("test-flow")}

	assert.True(t, events.IsNodeEvent(nodeEv))
	assert.False(t, events.IsNodeEvent(shortEv))
	assert.False(t, events.IsNodeEvent(longEv))
	assert.False(t, events.IsNodeEvent(flowEv))
}

func TestIsNodeEventID(t *testing.T) {
	assert.True(t, events.IsNodeEventID(events.NodeKey("node-1")))
	assert.False(t, events.IsNodeEventID(
		timebox.NewAggregateID(events.NodePrefix),
	))
	assert.False(t, events.IsNodeEventID(
		timebox.NewAggregateID(events.NodePrefix, "node-1", "extra"),
	))
	assert.False(t, events.IsNodeEventID(events.FlowKey("test-flow")))
}

func TestStepHealthChanged(t *testing.T) {
	st := events.NewNodeState().
		SetHealth("test-step", &api.HealthState{Status: api.HealthUnknown})

	data, err := json.Marshal(api.StepHealthChangedEvent{
		NodeID: "node-1",
		StepID: "test-step",
		Status: api.HealthHealthy,
	})
	assert.NoError(t, err)

	ev := &timebox.Event{
		AggregateID: events.NodeKey("node-1"),
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Data:        data,
	}

	res := events.NodeAppliers[ev.Type](st, ev)
	assert.Equal(t, api.NodeID("node-1"), res.ID)
	assert.Equal(t, api.HealthHealthy, res.Health["test-step"].Status)
}

func TestStepHealthChangedWithError(t *testing.T) {
	st := events.NewNodeState()

	data, err := json.Marshal(api.StepHealthChangedEvent{
		NodeID: "node-1",
		StepID: "test-step",
		Status: api.HealthUnhealthy,
		Error:  "connection refused",
	})
	assert.NoError(t, err)

	ev := &timebox.Event{
		AggregateID: events.NodeKey("node-1"),
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Data:        data,
	}

	res := events.NodeAppliers[ev.Type](st, ev)
	assert.Equal(t, api.HealthUnhealthy, res.Health["test-step"].Status)
	assert.Equal(t, "connection refused", res.Health["test-step"].Error)
}
