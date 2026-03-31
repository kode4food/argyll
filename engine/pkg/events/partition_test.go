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

func TestNewPartitionState(t *testing.T) {
	st := events.NewPartitionState()

	assert.NotNil(t, st)
	assert.NotNil(t, st.Health)
	assert.Empty(t, st.Health)
}

func TestIsPartitionEvent(t *testing.T) {
	partEv := &timebox.Event{AggregateID: events.PartitionKey("node-1")}
	flowEv := &timebox.Event{AggregateID: events.FlowKey("test-flow")}

	assert.True(t, events.IsPartitionEvent(partEv))
	assert.False(t, events.IsPartitionEvent(flowEv))
}

func TestStepHealthChanged(t *testing.T) {
	st := events.NewPartitionState().
		SetHealth("test-step", &api.HealthState{Status: api.HealthUnknown})
	now := time.Now()

	data, err := json.Marshal(api.StepHealthChangedEvent{
		StepID: "test-step",
		Status: api.HealthHealthy,
	})
	assert.NoError(t, err)

	ev := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey("node-1"),
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Data:        data,
	}

	res := events.PartitionAppliers[ev.Type](st, ev)

	assert.Equal(t, api.HealthHealthy, res.Health["test-step"].Status)
	assert.True(t, res.LastUpdated.Equal(now))
}

func TestStepHealthChangedWithError(t *testing.T) {
	st := events.NewPartitionState()
	now := time.Now()

	data, err := json.Marshal(api.StepHealthChangedEvent{
		StepID: "test-step",
		Status: api.HealthUnhealthy,
		Error:  "connection refused",
	})
	assert.NoError(t, err)

	ev := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey("node-1"),
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Data:        data,
	}

	res := events.PartitionAppliers[ev.Type](st, ev)

	assert.Equal(t, api.HealthUnhealthy, res.Health["test-step"].Status)
	assert.Equal(t, "connection refused", res.Health["test-step"].Error)
	assert.True(t, res.LastUpdated.Equal(now))
}
