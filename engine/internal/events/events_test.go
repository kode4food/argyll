package events_test

import (
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/events"
	"github.com/kode4food/argyll/engine/pkg/api"
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

func TestFilterAggregate(t *testing.T) {
	filter := events.FilterAggregate(
		timebox.NewAggregateID("flow", "test-flow-123"),
	)

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

func TestOrFiltersEmpty(t *testing.T) {
	combined := events.OrFilters()

	event := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}

	assert.False(t, combined(event))
}

func TestAndFilters(t *testing.T) {
	typeFilter := events.FilterEvents(
		timebox.EventType(api.EventTypeFlowStarted),
	)
	aggregateFilter := events.FilterAggregate(
		timebox.NewAggregateID("flow", "wf-123"),
	)

	combined := events.AndFilters(typeFilter, aggregateFilter)

	matchingEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	wrongTypeEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowCompleted),
		AggregateID: timebox.NewAggregateID("flow", "wf-123"),
	}
	wrongAggregateEvent := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeFlowStarted),
		AggregateID: timebox.NewAggregateID("flow", "wf-456"),
	}

	assert.True(t, combined(matchingEvent))
	assert.False(t, combined(wrongTypeEvent))
	assert.False(t, combined(wrongAggregateEvent))
}

func TestAndFiltersEmpty(t *testing.T) {
	combined := events.AndFilters()

	event := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}

	assert.True(t, combined(event))
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
