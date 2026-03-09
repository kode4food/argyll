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
	top := caravan.NewTopic[*timebox.Event]()
	return timebox.NewEventHub(top), top
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
				filters = append(filters, func(*timebox.Event) bool {
					calls[i]++
					return ret
				})
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
	id := api.FlowID("flow-single")
	filter := wait.FlowID(id)
	ev := newEvent(api.EventTypeFlowStarted, events.FlowKey(id),
		flowEvent{FlowID: id})
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
	id := api.FlowID("flow-1")
	fs := api.FlowStep{FlowID: id, StepID: "step-1"}

	stepEv := func(typ api.EventType) *timebox.Event {
		return newEvent(typ, events.FlowKey(id),
			stepEvent{FlowID: fs.FlowID, StepID: fs.StepID})
	}

	assert.True(t, wait.EngineEvent(api.EventTypeStepHealthChanged)(newEvent(
		api.EventTypeStepHealthChanged, events.PartitionKey,
		api.StepHealthChangedEvent{StepID: "step-1", Status: api.HealthHealthy},
	)))
	assert.False(t, wait.EngineEvent(api.EventTypeStepHealthChanged)(newEvent(
		api.EventTypeStepHealthChanged, events.FlowKey(id),
		api.StepHealthChangedEvent{StepID: "step-1", Status: api.HealthHealthy},
	)))

	assert.True(t, wait.FlowStarted(id)(newEvent(
		api.EventTypeFlowStarted, events.FlowKey(id),
		flowEvent{FlowID: id},
	)))
	assert.True(t, wait.FlowActivated(id)(newEvent(
		api.EventTypeFlowStarted, events.FlowKey(id),
		flowEvent{FlowID: id},
	)))
	assert.True(t, wait.FlowDeactivated(id)(newEvent(
		api.EventTypeFlowDeactivated, events.FlowKey(id),
		flowEvent{FlowID: id},
	)))
	assert.True(t, wait.FlowCompleted(id)(newEvent(
		api.EventTypeFlowCompleted, events.FlowKey(id),
		flowEvent{FlowID: id},
	)))
	assert.True(t, wait.FlowFailed(id)(newEvent(
		api.EventTypeFlowFailed, events.FlowKey(id),
		flowEvent{FlowID: id},
	)))
	assert.False(t, wait.FlowCompleted(id)(newEvent(
		api.EventTypeFlowCompleted, events.FlowKey("flow-2"),
		flowEvent{FlowID: "flow-2"},
	)))

	assert.True(t, wait.StepStarted(fs)(stepEv(api.EventTypeStepStarted)))
	assert.True(t, wait.StepTerminal(fs)(stepEv(api.EventTypeStepCompleted)))
	assert.True(t, wait.WorkStarted(fs)(stepEv(api.EventTypeWorkStarted)))
	assert.True(t, wait.WorkSucceeded(fs)(stepEv(api.EventTypeWorkSucceeded)))
	assert.True(t, wait.WorkFailed(fs)(stepEv(api.EventTypeWorkFailed)))
	assert.True(t, wait.WorkRetryScheduled(fs)(stepEv(api.EventTypeRetryScheduled)))
	assert.True(t, wait.WorkRetryScheduledAny(fs)(stepEv(
		api.EventTypeRetryScheduled,
	)))
	assert.True(t, wait.WorkRetryScheduledAny(fs)(stepEv(
		api.EventTypeRetryScheduled,
	)))
}

func TestWaitForEventFlowTerminal(t *testing.T) {
	hub, top := newHub()
	consumer := hub.NewConsumer()
	defer consumer.Close()
	producer := top.NewProducer()

	id := api.FlowID("flow-terminal")
	ev := newEvent(api.EventTypeFlowCompleted, events.FlowKey(id),
		flowEvent{FlowID: id})
	go func() {
		producer.Send() <- ev
	}()

	wait.On(t, consumer).ForEvent(wait.FlowTerminal(id))
}

func TestStepHealthChangedFilter(t *testing.T) {
	filter := wait.StepHealthChanged("step-a", api.HealthHealthy)
	ev := newEvent(api.EventTypeStepHealthChanged, events.PartitionKey,
		api.StepHealthChangedEvent{
			StepID: "step-a",
			Status: api.HealthHealthy,
		})
	assert.True(t, filter(ev))
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	filter := wait.PredicateFilter(func(data flowEvent) bool {
		return data.FlowID == "flow-a"
	})
	assert.False(t, filter(&timebox.Event{Data: []byte("{")}))
}
