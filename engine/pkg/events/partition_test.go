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
	state := events.NewPartitionState()

	assert.NotNil(t, state)
	assert.NotNil(t, state.Health)
	assert.NotNil(t, state.Active)
	assert.NotNil(t, state.FlowDigests)
	assert.NotNil(t, state.Archiving)
	assert.Empty(t, state.Health)
	assert.Empty(t, state.Active)
	assert.Empty(t, state.FlowDigests)
	assert.Empty(t, state.Deactivated)
}

func TestIsPartitionEvent(t *testing.T) {
	partEvent := &timebox.Event{
		AggregateID: events.PartitionKey,
	}
	flowEvent := &timebox.Event{
		AggregateID: events.FlowKey("test-flow"),
	}

	assert.True(t, events.IsPartitionEvent(partEvent))
	assert.False(t, events.IsPartitionEvent(flowEvent))
}

func TestStepHealthChanged(t *testing.T) {
	initialState := events.NewPartitionState().
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
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, api.HealthHealthy, result.Health["test-step"].Status)
	assert.Empty(t, result.Health["test-step"].Error)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestChangedWithError(t *testing.T) {
	initialState := events.NewPartitionState().
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
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeStepHealthChanged),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, api.HealthUnhealthy, result.Health["test-step"].Status)
	assert.Equal(t, "connection refused", result.Health["test-step"].Error)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestFlowActivated(t *testing.T) {
	initialState := events.NewPartitionState()
	now := time.Now()

	eventData := api.FlowActivatedEvent{
		FlowID:       "test-flow",
		ParentFlowID: "parent-flow",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeFlowActivated),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.NotNil(t, result.Active["test-flow"])
	assert.True(t, result.Active["test-flow"].StartedAt.Equal(now))
	assert.True(t, result.Active["test-flow"].LastActive.Equal(now))
	assert.Equal(t,
		api.FlowID("parent-flow"), result.Active["test-flow"].ParentFlowID,
	)
	assert.NotNil(t, result.FlowDigests["test-flow"])
	assert.Equal(t, api.FlowActive, result.FlowDigests["test-flow"].Status)
	assert.True(t, result.FlowDigests["test-flow"].CreatedAt.Equal(now))
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestFlowDeactivated(t *testing.T) {
	initialState := events.NewPartitionState().
		SetActiveFlow("test-flow", &api.ActiveFlow{
			ParentFlowID: "parent-flow",
			StartedAt:    time.Now(),
			LastActive:   time.Now(),
		})

	now := time.Now()

	eventData := api.FlowDeactivatedEvent{FlowID: "test-flow"}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeFlowDeactivated),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Nil(t, result.Active["test-flow"])
	assert.Equal(t,
		api.FlowID("parent-flow"), result.Deactivated[0].ParentFlowID,
	)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestFlowArchiving(t *testing.T) {
	initialState := events.NewPartitionState().
		AddDeactivated(&api.DeactivatedFlow{
			FlowID:        "test-flow",
			DeactivatedAt: time.Now(),
		})
	now := time.Now()

	eventData := api.FlowArchivingEvent{FlowID: "test-flow"}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeFlowArchiving),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Len(t, result.Deactivated, 0)
	assert.Len(t, result.Archiving, 1)
	assert.True(t, result.Archiving["test-flow"].Equal(now))
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestFlowArchived(t *testing.T) {
	now := time.Now()
	initialState := events.NewPartitionState().
		AddDeactivated(&api.DeactivatedFlow{
			FlowID:        "test-flow",
			DeactivatedAt: now.Add(-time.Minute),
		}).
		AddArchiving("test-flow", now).
		SetFlowDigest("test-flow", &api.FlowDigest{
			Status:    api.FlowCompleted,
			CreatedAt: now.Add(-time.Hour),
		})

	eventData := api.FlowArchivedEvent{FlowID: "test-flow"}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeFlowArchived),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Len(t, result.Archiving, 0)
	assert.Len(t, result.Deactivated, 1)
	assert.Nil(t, result.FlowDigests["test-flow"])
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestFlowDigestUpdated(t *testing.T) {
	now := time.Now()
	initialState := events.NewPartitionState().
		SetActiveFlow("test-flow", &api.ActiveFlow{
			StartedAt:  now.Add(-time.Minute),
			LastActive: now.Add(-time.Minute),
		}).
		SetFlowDigest("test-flow", &api.FlowDigest{
			Status:    api.FlowActive,
			CreatedAt: now.Add(-time.Minute),
		})

	eventData := api.FlowDigestUpdatedEvent{
		FlowID:      "test-flow",
		Status:      api.FlowCompleted,
		CompletedAt: now,
		Error:       "",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeFlowDigestUpdated),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Equal(t, api.FlowCompleted, result.FlowDigests["test-flow"].Status)
	assert.True(t, result.FlowDigests["test-flow"].CompletedAt.Equal(now))
	assert.True(t,
		result.FlowDigests["test-flow"].CreatedAt.Equal(now.Add(-time.Minute)),
	)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestTimeoutScheduled(t *testing.T) {
	initialState := events.NewPartitionState()
	now := time.Now()
	firesAt := now.Add(5 * time.Second)

	eventData := api.TimeoutScheduledEvent{
		FlowID:          "test-flow",
		StepID:          "test-step",
		FiresAt:         firesAt,
		Attributes:      []api.Name{"attr1", "attr2"},
		UpstreamStepIDs: []api.StepID{"upstream-1"},
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeTimeoutScheduled),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Len(t, result.Timeouts, 1)
	assert.Equal(t, api.FlowID("test-flow"), result.Timeouts[0].FlowID)
	assert.Equal(t, api.StepID("test-step"), result.Timeouts[0].StepID)
	assert.True(t, result.Timeouts[0].FiresAt.Equal(firesAt))
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestTimeoutScheduledMaintainsSortOrder(t *testing.T) {
	now := time.Now()
	earlierTime := now.Add(2 * time.Second)
	laterTime := now.Add(5 * time.Second)

	initialState := events.NewPartitionState()

	// Add later timeout first
	eventData1 := api.TimeoutScheduledEvent{
		FlowID:  "flow1",
		StepID:  "step1",
		FiresAt: laterTime,
	}
	data1, _ := json.Marshal(eventData1)
	event1 := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeTimeoutScheduled),
		Data:        data1,
	}

	applier := events.PartitionAppliers[event1.Type]
	state1 := applier(initialState, event1)

	// Add earlier timeout second
	eventData2 := api.TimeoutScheduledEvent{
		FlowID:  "flow2",
		StepID:  "step2",
		FiresAt: earlierTime,
	}
	data2, _ := json.Marshal(eventData2)
	event2 := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeTimeoutScheduled),
		Data:        data2,
	}

	state2 := applier(state1, event2)

	assert.NotNil(t, state2)
	assert.Len(t, state2.Timeouts, 2)
	// Earlier timeout should be first
	assert.True(t, state2.Timeouts[0].FiresAt.Equal(earlierTime))
	assert.True(t, state2.Timeouts[1].FiresAt.Equal(laterTime))
}

func TestTimeoutFired(t *testing.T) {
	now := time.Now()
	initialState := events.NewPartitionState().
		SetTimeouts([]*api.TimeoutEntry{
			{
				FlowID:  "test-flow",
				StepID:  "test-step",
				FiresAt: now.Add(-1 * time.Second),
			},
		})

	eventData := api.TimeoutFiredEvent{
		FlowID:  "test-flow",
		StepID:  "test-step",
		FiresAt: now,
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeTimeoutFired),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Empty(t, result.Timeouts)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestTimeoutFiredWithMultipleEntries(t *testing.T) {
	now := time.Now()
	initialState := events.NewPartitionState().
		SetTimeouts([]*api.TimeoutEntry{
			{
				FlowID:  "flow1",
				StepID:  "step1",
				FiresAt: now.Add(-1 * time.Second),
			},
			{
				FlowID:  "flow2",
				StepID:  "step2",
				FiresAt: now.Add(5 * time.Second),
			},
		})

	eventData := api.TimeoutFiredEvent{
		FlowID:  "flow1",
		StepID:  "step1",
		FiresAt: now,
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeTimeoutFired),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Len(t, result.Timeouts, 1)
	assert.Equal(t, api.FlowID("flow2"), result.Timeouts[0].FlowID)
	assert.Equal(t, api.StepID("step2"), result.Timeouts[0].StepID)
}

func TestTimeoutCanceled(t *testing.T) {
	now := time.Now()
	initialState := events.NewPartitionState().
		SetTimeouts([]*api.TimeoutEntry{
			{
				FlowID:  "test-flow",
				StepID:  "test-step",
				FiresAt: now.Add(5 * time.Second),
			},
		})

	eventData := api.TimeoutCanceledEvent{
		FlowID: "test-flow",
		StepID: "test-step",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeTimeoutCanceled),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Empty(t, result.Timeouts)
	assert.True(t, result.LastUpdated.Equal(now))
}

func TestTimeoutCanceledFromMiddle(t *testing.T) {
	now := time.Now()
	initialState := events.NewPartitionState().
		SetTimeouts([]*api.TimeoutEntry{
			{
				FlowID:  "flow1",
				StepID:  "step1",
				FiresAt: now.Add(1 * time.Second),
			},
			{
				FlowID:  "flow2",
				StepID:  "step2",
				FiresAt: now.Add(5 * time.Second),
			},
			{
				FlowID:  "flow3",
				StepID:  "step3",
				FiresAt: now.Add(10 * time.Second),
			},
		})

	eventData := api.TimeoutCanceledEvent{
		FlowID: "flow2",
		StepID: "step2",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeTimeoutCanceled),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Len(t, result.Timeouts, 2)
	assert.Equal(t, api.FlowID("flow1"), result.Timeouts[0].FlowID)
	assert.Equal(t, api.FlowID("flow3"), result.Timeouts[1].FlowID)
}

func TestTimeoutCanceledNotFound(t *testing.T) {
	now := time.Now()
	initialState := events.NewPartitionState().
		SetTimeouts([]*api.TimeoutEntry{
			{
				FlowID:  "flow1",
				StepID:  "step1",
				FiresAt: now.Add(5 * time.Second),
			},
		})

	eventData := api.TimeoutCanceledEvent{
		FlowID: "nonexistent-flow",
		StepID: "nonexistent-step",
	}
	data, err := json.Marshal(eventData)
	assert.NoError(t, err)

	event := &timebox.Event{
		Timestamp:   now,
		AggregateID: events.PartitionKey,
		Type:        timebox.EventType(api.EventTypeTimeoutCanceled),
		Data:        data,
	}

	applier := events.PartitionAppliers[event.Type]
	result := applier(initialState, event)

	assert.NotNil(t, result)
	assert.Len(t, result.Timeouts, 1)
	assert.Equal(t, api.FlowID("flow1"), result.Timeouts[0].FlowID)
}
