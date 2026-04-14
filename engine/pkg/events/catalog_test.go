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

func TestNewCatalogState(t *testing.T) {
	cat := events.NewCatalogState()

	assert.NotNil(t, cat)
	assert.NotNil(t, cat.Steps)
	assert.NotNil(t, cat.Attributes)
	assert.Empty(t, cat.Steps)
	assert.Empty(t, cat.Attributes)
}

func TestIsCatalogEvent(t *testing.T) {
	catEvent := &timebox.Event{
		AggregateID: events.CatalogKey,
	}
	nestedEvent := &timebox.Event{
		AggregateID: timebox.NewAggregateID(events.CatalogPrefix, "bad"),
	}
	flowEvent := &timebox.Event{
		AggregateID: events.FlowKey("test-flow"),
	}

	assert.True(t, events.IsCatalogEvent(catEvent))
	assert.False(t, events.IsCatalogEvent(nestedEvent))
	assert.False(t, events.IsCatalogEvent(flowEvent))
}

func TestIsCatalogEventID(t *testing.T) {
	assert.True(t, events.IsCatalogEventID(events.CatalogKey))
	assert.False(t, events.IsCatalogEventID(
		timebox.NewAggregateID(events.CatalogPrefix, "bad"),
	))
	assert.False(t, events.IsCatalogEventID(events.FlowKey("test-flow")))
}

func TestStepRegistered(t *testing.T) {
	cat := events.NewCatalogState()
	now := time.Now()

	st := &api.Step{
		ID:   "test-step",
		Name: "Test Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
	}

	eventData := api.StepRegisteredEvent{Step: st}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.CatalogKey,
		Type:        timebox.EventType(api.EventTypeStepRegistered),
		Data:        data,
	}

	applier := events.CatalogAppliers[event.Type]
	result := applier(cat, event)

	assert.NotNil(t, result)
	assert.Equal(t, st, result.Steps["test-step"])
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestStepUnregistered(t *testing.T) {
	st := &api.Step{
		ID:   "test-step",
		Name: "Test Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
	}

	cat := events.NewCatalogState().SetStep("test-step", st)
	now := time.Now()

	eventData := api.StepUnregisteredEvent{StepID: "test-step"}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.CatalogKey,
		Type:        timebox.EventType(api.EventTypeStepUnregistered),
		Data:        data,
	}

	applier := events.CatalogAppliers[event.Type]
	result := applier(cat, event)

	assert.NotNil(t, result)
	assert.Nil(t, result.Steps["test-step"])
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestStepUpdated(t *testing.T) {
	oldStep := &api.Step{
		ID:   "test-step",
		Name: "Old Name",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
	}

	newStep := &api.Step{
		ID:   "test-step",
		Name: "New Name",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:9090",
		},
	}

	cat := events.NewCatalogState().SetStep("test-step", oldStep)
	now := time.Now()

	eventData := api.StepUpdatedEvent{Step: newStep}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.CatalogKey,
		Type:        timebox.EventType(api.EventTypeStepUpdated),
		Data:        data,
	}

	applier := events.CatalogAppliers[event.Type]
	result := applier(cat, event)

	assert.NotNil(t, result)
	assert.Equal(t, newStep, result.Steps["test-step"])
	assert.Equal(t, api.Name("New Name"), result.Steps["test-step"].Name)
	assert.True(t, result.LastUpdated.Equal(now))
}
