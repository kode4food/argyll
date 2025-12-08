package events_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/events"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestNewEngineState(t *testing.T) {
	state := events.NewEngineState()

	assert.NotNil(t, state)
	assert.NotNil(t, state.Steps)
	assert.NotNil(t, state.Health)
	assert.NotNil(t, state.ActiveFlows)
	assert.NotNil(t, state.Attributes)
	assert.Empty(t, state.Steps)
	assert.Empty(t, state.Health)
	assert.Empty(t, state.ActiveFlows)
	assert.Empty(t, state.Attributes)
}

func TestIsEngineEvent(t *testing.T) {
	engineEvent := &timebox.Event{
		AggregateID: events.EngineID,
	}
	flowEvent := &timebox.Event{
		AggregateID: timebox.NewAggregateID("flow", "test-flow"),
	}

	assert.True(t, events.IsEngineEvent(engineEvent))
	assert.False(t, events.IsEngineEvent(flowEvent))
}

func TestStepRegistered(t *testing.T) {
	initialState := events.NewEngineState()
	now := time.Now()

	step := &api.Step{
		ID:      "test-step",
		Name:    "Test Step",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
	}

	eventData := api.StepRegisteredEvent{Step: step}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeStepRegistered),
		Data:        data,
	}

	applier := events.EngineAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, step, result.Steps["test-step"])
	assert.NotNil(t, result.Health["test-step"])
	assert.Equal(t, api.HealthUnknown, result.Health["test-step"].Status)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestStepUnregistered(t *testing.T) {
	step := &api.Step{
		ID:      "test-step",
		Name:    "Test Step",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
	}

	initialState := events.NewEngineState().
		SetStep("test-step", step).
		SetHealth("test-step", &api.HealthState{Status: api.HealthHealthy})

	now := time.Now()

	eventData := api.StepUnregisteredEvent{StepID: "test-step"}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeStepUnregistered),
		Data:        data,
	}

	applier := events.EngineAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Nil(t, result.Steps["test-step"])
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestStepUpdated(t *testing.T) {
	oldStep := &api.Step{
		ID:      "test-step",
		Name:    "Old Name",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:8080",
		},
	}

	newStep := &api.Step{
		ID:      "test-step",
		Name:    "New Name",
		Type:    api.StepTypeSync,
		Version: "2.0.0",
		HTTP: &api.HTTPConfig{
			Endpoint: "http://localhost:9090",
		},
	}

	initialState := events.NewEngineState().SetStep("test-step", oldStep)
	now := time.Now()

	eventData := api.StepUpdatedEvent{Step: newStep}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeStepUpdated),
		Data:        data,
	}

	applier := events.EngineAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, newStep, result.Steps["test-step"])
	assert.Equal(t, api.Name("New Name"), result.Steps["test-step"].Name)
	assert.Equal(t, "2.0.0", result.Steps["test-step"].Version)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestStepHealthChanged(t *testing.T) {
	initialState := events.NewEngineState().
		SetHealth("test-step", &api.HealthState{Status: api.HealthUnknown})

	now := time.Now()

	eventData := api.StepHealthChangedEvent{
		StepID: "test-step",
		Status: api.HealthHealthy,
		Error:  "",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Data:        data,
	}

	applier := events.EngineAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, api.HealthHealthy, result.Health["test-step"].Status)
	assert.Empty(t, result.Health["test-step"].Error)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestChangedWithError(t *testing.T) {
	initialState := events.NewEngineState().
		SetHealth("test-step", &api.HealthState{Status: api.HealthHealthy})

	now := time.Now()

	eventData := api.StepHealthChangedEvent{
		StepID: "test-step",
		Status: api.HealthUnhealthy,
		Error:  "connection refused",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Data:        data,
	}

	applier := events.EngineAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, api.HealthUnhealthy, result.Health["test-step"].Status)
	assert.Equal(t, "connection refused", result.Health["test-step"].Error)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestFlowActivated(t *testing.T) {
	initialState := events.NewEngineState()
	now := time.Now()

	eventData := api.FlowActivatedEvent{FlowID: "test-flow"}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeFlowActivated),
		Data:        data,
	}

	applier := events.EngineAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.NotNil(t, result.ActiveFlows["test-flow"])
	assert.Equal(
		t,
		api.FlowID("test-flow"),
		result.ActiveFlows["test-flow"].FlowID,
	)
	assert.True(t, result.ActiveFlows["test-flow"].StartedAt.Equal(now))
	assert.True(t, result.ActiveFlows["test-flow"].LastActive.Equal(now))
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestFlowDeactivated(t *testing.T) {
	initialState := events.NewEngineState().
		SetActiveFlow("test-flow", &api.ActiveFlowInfo{
			FlowID:     "test-flow",
			StartedAt:  time.Now(),
			LastActive: time.Now(),
		})

	now := time.Now()

	eventData := api.FlowDeactivatedEvent{FlowID: "test-flow"}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.EngineID,
		Type:        timebox.EventType(api.EventTypeFlowDeactivated),
		Data:        data,
	}

	applier := events.EngineAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Nil(t, result.ActiveFlows["test-flow"])
	assert.True(t, result.LastUpdated.Equal(now))
}
