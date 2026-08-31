package events_test

import (
	"encoding/json"
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestNewCatalogState(t *testing.T) {
	cat := events.NewCatalogState()

	assert.NotNil(t, cat)
	assert.NotNil(t, cat.Steps)
	assert.NotNil(t, cat.Spaces)
	assert.NotNil(t, cat.Attributes)
	assert.Empty(t, cat.Steps)
	assert.Empty(t, cat.Spaces)
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
	now := scheduler.Now()

	st := &api.Step{
		ID:   "test-step",
		Name: "Test Step",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://localhost:8080"},
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
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://localhost:8080"},
		},
	}

	cat := events.NewCatalogState().SetStep("test-step", st)
	now := scheduler.Now()

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
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://localhost:8080"},
		},
	}

	newStep := &api.Step{
		ID:   "test-step",
		Name: "New Name",
		Type: api.StepTypeService,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://localhost:9090"},
		},
	}

	cat := events.NewCatalogState().SetStep("test-step", oldStep)
	now := scheduler.Now()

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

func TestSpaceEvents(t *testing.T) {
	cat := events.NewCatalogState()
	space := api.Space{ID: "payments", Name: "Payments"}

	cat = applyCatalogEvent(t, cat, api.EventTypeSpaceRegistered,
		api.SpaceRegisteredEvent{Space: space})
	assert.Equal(t, space, cat.Spaces[space.ID])

	updated := api.Space{
		ID:          space.ID,
		Name:        space.Name,
		Description: "Updated",
	}
	cat = applyCatalogEvent(t, cat, api.EventTypeSpaceUpdated,
		api.SpaceUpdatedEvent{Space: updated})
	assert.Equal(t, updated, cat.Spaces[space.ID])

	cat = applyCatalogEvent(t, cat, api.EventTypeSpaceUnregistered,
		api.SpaceUnregisteredEvent{SpaceID: space.ID})
	assert.NotContains(t, cat.Spaces, space.ID)
}

func TestSpaceSelectionProjection(t *testing.T) {
	cat := events.NewCatalogState()
	space := api.Space{
		ID:   "payments",
		Name: "Payments",
		QBE:  api.SpaceQuery{"domain": {"payments"}},
	}
	inside := &api.Step{ID: "inside", Labels: api.Labels{
		"domain": "payments",
	}}
	outside := &api.Step{ID: "outside", Labels: api.Labels{
		"domain": "orders",
	}}

	cat = applyCatalogEvent(t, cat, api.EventTypeSpaceRegistered,
		api.SpaceRegisteredEvent{Space: space})
	assert.Empty(t, cat.SpaceSteps(space.ID))

	cat = applyCatalogEvent(t, cat, api.EventTypeStepRegistered,
		api.StepRegisteredEvent{Step: inside, Spaces: []api.SpaceID{space.ID}})
	cat = applyCatalogEvent(t, cat, api.EventTypeStepRegistered,
		api.StepRegisteredEvent{Step: outside})
	assert.Equal(t, api.Steps{inside.ID: inside}, cat.SpaceSteps(space.ID))

	// Relabelling out of the Space drops the membership
	cat = applyCatalogEvent(t, cat, api.EventTypeStepUpdated,
		api.StepUpdatedEvent{Step: inside})
	assert.Empty(t, cat.SpaceSteps(space.ID))

	// A Space write replaces membership wholesale
	cat = applyCatalogEvent(t, cat, api.EventTypeSpaceUpdated,
		api.SpaceUpdatedEvent{
			Space:   space,
			StepIDs: []api.StepID{outside.ID},
		})
	assert.Equal(t, api.Steps{outside.ID: outside}, cat.SpaceSteps(space.ID))

	cat = applyCatalogEvent(t, cat, api.EventTypeStepUnregistered,
		api.StepUnregisteredEvent{
			StepID: outside.ID,
			Spaces: []api.SpaceID{space.ID},
		})
	assert.Empty(t, cat.SpaceSteps(space.ID))

	cat = applyCatalogEvent(t, cat, api.EventTypeSpaceUnregistered,
		api.SpaceUnregisteredEvent{SpaceID: space.ID})
	assert.NotContains(t, cat.Selection, space.ID)
}

func applyCatalogEvent(
	t *testing.T, cat api.CatalogState, typ api.EventType, value any,
) api.CatalogState {
	t.Helper()
	data, err := json.Marshal(value)
	assert.NoError(t, err)
	ev := &timebox.Event{
		Timestamp:   scheduler.Now(),
		AggregateID: events.CatalogKey,
		Type:        timebox.EventType(typ),
		Data:        data,
	}
	return events.CatalogAppliers[ev.Type](cat, ev)
}
