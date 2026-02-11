package wait_test

import (
	"encoding/json"
	"testing"

	"github.com/kode4food/caravan"
	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type (
	flowEvent struct {
		FlowID api.FlowID `json:"flow_id"`
	}

	stepEvent struct {
		FlowID api.FlowID `json:"flow_id"`
		StepID api.StepID `json:"step_id"`
	}
)

func newHub() (*timebox.EventHub, topic.Topic[*timebox.Event]) {
	topic := caravan.NewTopic[*timebox.Event]()
	return timebox.NewEventHub(topic), topic
}

func newEvent(
	typ api.EventType, agg timebox.AggregateID, data any,
) *timebox.Event {
	payload, _ := json.Marshal(data)
	return &timebox.Event{
		Type:        timebox.EventType(typ),
		AggregateID: agg,
		Data:        payload,
	}
}

func TestTypesFilter(t *testing.T) {
	filter := wait.Types(api.EventTypeFlowStarted, api.EventTypeFlowFailed)
	assert.False(t, filter(nil))
	assert.True(t, filter(&timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}))
	assert.False(t, filter(&timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowCompleted),
	}))
}

func TestFlowIDFilterConsumesEach(t *testing.T) {
	flowA := api.FlowID("flow-a")
	flowB := api.FlowID("flow-b")
	filter := wait.FlowID(flowA, flowB)
	evA := newEvent(api.EventTypeFlowStarted, events.FlowKey(flowA),
		flowEvent{FlowID: flowA})
	evB := newEvent(api.EventTypeFlowStarted, events.FlowKey(flowB),
		flowEvent{FlowID: flowB})
	assert.True(t, filter(evA))
	assert.False(t, filter(evA))
	assert.True(t, filter(evB))
	assert.False(t, filter(evB))
}

func TestFlowStepAnyMatchesRepeated(t *testing.T) {
	fs := api.FlowStep{FlowID: "flow-a", StepID: "step-a"}
	filter := wait.FlowStepAny(fs)
	ev := newEvent(api.EventTypeStepStarted, events.FlowKey(fs.FlowID),
		stepEvent{FlowID: fs.FlowID, StepID: fs.StepID})
	assert.True(t, filter(ev))
	assert.True(t, filter(ev))
}

func TestWaitForEventFlowTerminal(t *testing.T) {
	hub, topic := newHub()
	consumer := hub.NewConsumer()
	defer consumer.Close()
	producer := topic.NewProducer()

	flowID := api.FlowID("flow-terminal")
	ev := newEvent(api.EventTypeFlowCompleted, events.FlowKey(flowID),
		flowEvent{FlowID: flowID})
	go func() {
		producer.Send() <- ev
	}()

	wait.On(t, consumer).ForEvent(wait.FlowTerminal(flowID))
}

func TestStepHealthChangedFilter(t *testing.T) {
	filter := wait.StepHealthChanged("step-a", api.HealthHealthy)
	ev := newEvent(api.EventTypeStepHealthChanged, events.EngineKey,
		api.StepHealthChangedEvent{
			StepID: "step-a",
			Status: api.HealthHealthy,
		})
	assert.True(t, filter(ev))
}
