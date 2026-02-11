package wait

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	Wait struct {
		t        *testing.T
		consumer *timebox.Consumer
		timeout  time.Duration
	}

	EventFilter predicate[*timebox.Event]

	flowEvent struct {
		FlowID api.FlowID `json:"flow_id"`
	}

	stepEvent struct {
		FlowID api.FlowID `json:"flow_id"`
		StepID api.StepID `json:"step_id"`
	}

	predicate[T any] func(T) bool
)

const DefaultTimeout = time.Second * 5

func On(t *testing.T, consumer *timebox.Consumer) *Wait {
	return &Wait{
		t:        t,
		consumer: consumer,
		timeout:  DefaultTimeout,
	}
}

func (w *Wait) WithTimeout(timeout time.Duration) *Wait {
	res := *w
	res.timeout = timeout
	return &res
}

// ForEvents waits for matching events from the consumer
func (w *Wait) ForEvents(count int, filter EventFilter) {
	w.t.Helper()

	deadline := time.NewTimer(w.timeout)
	defer deadline.Stop()

	for seen := 0; seen < count; {
		select {
		case ev, ok := <-w.consumer.Receive():
			if !ok {
				w.t.Fatalf(
					"event consumer closed before receiving %d events", count,
				)
			}
			if ev == nil || !filter(ev) {
				continue
			}
			seen++
		case <-deadline.C:
			w.t.Fatalf("timeout waiting for %d events", count)
		}
	}
}

// ForEvent waits for a single matching event
func (w *Wait) ForEvent(filter EventFilter) {
	w.ForEvents(1, filter)
}

// And composes event filters and returns true when all match
func And(filters ...EventFilter) EventFilter {
	return func(ev *timebox.Event) bool {
		for _, filter := range filters {
			if filter == nil || !filter(ev) {
				return false
			}
		}
		return true
	}
}

// Type creates a filter for the given event types
func Type(eventTypes ...api.EventType) EventFilter {
	return filterEventTypes(eventTypes...)
}

// EventType creates a filter for a single event type
func EventType(eventType api.EventType) EventFilter {
	return Type(eventType)
}

// EngineFilter restricts events to the engine aggregate
func EngineFilter() EventFilter {
	return filterAggregate(events.EngineKey)
}

// EngineEvent matches engine aggregate events for the given types
func EngineEvent(eventTypes ...api.EventType) EventFilter {
	return And(EngineFilter(), Type(eventTypes...))
}

// FlowStarted matches flow started events for the provided flow IDs
func FlowStarted(ids ...api.FlowID) EventFilter {
	return And(Type(api.EventTypeFlowStarted), FlowID(ids...))
}

// FlowActivated matches flow activated events for the provided flow IDs
func FlowActivated(ids ...api.FlowID) EventFilter {
	return And(
		EngineFilter(),
		Type(api.EventTypeFlowActivated),
		FlowID(ids...),
	)
}

// FlowDeactivated matches flow deactivated events for the provided flow IDs
func FlowDeactivated(ids ...api.FlowID) EventFilter {
	return And(
		EngineFilter(),
		Type(api.EventTypeFlowDeactivated),
		FlowID(ids...),
	)
}

// FlowTerminal matches flow terminal events for the provided flow IDs
func FlowTerminal(ids ...api.FlowID) EventFilter {
	return And(
		Type(api.EventTypeFlowCompleted, api.EventTypeFlowFailed),
		FlowID(ids...),
	)
}

// FlowCompleted matches flow completed events for the provided flow IDs
func FlowCompleted(ids ...api.FlowID) EventFilter {
	return And(Type(api.EventTypeFlowCompleted), FlowID(ids...))
}

// FlowFailed matches flow failed events for the provided flow IDs
func FlowFailed(ids ...api.FlowID) EventFilter {
	return And(Type(api.EventTypeFlowFailed), FlowID(ids...))
}

// StepStarted matches step started events for the provided flow steps
func StepStarted(steps ...api.FlowStep) EventFilter {
	return And(Type(api.EventTypeStepStarted), FlowStep(steps...))
}

// StepTerminal matches step terminal events for the provided flow steps
func StepTerminal(steps ...api.FlowStep) EventFilter {
	return And(
		Type(
			api.EventTypeStepCompleted,
			api.EventTypeStepFailed,
			api.EventTypeStepSkipped,
		),
		FlowStep(steps...),
	)
}

// WorkStarted matches work started events for the provided flow steps
func WorkStarted(steps ...api.FlowStep) EventFilter {
	return And(Type(api.EventTypeWorkStarted), FlowStep(steps...))
}

// WorkSucceeded matches work succeeded events for the provided flow steps
func WorkSucceeded(steps ...api.FlowStep) EventFilter {
	return And(Type(api.EventTypeWorkSucceeded), FlowStep(steps...))
}

// WorkFailed matches work failed events for the provided flow steps
func WorkFailed(steps ...api.FlowStep) EventFilter {
	return And(Type(api.EventTypeWorkFailed), FlowStep(steps...))
}

// WorkRetryScheduled matches retry scheduled events for flow steps
func WorkRetryScheduled(steps ...api.FlowStep) EventFilter {
	return And(
		Type(api.EventTypeRetryScheduled),
		FlowStep(steps...),
	)
}

// WorkRetryScheduledAny matches retry scheduled events for flow steps
func WorkRetryScheduledAny(steps ...api.FlowStep) EventFilter {
	return And(
		Type(api.EventTypeRetryScheduled),
		FlowStepAny(steps...),
	)
}

// FlowID matches events for the provided flow IDs
func FlowID(ids ...api.FlowID) EventFilter {
	expected := make(util.Set[api.FlowID], len(ids))
	for _, flowID := range ids {
		expected.Add(flowID)
	}
	return Predicate(func(data flowEvent) bool {
		if expected.Contains(data.FlowID) {
			expected.Remove(data.FlowID)
			return true
		}
		return false
	})
}

// FlowStep matches events for the provided flow steps
func FlowStep(steps ...api.FlowStep) EventFilter {
	expected := make(util.Set[api.FlowStep], len(steps))
	for _, step := range steps {
		expected.Add(step)
	}
	return Predicate(func(data stepEvent) bool {
		key := api.FlowStep{FlowID: data.FlowID, StepID: data.StepID}
		if expected.Contains(key) {
			expected.Remove(key)
			return true
		}
		return false
	})
}

// FlowStepAny matches events for the provided flow steps
func FlowStepAny(steps ...api.FlowStep) EventFilter {
	expected := make(util.Set[api.FlowStep], len(steps))
	for _, step := range steps {
		expected.Add(step)
	}
	return Predicate(func(data stepEvent) bool {
		key := api.FlowStep{FlowID: data.FlowID, StepID: data.StepID}
		return expected.Contains(key)
	})
}

// StepHealth matches step health change events for a step and status
func StepHealth(stepID api.StepID, status api.HealthStatus) EventFilter {
	return Predicate(func(data api.StepHealthChangedEvent) bool {
		return data.StepID == stepID && data.Status == status
	})
}

// StepHealthChanged matches step health change events for a step/status
func StepHealthChanged(stepID api.StepID, status api.HealthStatus) EventFilter {
	return And(
		Type(api.EventTypeStepHealthChanged),
		StepHealth(stepID, status),
	)
}

// Predicate creates a filter that unmarshals event data and applies pred
func Predicate[T any](pred predicate[T]) EventFilter {
	return func(ev *timebox.Event) bool {
		var data T
		if json.Unmarshal(ev.Data, &data) != nil {
			return false
		}
		return pred(data)
	}
}

func filterEventTypes(eventTypes ...api.EventType) EventFilter {
	if len(eventTypes) == 0 {
		return func(*timebox.Event) bool { return false }
	}
	lookup := make(util.Set[timebox.EventType], len(eventTypes))
	for _, et := range eventTypes {
		lookup.Add(timebox.EventType(et))
	}
	return func(ev *timebox.Event) bool {
		if ev == nil {
			return false
		}
		return lookup.Contains(ev.Type)
	}
}

func filterAggregate(id timebox.AggregateID) EventFilter {
	return func(ev *timebox.Event) bool {
		return ev != nil && ev.AggregateID.Equal(id)
	}
}
