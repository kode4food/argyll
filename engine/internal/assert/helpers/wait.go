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
func (w *Wait) ForEvents(filter EventFilter, count int) {
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

// ForFlowEvents waits for one event per flow ID for the given types
func (w *Wait) ForFlowEvents(ids []api.FlowID, eventTypes ...api.EventType) {
	w.t.Helper()

	expected := make(util.Set[api.FlowID], len(ids))
	for _, flowID := range ids {
		expected.Add(flowID)
	}

	WaitForEventData(w,
		filterEventTypes(eventTypes...),
		func(data flowEvent) bool {
			if expected.Contains(data.FlowID) {
				expected.Remove(data.FlowID)
				return true
			}
			return false
		},
		len(ids),
	)
}

// ForFlowStarted waits for flow start events for the given IDs
func (w *Wait) ForFlowStarted(ids ...api.FlowID) {
	w.t.Helper()
	w.ForFlowEvents(ids, api.EventTypeFlowStarted)
}

// ForFlowActivated waits for flow activation events for the given IDs
func (w *Wait) ForFlowActivated(ids ...api.FlowID) {
	w.t.Helper()
	w.forEngineFlowEvents(ids, api.EventTypeFlowActivated)
}

// ForFlowDeactivated waits for flow deactivation events for the given IDs
func (w *Wait) ForFlowDeactivated(ids ...api.FlowID) {
	w.t.Helper()
	w.forEngineFlowEvents(ids, api.EventTypeFlowDeactivated)
}

// ForFlowTerminal waits for flow completion or failure events
func (w *Wait) ForFlowTerminal(ids ...api.FlowID) {
	w.t.Helper()
	w.ForFlowEvents(ids,
		api.EventTypeFlowCompleted, api.EventTypeFlowFailed,
	)
}

// ForFlowCompleted waits for flow completion events for the given IDs
func (w *Wait) ForFlowCompleted(ids ...api.FlowID) {
	w.t.Helper()
	w.ForFlowEvents(ids, api.EventTypeFlowCompleted)
}

// ForFlowFailed waits for flow failure events for the given IDs
func (w *Wait) ForFlowFailed(ids ...api.FlowID) {
	w.t.Helper()
	w.ForFlowEvents(ids, api.EventTypeFlowFailed)
}

// ForStepEvents waits for matching step events for the given flow/step
func (w *Wait) ForStepEvents(
	fs api.FlowStep, count int, eventTypes ...api.EventType,
) {
	w.t.Helper()

	WaitForEventData(w,
		filterEventTypes(eventTypes...),
		func(data stepEvent) bool {
			return data.FlowID == fs.FlowID && data.StepID == fs.StepID
		},
		count,
	)
}

// ForWorkEvents waits for matching work events for the given flow/step
func (w *Wait) ForWorkEvents(
	fs api.FlowStep, count int, eventTypes ...api.EventType,
) {
	w.t.Helper()
	w.ForStepEvents(fs, count, eventTypes...)
}

// ForStepStartedEvent waits for step started events
func (w *Wait) ForStepStartedEvent(fs api.FlowStep) {
	w.t.Helper()
	w.ForStepEvents(fs, 1, api.EventTypeStepStarted)
}

// ForStepTerminalEvent waits for step completion, failure, or skip events
func (w *Wait) ForStepTerminalEvent(fs api.FlowStep) {
	w.t.Helper()
	w.ForStepEvents(fs, 1,
		api.EventTypeStepCompleted, api.EventTypeStepFailed,
		api.EventTypeStepSkipped,
	)
}

// ForWorkStarted waits for work started events
func (w *Wait) ForWorkStarted(fs api.FlowStep, count int) {
	w.t.Helper()
	w.ForWorkEvents(fs, count, api.EventTypeWorkStarted)
}

// ForWorkSucceeded waits for work succeeded events
func (w *Wait) ForWorkSucceeded(fs api.FlowStep, count int) {
	w.t.Helper()
	w.ForWorkEvents(fs, count, api.EventTypeWorkSucceeded)
}

// ForWorkFailed waits for work failed events
func (w *Wait) ForWorkFailed(fs api.FlowStep, count int) {
	w.t.Helper()
	w.ForWorkEvents(fs, count, api.EventTypeWorkFailed)
}

// ForWorkRetryScheduled waits for retry scheduled events
func (w *Wait) ForWorkRetryScheduled(fs api.FlowStep, count int) {
	w.t.Helper()
	w.ForWorkEvents(fs, count, api.EventTypeRetryScheduled)
}

// ForStepHealth waits for a step health change to the target status
func (w *Wait) ForStepHealth(stepID api.StepID, status api.HealthStatus) {
	w.t.Helper()

	WaitForEventData(w,
		filterEventTypes(api.EventTypeStepHealthChanged),
		func(data api.StepHealthChangedEvent) bool {
			return data.StepID == stepID && data.Status == status
		},
		1,
	)
}

// ForEngineEvents waits for engine aggregate events of the given types
func (w *Wait) ForEngineEvents(count int, eventTypes ...api.EventType) {
	w.t.Helper()
	filter := func(ev *timebox.Event) bool {
		return filterAggregate(events.EngineKey)(ev) &&
			filterEventTypes(eventTypes...)(ev)
	}
	w.ForEvents(filter, count)
}

func (w *Wait) forEngineFlowEvents(
	ids []api.FlowID, eventTypes ...api.EventType,
) {
	w.t.Helper()

	expected := make(util.Set[api.FlowID], len(ids))
	for _, flowID := range ids {
		expected.Add(flowID)
	}

	typeFilter := func(ev *timebox.Event) bool {
		return filterAggregate(events.EngineKey)(ev) &&
			filterEventTypes(eventTypes...)(ev)
	}
	WaitForEventData(w, typeFilter,
		func(data flowEvent) bool {
			if expected.Contains(data.FlowID) {
				expected.Remove(data.FlowID)
				return true
			}
			return false
		},
		len(ids),
	)
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

func (e *TestEngineEnv) WaitForFlowStarted(ids []api.FlowID, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForFlowStarted(ids...)
	})
}

func (e *TestEngineEnv) WaitForFlowActivated(ids []api.FlowID, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForFlowActivated(ids...)
	})
}

func (e *TestEngineEnv) WaitForFlowDeactivated(ids []api.FlowID, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForFlowDeactivated(ids...)
	})
}

func (e *TestEngineEnv) WaitForFlowTerminal(ids []api.FlowID, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForFlowTerminal(ids...)
	})
}

func (e *TestEngineEnv) WaitForFlowCompleted(ids []api.FlowID, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForFlowCompleted(ids...)
	})
}

func (e *TestEngineEnv) WaitForFlowFailed(ids []api.FlowID, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForFlowFailed(ids...)
	})
}

func (e *TestEngineEnv) WaitForStepStartedEvent(fs api.FlowStep, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForStepStartedEvent(fs)
	})
}

func (e *TestEngineEnv) WaitForStepTerminalEvent(fs api.FlowStep, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForStepTerminalEvent(fs)
	})
}

func (e *TestEngineEnv) WaitForWorkStarted(
	fs api.FlowStep, count int, fn func(),
) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForWorkStarted(fs, count)
	})
}

func (e *TestEngineEnv) WaitForWorkSucceeded(
	fs api.FlowStep, count int, fn func(),
) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForWorkSucceeded(fs, count)
	})
}

func (e *TestEngineEnv) WaitForWorkFailed(
	fs api.FlowStep, count int, fn func(),
) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForWorkFailed(fs, count)
	})
}

func (e *TestEngineEnv) WaitForWorkRetryScheduled(
	fs api.FlowStep, count int, fn func(),
) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForWorkRetryScheduled(fs, count)
	})
}

func (e *TestEngineEnv) WaitForStepHealth(
	stepID api.StepID, status api.HealthStatus, fn func(),
) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForStepHealth(stepID, status)
	})
}

func (e *TestEngineEnv) WaitForEngineEvents(
	count int, eventTypes []api.EventType, fn func(),
) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		wait := WaitOn(e.T, consumer)
		wait.ForEngineEvents(count, eventTypes...)
	})
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
		wait.ForFlowTerminal(flowID)
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
		wait.ForStepStartedEvent(fs)
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
		wait.ForStepTerminalEvent(api.FlowStep{FlowID: flowID, StepID: stepID})
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

// WaitForEventData waits for matching event data for the given filter
func WaitForEventData[T any](
	w *Wait, typeFilter EventFilter, pred predicate[T], count int,
) {
	w.t.Helper()
	filter := EventDataFilter(typeFilter, pred)
	w.ForEvents(filter, count)
}

// EventDataFilter creates a filter that unmarshals event data into T
func EventDataFilter[T any](filter EventFilter, pred predicate[T]) EventFilter {
	if pred == nil {
		pred = func(T) bool { return true }
	}

	return func(ev *timebox.Event) bool {
		if !filter(ev) {
			return false
		}
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
