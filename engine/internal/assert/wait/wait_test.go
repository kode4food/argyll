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
	assert.True(t, filter(&timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}))
	assert.False(t, filter(&timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowCompleted),
	}))
}

func TestTypesFilterNoTypes(t *testing.T) {
	filter := wait.Types()
	assert.False(t, filter(&timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
	}))
}

func TestAnd(t *testing.T) {
	tests := []struct {
		name      string
		returns   []bool
		want      bool
		wantCalls []int
	}{
		{
			name:      "stops when first filter is false",
			returns:   []bool{false, true, true},
			want:      false,
			wantCalls: []int{1, 0, 0},
		},
		{
			name:      "stops when second filter is false",
			returns:   []bool{true, false, true},
			want:      false,
			wantCalls: []int{1, 1, 0},
		},
		{
			name:      "calls all filters when all are true",
			returns:   []bool{true, true, true},
			want:      true,
			wantCalls: []int{1, 1, 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := make([]int, len(tc.returns))
			filters := make([]wait.EventFilter, 0, len(tc.returns))
			for i, ret := range tc.returns {
				filters = append(filters, wait.EventFilter(
					func(*timebox.Event) bool {
						calls[i]++
						return ret
					},
				))
			}

			filter := wait.And(filters...)
			assert.Equal(t, tc.want, filter(&timebox.Event{}))
			assert.Equal(t, tc.wantCalls, calls)
		})
	}
}

func TestFlowIDFilterConsumesEach(t *testing.T) {
	flowA := api.FlowID("flow-a")
	flowB := api.FlowID("flow-b")
	filter := wait.FlowIDs(flowA, flowB)
	evA := newEvent(api.EventTypeFlowStarted, events.FlowKey(flowA),
		flowEvent{FlowID: flowA})
	evB := newEvent(api.EventTypeFlowStarted, events.FlowKey(flowB),
		flowEvent{FlowID: flowB})
	assert.True(t, filter(evA))
	assert.False(t, filter(evA))
	assert.True(t, filter(evB))
	assert.False(t, filter(evB))
}

func TestFlowIDFilterSingle(t *testing.T) {
	flowID := api.FlowID("flow-single")
	filter := wait.FlowID(flowID)
	ev := newEvent(api.EventTypeFlowStarted, events.FlowKey(flowID),
		flowEvent{FlowID: flowID})
	assert.True(t, filter(ev))
	assert.False(t, filter(ev))
}

func TestFlowStepAnyMatchesRepeated(t *testing.T) {
	fs := api.FlowStep{FlowID: "flow-a", StepID: "step-a"}
	filter := wait.FlowStepAny(fs)
	ev := newEvent(api.EventTypeStepStarted, events.FlowKey(fs.FlowID),
		stepEvent{FlowID: fs.FlowID, StepID: fs.StepID})
	assert.True(t, filter(ev))
	assert.True(t, filter(ev))
}

func TestFlowStepsAndFlowStepFilter(t *testing.T) {
	a := api.FlowStep{FlowID: "flow-a", StepID: "step-a"}
	b := api.FlowStep{FlowID: "flow-b", StepID: "step-b"}
	filter := wait.FlowSteps(a, b)
	evA := newEvent(api.EventTypeStepStarted, events.FlowKey(a.FlowID),
		stepEvent{FlowID: a.FlowID, StepID: a.StepID})
	evB := newEvent(api.EventTypeStepStarted, events.FlowKey(b.FlowID),
		stepEvent{FlowID: b.FlowID, StepID: b.StepID})
	assert.True(t, filter(evA))
	assert.False(t, filter(evA))
	assert.True(t, filter(evB))

	one := wait.FlowStep(a)
	assert.True(t, one(evA))
	assert.False(t, one(evA))
}

func TestWrapperFilters(t *testing.T) {
	flowID := api.FlowID("flow-1")
	step := api.FlowStep{FlowID: flowID, StepID: "step-1"}

	flowEv := func(typ api.EventType) *timebox.Event {
		return newEvent(typ, events.EngineKey, flowEvent{FlowID: flowID})
	}
	stepEv := func(typ api.EventType) *timebox.Event {
		return newEvent(typ, events.FlowKey(flowID),
			stepEvent{FlowID: step.FlowID, StepID: step.StepID})
	}

	assert.True(t, wait.EngineEvent(api.EventTypeStepHealthChanged)(newEvent(
		api.EventTypeStepHealthChanged, events.EngineKey,
		api.StepHealthChangedEvent{StepID: "step-1", Status: api.HealthHealthy},
	)))
	assert.False(t, wait.EngineEvent(api.EventTypeStepHealthChanged)(newEvent(
		api.EventTypeStepHealthChanged, events.FlowKey(flowID),
		api.StepHealthChangedEvent{StepID: "step-1", Status: api.HealthHealthy},
	)))

	assert.True(t, wait.FlowStarted(flowID)(newEvent(
		api.EventTypeFlowStarted, events.FlowKey(flowID),
		flowEvent{FlowID: flowID},
	)))
	assert.True(t, wait.FlowActivated(flowID)(flowEv(api.EventTypeFlowActivated)))
	assert.True(t, wait.FlowDeactivated(flowID)(flowEv(api.EventTypeFlowDeactivated)))
	assert.True(t, wait.FlowCompleted(flowID)(newEvent(
		api.EventTypeFlowCompleted, events.FlowKey(flowID),
		flowEvent{FlowID: flowID},
	)))
	assert.True(t, wait.FlowFailed(flowID)(newEvent(
		api.EventTypeFlowFailed, events.FlowKey(flowID),
		flowEvent{FlowID: flowID},
	)))
	assert.False(t, wait.FlowCompleted(flowID)(newEvent(
		api.EventTypeFlowCompleted, events.FlowKey("flow-2"),
		flowEvent{FlowID: "flow-2"},
	)))

	assert.True(t, wait.StepStarted(step)(stepEv(api.EventTypeStepStarted)))
	assert.True(t, wait.StepTerminal(step)(stepEv(api.EventTypeStepCompleted)))
	assert.True(t, wait.WorkStarted(step)(stepEv(api.EventTypeWorkStarted)))
	assert.True(t, wait.WorkSucceeded(step)(stepEv(api.EventTypeWorkSucceeded)))
	assert.True(t, wait.WorkFailed(step)(stepEv(api.EventTypeWorkFailed)))
	assert.True(t, wait.WorkRetryScheduled(step)(stepEv(api.EventTypeRetryScheduled)))
	assert.True(t, wait.WorkRetryScheduledAny(step)(stepEv(
		api.EventTypeRetryScheduled,
	)))
	assert.True(t, wait.WorkRetryScheduledAny(step)(stepEv(
		api.EventTypeRetryScheduled,
	)))
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

func TestUnmarshalInvalidJSON(t *testing.T) {
	filter := wait.Unmarshal(func(data flowEvent) bool {
		return data.FlowID == "flow-a"
	})
	assert.False(t, filter(&timebox.Event{Data: []byte("{")}))
}
