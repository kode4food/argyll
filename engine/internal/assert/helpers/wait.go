package helpers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/events"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// EventWaiter waits for events matching a filter. Create before triggering the
// action
type EventWaiter[T any] struct {
	consumer topic.Consumer[*timebox.Event]
	filter   events.EventFilter
	getState func(context.Context) (T, error)
	desc     string // for error messages
}

// Wait blocks until a matching event and returns the state
func (w *EventWaiter[T]) Wait(
	t *testing.T, ctx context.Context, timeout time.Duration,
) T {
	t.Helper()
	defer w.consumer.Close()

	deadline := time.After(timeout)
	for {
		select {
		case event := <-w.consumer.Receive():
			if event != nil && w.filter(event) {
				state, err := w.getState(ctx)
				assert.NoError(t, err)
				return state
			}
		case <-deadline:
			t.Fatalf("timeout waiting for %s", w.desc)
		case <-ctx.Done():
			t.FailNow()
		}
	}
}

// SubscribeToFlowStatus creates a waiter for flow completion/failure
func (e *TestEngineEnv) SubscribeToFlowStatus(
	flowID api.FlowID,
) *EventWaiter[*api.FlowState] {
	return &EventWaiter[*api.FlowState]{
		consumer: e.EventHub.NewConsumer(),
		filter: filterFlowEvents(
			flowID, api.EventTypeFlowCompleted, api.EventTypeFlowFailed,
		),
		getState: func(ctx context.Context) (*api.FlowState, error) {
			return e.Engine.GetFlowState(ctx, flowID)
		},
		desc: string(flowID),
	}
}

// SubscribeToStepStarted creates a waiter for step start events
func (e *TestEngineEnv) SubscribeToStepStarted(
	flowID api.FlowID, stepID api.StepID,
) *EventWaiter[*api.ExecutionState] {
	return &EventWaiter[*api.ExecutionState]{
		consumer: e.EventHub.NewConsumer(),
		filter:   filterStepEvents(flowID, stepID, api.EventTypeStepStarted),
		getState: func(ctx context.Context) (*api.ExecutionState, error) {
			flow, err := e.Engine.GetFlowState(ctx, flowID)
			if err != nil {
				return nil, err
			}
			return flow.Executions[stepID], nil
		},
		desc: string(stepID),
	}
}

// SubscribeToStepStatus creates a waiter for step completion/failure/skip
func (e *TestEngineEnv) SubscribeToStepStatus(
	flowID api.FlowID, stepID api.StepID,
) *EventWaiter[*api.ExecutionState] {
	return &EventWaiter[*api.ExecutionState]{
		consumer: e.EventHub.NewConsumer(),
		filter: filterStepEvents(
			flowID, stepID, api.EventTypeStepCompleted,
			api.EventTypeStepFailed, api.EventTypeStepSkipped,
		),
		getState: func(ctx context.Context) (*api.ExecutionState, error) {
			flow, err := e.Engine.GetFlowState(ctx, flowID)
			if err != nil {
				return nil, err
			}
			return flow.Executions[stepID], nil
		},
		desc: string(stepID),
	}
}

// Convenience methods that subscribe and wait in one call

func (e *TestEngineEnv) WaitForFlowStatus(
	t *testing.T, ctx context.Context, flowID api.FlowID, timeout time.Duration,
) *api.FlowState {
	t.Helper()
	start := time.Now()
	state := e.SubscribeToFlowStatus(flowID).Wait(t, ctx, timeout)
	if state.Status == api.FlowCompleted || state.Status == api.FlowFailed {
		return state
	}

	remaining := timeout - time.Since(start)
	if remaining <= 0 {
		return state
	}

	deadline := time.Now().Add(remaining)
	for time.Now().Before(deadline) {
		cur, err := e.Engine.GetFlowState(ctx, flowID)
		assert.NoError(t, err)
		if cur.Status == api.FlowCompleted || cur.Status == api.FlowFailed {
			return cur
		}
		time.Sleep(10 * time.Millisecond)
	}
	return state
}

func (e *TestEngineEnv) WaitForStepStarted(
	t *testing.T, ctx context.Context, flowID api.FlowID, stepID api.StepID,
	timeout time.Duration,
) *api.ExecutionState {
	t.Helper()
	return e.SubscribeToStepStarted(flowID, stepID).Wait(t, ctx, timeout)
}

func (e *TestEngineEnv) WaitForStepStatus(
	t *testing.T, ctx context.Context, flowID api.FlowID, stepID api.StepID,
	timeout time.Duration,
) *api.ExecutionState {
	t.Helper()
	return e.SubscribeToStepStatus(flowID, stepID).Wait(t, ctx, timeout)
}

// Filter helpers

func filterFlowEvents(
	flowID api.FlowID, eventTypes ...api.EventType,
) events.EventFilter {
	typeFilter := events.FilterEvents(toTimeboxTypes(eventTypes)...)
	return func(ev *timebox.Event) bool {
		if !typeFilter(ev) {
			return false
		}
		var e api.FlowCompletedEvent
		if json.Unmarshal(ev.Data, &e) != nil {
			return false
		}
		return e.FlowID == flowID
	}
}

func filterStepEvents(
	flowID api.FlowID, stepID api.StepID, eventTypes ...api.EventType,
) events.EventFilter {
	typeFilter := events.FilterEvents(toTimeboxTypes(eventTypes)...)
	return func(ev *timebox.Event) bool {
		if !typeFilter(ev) {
			return false
		}
		var e api.StepStartedEvent
		if json.Unmarshal(ev.Data, &e) != nil {
			return false
		}
		return e.FlowID == flowID && e.StepID == stepID
	}
}

func toTimeboxTypes(eventTypes []api.EventType) []timebox.EventType {
	result := make([]timebox.EventType, len(eventTypes))
	for i, et := range eventTypes {
		result[i] = timebox.EventType(et)
	}
	return result
}
