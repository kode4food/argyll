package helpers

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

const DefaultWaitTimeout = time.Second * 5

func WaitOn(t *testing.T, consumer *timebox.Consumer) *Wait {
	return &Wait{
		t:        t,
		consumer: consumer,
		timeout:  DefaultWaitTimeout,
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

// WaitFor runs fn and waits for a matching event
func (e *TestEngineEnv) WaitFor(filter EventFilter, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForEvent(filter)
	})
}

// WaitForCount runs fn and waits for count events
func (e *TestEngineEnv) WaitForCount(count int, filter EventFilter, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForEvents(count, filter)
	})
}

// WithConsumer provides a scoped event consumer for tests
func (e *TestEngineEnv) WithConsumer(fn func(*timebox.Consumer)) {
	e.T.Helper()
	consumer := e.EventHub.NewConsumer()
	defer consumer.Close()
	fn(consumer)
}

// WaitAfterAll creates multiple consumers, runs fn, and waits with each
func (e *TestEngineEnv) WaitAfterAll(count int, fn func([]*Wait)) {
	e.T.Helper()

	consumers := make([]*timebox.Consumer, count)
	waits := make([]*Wait, count)
	for i := 0; i < count; i++ {
		consumer := e.EventHub.NewConsumer()
		consumers[i] = consumer
		waits[i] = WaitOn(e.T, consumer)
	}
	defer func() {
		for _, consumer := range consumers {
			consumer.Close()
		}
	}()

	fn(waits)
}

func (e *TestEngineEnv) WaitForFlowStatus(
	flowID api.FlowID, fn func(),
) *api.FlowState {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		state, err := e.Engine.GetFlowState(flowID)
		if err == nil && isFlowTerminal(state) {
			return
		}
		wait.ForEvent(FlowTerminal(flowID))
	})

	state, err := e.Engine.GetFlowState(flowID)
	if err != nil {
		e.T.Fatalf("failed to fetch flow %s: %v", flowID, err)
	}
	if !isFlowTerminal(state) {
		e.T.Fatalf("flow %s not terminal after event", flowID)
	}
	return state
}

// WaitForStepStarted waits for a step to start and returns the execution
func (e *TestEngineEnv) WaitForStepStarted(
	fs api.FlowStep, fn func(),
) *api.ExecutionState {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		exec, err := e.getExecutionState(fs.FlowID, fs.StepID)
		if err == nil && exec != nil && isStepStarted(exec.Status) {
			return
		}
		wait.ForEvent(StepStarted(fs))
	})

	exec, err := e.getExecutionState(fs.FlowID, fs.StepID)
	if err != nil {
		e.T.Fatalf("failed to fetch execution %s: %v", fs.StepID, err)
	}
	if exec == nil || !isStepStarted(exec.Status) {
		e.T.Fatalf("execution %s not started after event", fs.StepID)
	}
	return exec
}

// WaitForStepStatus waits for a step to finish and returns the execution
func (e *TestEngineEnv) WaitForStepStatus(
	flowID api.FlowID, stepID api.StepID, fn func(),
) *api.ExecutionState {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		exec, err := e.getExecutionState(flowID, stepID)
		if err == nil && exec != nil && isStepTerminal(exec.Status) {
			return
		}
		wait.ForEvent(StepTerminal(
			api.FlowStep{FlowID: flowID, StepID: stepID},
		))
	})

	exec, err := e.getExecutionState(flowID, stepID)
	if err != nil {
		e.T.Fatalf("failed to fetch execution %s: %v", stepID, err)
	}
	if exec == nil || !isStepTerminal(exec.Status) {
		e.T.Fatalf("execution %s not terminal after event", stepID)
	}
	return exec
}

func (e *TestEngineEnv) getExecutionState(
	flowID api.FlowID, stepID api.StepID,
) (*api.ExecutionState, error) {
	flow, err := e.Engine.GetFlowState(flowID)
	if err != nil {
		return nil, err
	}
	return flow.Executions[stepID], nil
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

func isFlowTerminal(state *api.FlowState) bool {
	if state == nil {
		return false
	}
	return state.Status == api.FlowCompleted ||
		state.Status == api.FlowFailed
}

func isStepStarted(status api.StepStatus) bool {
	return status != api.StepPending
}

func isStepTerminal(status api.StepStatus) bool {
	return status == api.StepCompleted ||
		status == api.StepFailed ||
		status == api.StepSkipped
}
