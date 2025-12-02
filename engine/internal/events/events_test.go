package events_test

import (
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/spuds/engine/internal/events"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestFilterEvents(t *testing.T) {
	filter := events.FilterEvents(
		timebox.EventType(api.EventTypeFlowStarted),
		timebox.EventType(api.EventTypeFlowCompleted),
	)

	event1 := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}
	event2 := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowCompleted),
	}
	event3 := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowFailed),
	}

	assert.True(t, filter(event1))
	assert.True(t, filter(event2))
	assert.False(t, filter(event3))
}

func TestFilterFlow(t *testing.T) {
	flowID := api.FlowID("test-flow-123")
	filter := events.FilterFlow(flowID)

	flowEvent := &timebox.Event{
		AggregateID: timebox.NewAggregateID("flow", "test-flow-123"),
	}
	otherFlowEvent := &timebox.Event{
		AggregateID: timebox.NewAggregateID("flow", "different-flow"),
	}
	engineEvent := &timebox.Event{
		AggregateID: timebox.NewAggregateID("engine"),
	}

	assert.True(t, filter(flowEvent))
	assert.False(t, filter(otherFlowEvent))
	assert.False(t, filter(engineEvent))
}

func TestOrFilters(t *testing.T) {
	filter1 := events.FilterEvents(
		timebox.EventType(api.EventTypeFlowStarted),
	)
	filter2 := events.FilterEvents(
		timebox.EventType(api.EventTypeFlowCompleted),
	)

	combined := events.OrFilters(filter1, filter2)

	event1 := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}
	event2 := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowCompleted),
	}
	event3 := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowFailed),
	}

	assert.True(t, combined(event1))
	assert.True(t, combined(event2))
	assert.False(t, combined(event3))
}

func TestNoFilters(t *testing.T) {
	combined := events.OrFilters()

	event := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}

	assert.False(t, combined(event))
}

func TestMakeAppliers(t *testing.T) {
	applierFunc := func(
		st *api.FlowState, ev *timebox.Event, data api.FlowStartedEvent,
	) *api.FlowState {
		return st
	}

	appMap := map[api.EventType]timebox.Applier[*api.FlowState]{
		api.EventTypeFlowStarted: timebox.MakeApplier(applierFunc),
	}

	result := events.MakeAppliers(appMap)

	assert.Len(t, result, 1)
	assert.Contains(
		t,
		result,
		timebox.EventType(api.EventTypeFlowStarted),
	)
}
