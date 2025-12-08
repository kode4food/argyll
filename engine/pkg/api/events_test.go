package api_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestEventJSONMarshaling(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	data := json.RawMessage(`{"step": {"id": "test-step"}}`)

	in := &timebox.Event{
		Type:        timebox.EventType(api.EventTypeStepRegistered),
		AggregateID: timebox.NewAggregateID("engine"),
		Timestamp:   now,
		Data:        data,
	}

	jsonBytes, err := json.Marshal(in)
	assert.NoError(t, err)

	var out timebox.Event
	err = json.Unmarshal(jsonBytes, &out)
	assert.NoError(t, err)

	assert.Equal(t, in.Type, out.Type)
	assert.Equal(t, in.Timestamp.Unix(), out.Timestamp.Unix())
	assert.Equal(t, in.AggregateID, out.AggregateID)
}

func TestWebSocketEventMarshaling(t *testing.T) {
	data := json.RawMessage(`{"key": "value"}`)
	in := &api.WebSocketEvent{
		Type:        api.EventTypeStepCompleted,
		Data:        data,
		Timestamp:   1234567890,
		Sequence:    42,
		AggregateID: []string{"wf-1", "step-1"},
	}

	jsonBytes, err := json.Marshal(in)
	assert.NoError(t, err)

	var out api.WebSocketEvent
	err = json.Unmarshal(jsonBytes, &out)
	assert.NoError(t, err)

	assert.Equal(t, in.Type, out.Type)
	assert.Equal(t, in.Timestamp, out.Timestamp)
	assert.Equal(t, in.Sequence, out.Sequence)
}

func TestEventTypes(t *testing.T) {
	eventTypes := []api.EventType{
		api.EventTypeStepRegistered,
		api.EventTypeStepUnregistered,
		api.EventTypeStepHealthChanged,
		api.EventTypeFlowStarted,
		api.EventTypeFlowCompleted,
		api.EventTypeFlowFailed,
		api.EventTypeAttributeSet,
		api.EventTypeStepStarted,
		api.EventTypeStepCompleted,
		api.EventTypeStepFailed,
		api.EventTypeStepSkipped,
	}

	for _, et := range eventTypes {
		assert.NotEmpty(t, string(et))
	}
}
