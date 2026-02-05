package helpers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kode4food/timebox"

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

// WaitForEvents waits for matching events from the consumer
func WaitForEvents(
	t *testing.T, consumer *timebox.Consumer, filter func(*timebox.Event) bool,
	count int, timeout time.Duration,
) {
	t.Helper()
	defer consumer.Close()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for seen := 0; seen < count; {
		select {
		case ev, ok := <-consumer.Receive():
			if !ok {
				t.Fatalf(
					"event consumer closed before receiving %d events", count,
				)
			}
			if ev == nil || !filter(ev) {
				continue
			}
			seen++
		case <-deadline.C:
			t.Fatalf("timeout waiting for %d events", count)
		}
	}
}

// EventDataFilter creates a filter that unmarshals event data into T
func EventDataFilter[T any](
	typeFilter func(*timebox.Event) bool, predicate func(T) bool,
) func(*timebox.Event) bool {
	if predicate == nil {
		predicate = func(T) bool { return true }
	}

	return func(ev *timebox.Event) bool {
		if !typeFilter(ev) {
			return false
		}
		var data T
		if json.Unmarshal(ev.Data, &data) != nil {
			return false
		}
		return predicate(data)
	}
}

// WaitForEventData waits for matching event data for the given filter
func WaitForEventData[T any](
	t *testing.T, consumer *timebox.Consumer,
	typeFilter func(*timebox.Event) bool,
	predicate func(T) bool, count int, timeout time.Duration,
) {
	t.Helper()
	filter := EventDataFilter(typeFilter, predicate)
	WaitForEvents(t, consumer, filter, count, timeout)
}

// WaitForFlowEvents waits for one event per flow ID for the given types
func WaitForFlowEvents(
	t *testing.T, consumer *timebox.Consumer, flowIDs []api.FlowID,
	timeout time.Duration, eventTypes ...api.EventType,
) {
	t.Helper()

	expected := make(map[api.FlowID]struct{}, len(flowIDs))
	for _, flowID := range flowIDs {
		expected[flowID] = struct{}{}
	}

	WaitForEventData(t, consumer,
		filterEventTypes(eventTypes...),
		func(data flowEvent) bool {
			if _, ok := expected[data.FlowID]; ok {
				delete(expected, data.FlowID)
				return true
			}
			return false
		},
		len(flowIDs), timeout,
	)
}

func waitForEngineFlowEvents(
	t *testing.T, consumer *timebox.Consumer, flowIDs []api.FlowID,
	timeout time.Duration, eventTypes ...api.EventType,
) {
	t.Helper()

	expected := make(map[api.FlowID]struct{}, len(flowIDs))
	for _, flowID := range flowIDs {
		expected[flowID] = struct{}{}
	}

	typeFilter := func(ev *timebox.Event) bool {
		return filterAggregate(events.EngineID)(ev) &&
			filterEventTypes(eventTypes...)(ev)
	}
	WaitForEventData(t, consumer, typeFilter,
		func(data flowEvent) bool {
			if _, ok := expected[data.FlowID]; ok {
				delete(expected, data.FlowID)
				return true
			}
			return false
		},
		len(flowIDs), timeout,
	)
}

// WaitForFlowStarted waits for flow start events for the given IDs
func WaitForFlowStarted(
	t *testing.T, consumer *timebox.Consumer, timeout time.Duration,
	flowIDs ...api.FlowID,
) {
	t.Helper()
	WaitForFlowEvents(t,
		consumer, flowIDs, timeout, api.EventTypeFlowStarted,
	)
}

// WaitForFlowActivated waits for flow activation events for the given IDs
func WaitForFlowActivated(
	t *testing.T, consumer *timebox.Consumer, timeout time.Duration,
	flowIDs ...api.FlowID,
) {
	t.Helper()
	waitForEngineFlowEvents(t,
		consumer, flowIDs, timeout, api.EventTypeFlowActivated,
	)
}

// WaitForFlowDeactivated waits for flow deactivation events for the given IDs
func WaitForFlowDeactivated(
	t *testing.T, consumer *timebox.Consumer, timeout time.Duration,
	flowIDs ...api.FlowID,
) {
	t.Helper()
	waitForEngineFlowEvents(t,
		consumer, flowIDs, timeout, api.EventTypeFlowDeactivated,
	)
}

// WaitForFlowTerminal waits for flow completion or failure events
func WaitForFlowTerminal(
	t *testing.T, consumer *timebox.Consumer, timeout time.Duration,
	flowIDs ...api.FlowID,
) {
	t.Helper()
	WaitForFlowEvents(t,
		consumer, flowIDs, timeout, api.EventTypeFlowCompleted,
		api.EventTypeFlowFailed,
	)
}

// WaitForFlowCompleted waits for flow completion events for the given IDs
func WaitForFlowCompleted(
	t *testing.T, consumer *timebox.Consumer, timeout time.Duration,
	flowIDs ...api.FlowID,
) {
	t.Helper()
	WaitForFlowEvents(t,
		consumer, flowIDs, timeout, api.EventTypeFlowCompleted,
	)
}

// WaitForFlowFailed waits for flow failure events for the given IDs
func WaitForFlowFailed(
	t *testing.T, consumer *timebox.Consumer, timeout time.Duration,
	flowIDs ...api.FlowID,
) {
	t.Helper()
	WaitForFlowEvents(t,
		consumer, flowIDs, timeout, api.EventTypeFlowFailed,
	)
}

// WaitForStepEvents waits for matching step events for the given flow/step
func WaitForStepEvents(
	t *testing.T, consumer *timebox.Consumer, flowID api.FlowID,
	stepID api.StepID, count int, timeout time.Duration,
	eventTypes ...api.EventType,
) {
	t.Helper()

	WaitForEventData(t, consumer,
		filterEventTypes(eventTypes...),
		func(data stepEvent) bool {
			return data.FlowID == flowID && data.StepID == stepID
		},
		count, timeout,
	)
}

// WaitForWorkEvents waits for matching work events for the given flow/step
func WaitForWorkEvents(
	t *testing.T, consumer *timebox.Consumer, flowID api.FlowID,
	stepID api.StepID, count int, timeout time.Duration,
	eventTypes ...api.EventType,
) {
	t.Helper()
	WaitForStepEvents(t,
		consumer, flowID, stepID, count, timeout, eventTypes...,
	)
}

// WaitForStepStartedEvent waits for step started events
func WaitForStepStartedEvent(
	t *testing.T, consumer *timebox.Consumer, flowID api.FlowID,
	stepID api.StepID, timeout time.Duration,
) {
	t.Helper()
	WaitForStepEvents(t,
		consumer, flowID, stepID, 1, timeout, api.EventTypeStepStarted,
	)
}

// WaitForStepTerminalEvent waits for step completion, failure, or skip events
func WaitForStepTerminalEvent(
	t *testing.T, consumer *timebox.Consumer, flowID api.FlowID,
	stepID api.StepID, timeout time.Duration,
) {
	t.Helper()
	WaitForStepEvents(t,
		consumer, flowID, stepID, 1, timeout, api.EventTypeStepCompleted,
		api.EventTypeStepFailed, api.EventTypeStepSkipped,
	)
}

// WaitForWorkStarted waits for work started events
func WaitForWorkStarted(
	t *testing.T, consumer *timebox.Consumer, flowID api.FlowID,
	stepID api.StepID, count int, timeout time.Duration,
) {
	t.Helper()
	WaitForWorkEvents(t,
		consumer, flowID, stepID, count, timeout, api.EventTypeWorkStarted,
	)
}

// WaitForWorkSucceeded waits for work succeeded events
func WaitForWorkSucceeded(
	t *testing.T, consumer *timebox.Consumer, flowID api.FlowID,
	stepID api.StepID, count int, timeout time.Duration,
) {
	t.Helper()
	WaitForWorkEvents(t,
		consumer, flowID, stepID, count, timeout, api.EventTypeWorkSucceeded,
	)
}

// WaitForWorkFailed waits for work failed events
func WaitForWorkFailed(
	t *testing.T, consumer *timebox.Consumer, flowID api.FlowID,
	stepID api.StepID, count int, timeout time.Duration,
) {
	t.Helper()
	WaitForWorkEvents(t,
		consumer, flowID, stepID, count, timeout, api.EventTypeWorkFailed,
	)
}

// WaitForWorkRetryScheduled waits for retry scheduled events
func WaitForWorkRetryScheduled(
	t *testing.T, consumer *timebox.Consumer, flowID api.FlowID,
	stepID api.StepID, count int, timeout time.Duration,
) {
	t.Helper()
	WaitForWorkEvents(t,
		consumer, flowID, stepID, count, timeout,
		api.EventTypeRetryScheduled,
	)
}

// WaitForStepHealth waits for a step health change to the target status
func WaitForStepHealth(
	t *testing.T, consumer *timebox.Consumer, stepID api.StepID,
	status api.HealthStatus, timeout time.Duration,
) {
	t.Helper()

	WaitForEventData(t, consumer,
		filterEventTypes(api.EventTypeStepHealthChanged),
		func(data api.StepHealthChangedEvent) bool {
			return data.StepID == stepID && data.Status == status
		},
		1, timeout,
	)
}

// WaitForEngineEvents waits for engine aggregate events of the given types
func WaitForEngineEvents(
	t *testing.T, consumer *timebox.Consumer, count int,
	timeout time.Duration, eventTypes ...api.EventType,
) {
	t.Helper()
	filter := func(ev *timebox.Event) bool {
		return filterAggregate(events.EngineID)(ev) &&
			filterEventTypes(eventTypes...)(ev)
	}
	WaitForEvents(t, consumer, filter, count, timeout)
}

// WaitForFlowStatus waits for a flow to complete or fail and returns it
func (e *TestEngineEnv) WaitForFlowStatus(
	t *testing.T, flowID api.FlowID, timeout time.Duration,
) *api.FlowState {
	t.Helper()
	consumer := e.EventHub.NewConsumer()
	defer consumer.Close()

	filter := filterFlowEvents(
		flowID, api.EventTypeFlowCompleted, api.EventTypeFlowFailed,
	)
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		state, err := e.Engine.GetFlowState(flowID)
		if err == nil && flowTerminal(state) {
			return state
		}

		select {
		case ev, ok := <-consumer.Receive():
			if !ok {
				t.Fatalf(
					"event consumer closed before flow %s completed", flowID,
				)
			}
			if ev == nil || !filter(ev) {
				continue
			}
		case <-deadline.C:
			t.Fatalf("timeout waiting for %s", flowID)
		}
	}
}

// WaitForStepStarted waits for a step to start and returns the execution
func (e *TestEngineEnv) WaitForStepStarted(
	t *testing.T, flowID api.FlowID, stepID api.StepID, timeout time.Duration,
) *api.ExecutionState {
	t.Helper()
	consumer := e.EventHub.NewConsumer()
	defer consumer.Close()

	filter := filterStepEvents(flowID, stepID, api.EventTypeStepStarted)
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		exec, err := e.getExecutionState(flowID, stepID)
		if err == nil && exec != nil && stepStarted(exec.Status) {
			return exec
		}

		select {
		case ev, ok := <-consumer.Receive():
			if !ok {
				t.Fatalf(
					"event consumer closed before %s started", stepID,
				)
			}
			if ev == nil || !filter(ev) {
				continue
			}
		case <-deadline.C:
			t.Fatalf("timeout waiting for %s", stepID)
		}
	}
}

// WaitForStepStatus waits for a step to finish and returns the execution
func (e *TestEngineEnv) WaitForStepStatus(
	t *testing.T, flowID api.FlowID, stepID api.StepID, timeout time.Duration,
) *api.ExecutionState {
	t.Helper()
	consumer := e.EventHub.NewConsumer()
	defer consumer.Close()

	filter := filterStepEvents(
		flowID, stepID, api.EventTypeStepCompleted,
		api.EventTypeStepFailed, api.EventTypeStepSkipped,
	)
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		exec, err := e.getExecutionState(flowID, stepID)
		if err == nil && exec != nil && stepTerminal(exec.Status) {
			return exec
		}

		select {
		case ev, ok := <-consumer.Receive():
			if !ok {
				t.Fatalf(
					"event consumer closed before %s finished", stepID,
				)
			}
			if ev == nil || !filter(ev) {
				continue
			}
		case <-deadline.C:
			t.Fatalf("timeout waiting for %s", stepID)
		}
	}
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

func flowTerminal(state *api.FlowState) bool {
	if state == nil {
		return false
	}
	return state.Status == api.FlowCompleted ||
		state.Status == api.FlowFailed
}

func stepStarted(status api.StepStatus) bool {
	return status != api.StepPending
}

func stepTerminal(status api.StepStatus) bool {
	return status == api.StepCompleted ||
		status == api.StepFailed ||
		status == api.StepSkipped
}

func filterFlowEvents(
	flowID api.FlowID, eventTypes ...api.EventType,
) func(*timebox.Event) bool {
	typeFilter := filterEventTypes(eventTypes...)
	return EventDataFilter(typeFilter, func(data flowEvent) bool {
		return data.FlowID == flowID
	})
}

func filterStepEvents(
	flowID api.FlowID, stepID api.StepID, eventTypes ...api.EventType,
) func(*timebox.Event) bool {
	typeFilter := filterEventTypes(eventTypes...)
	return EventDataFilter(typeFilter, func(data stepEvent) bool {
		return data.FlowID == flowID && data.StepID == stepID
	})
}

func filterAggregate(id timebox.AggregateID) func(*timebox.Event) bool {
	return func(ev *timebox.Event) bool {
		return ev != nil && ev.AggregateID.Equal(id)
	}
}

func filterEventTypes(
	eventTypes ...api.EventType,
) func(*timebox.Event) bool {
	if len(eventTypes) == 0 {
		return func(*timebox.Event) bool { return false }
	}
	lookup := map[timebox.EventType]struct{}{}
	for _, et := range eventTypes {
		lookup[timebox.EventType(et)] = struct{}{}
	}
	return func(ev *timebox.Event) bool {
		if ev == nil {
			return false
		}
		_, ok := lookup[ev.Type]
		return ok
	}
}
