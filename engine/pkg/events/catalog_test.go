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
	state := events.NewCatalogState()

	assert.NotNil(t, state)
	assert.NotNil(t, state.Steps)
	assert.NotNil(t, state.Attributes)
	assert.Empty(t, state.Steps)
	assert.Empty(t, state.Attributes)
}

func TestIsCatalogEvent(t *testing.T) {
	catEvent := &timebox.Event{
		AggregateID: events.CatalogKey,
	}
	flowEvent := &timebox.Event{
		AggregateID: events.FlowKey("test-flow"),
	}

	assert.True(t, events.IsCatalogEvent(catEvent))
	assert.False(t, events.IsCatalogEvent(flowEvent))
}

func TestStepRegistered(t *testing.T) {
	initialState := events.NewCatalogState()
	now := time.Now()

	step := &api.Step{
		ID:   "test-step",
		Name: "Test Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
	}

	eventData := api.StepRegisteredEvent{Step: step}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.CatalogKey,
		Type:        timebox.EventType(api.EventTypeStepRegistered),
		Data:        data,
	}

	applier := events.CatalogAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, step, result.Steps["test-step"])
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestStepUnregistered(t *testing.T) {
	step := &api.Step{
		ID:   "test-step",
		Name: "Test Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
	}

	initialState := events.NewCatalogState().SetStep("test-step", step)
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
	result := applier(initialState, event)

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

	initialState := events.NewCatalogState().SetStep("test-step", oldStep)
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
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, newStep, result.Steps["test-step"])
	assert.Equal(t, api.Name("New Name"), result.Steps["test-step"].Name)
	assert.True(t, result.LastUpdated.Equal(now))
}
