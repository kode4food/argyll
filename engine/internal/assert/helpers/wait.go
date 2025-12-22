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

type (
	// FlowWaiter waits for flow events. Create before triggering the action
	FlowWaiter struct {
		env      *TestEngineEnv
		consumer topic.Consumer[*timebox.Event]
		flowID   api.FlowID
		filter   events.EventFilter
	}

	// StepWaiter waits for step events. Create before triggering the action.
	StepWaiter struct {
		env      *TestEngineEnv
		consumer topic.Consumer[*timebox.Event]
		flowID   api.FlowID
		stepID   api.StepID
		filter   events.EventFilter
	}
)

// SubscribeToFlowStatus creates a waiter for flow completion/failure. Call
// this BEFORE the action that triggers events
func (env *TestEngineEnv) SubscribeToFlowStatus(flowID api.FlowID) *FlowWaiter {
	return &FlowWaiter{
		env:      env,
		consumer: (*env.EventHub).NewConsumer(),
		flowID:   flowID,
		filter: filterFlowEvents(
			flowID, api.EventTypeFlowCompleted, api.EventTypeFlowFailed,
		),
	}
}

// SubscribeToStepStarted creates a waiter for step start events
func (env *TestEngineEnv) SubscribeToStepStarted(
	flowID api.FlowID, stepID api.StepID,
) *StepWaiter {
	return &StepWaiter{
		env:      env,
		consumer: (*env.EventHub).NewConsumer(),
		flowID:   flowID,
		stepID:   stepID,
		filter:   filterStepEvents(flowID, stepID, api.EventTypeStepStarted),
	}
}

// SubscribeToStepStatus creates a waiter for step completion/failure/skip
func (env *TestEngineEnv) SubscribeToStepStatus(
	flowID api.FlowID, stepID api.StepID,
) *StepWaiter {
	return &StepWaiter{
		env:      env,
		consumer: (*env.EventHub).NewConsumer(),
		flowID:   flowID,
		stepID:   stepID,
		filter: filterStepEvents(
			flowID, stepID, api.EventTypeStepCompleted,
			api.EventTypeStepFailed, api.EventTypeStepSkipped,
		),
	}
}

// Wait blocks until the flow reaches a terminal status or times out
func (w *FlowWaiter) Wait(
	t *testing.T, ctx context.Context, timeout time.Duration,
) *api.FlowState {
	t.Helper()
	defer w.consumer.Close()

	deadline := time.After(timeout)
	for {
		select {
		case event := <-w.consumer.Receive():
			if event != nil && w.filter(event) {
				flow, err := w.env.Engine.GetFlowState(ctx, w.flowID)
				assert.NoError(t, err)
				return flow
			}
		case <-deadline:
			t.Fatalf("timeout waiting for flow %s", w.flowID)
			return nil
		case <-ctx.Done():
			t.FailNow()
			return nil
		}
	}
}

// Wait blocks until the step event or times out
func (w *StepWaiter) Wait(
	t *testing.T, ctx context.Context, timeout time.Duration,
) *api.ExecutionState {
	t.Helper()
	defer w.consumer.Close()

	deadline := time.After(timeout)
	for {
		select {
		case event := <-w.consumer.Receive():
			if event != nil && w.filter(event) {
				flow, err := w.env.Engine.GetFlowState(ctx, w.flowID)
				assert.NoError(t, err)
				return flow.Executions[w.stepID]
			}
		case <-deadline:
			t.Fatalf("timeout waiting for step %s", w.stepID)
			return nil
		case <-ctx.Done():
			t.FailNow()
			return nil
		}
	}
}

// Convenience methods that subscribe and wait in one call. Use these when you
// call them BEFORE the action that triggers events

func (env *TestEngineEnv) WaitForFlowStatus(
	t *testing.T, ctx context.Context, flowID api.FlowID, timeout time.Duration,
) *api.FlowState {
	t.Helper()
	return env.SubscribeToFlowStatus(flowID).Wait(t, ctx, timeout)
}

func (env *TestEngineEnv) WaitForStepStarted(
	t *testing.T, ctx context.Context, flowID api.FlowID, stepID api.StepID,
	timeout time.Duration,
) *api.ExecutionState {
	t.Helper()

	// Check if step has already started (avoid race condition)
	flow, err := env.Engine.GetFlowState(ctx, flowID)
	if err == nil && flow != nil {
		exec, ok := flow.Executions[stepID]
		if ok && exec.Status != api.StepPending {
			return exec
		}
	}

	return env.SubscribeToStepStarted(flowID, stepID).Wait(t, ctx, timeout)
}

func (env *TestEngineEnv) WaitForStepStatus(
	t *testing.T, ctx context.Context, flowID api.FlowID, stepID api.StepID,
	timeout time.Duration,
) *api.ExecutionState {
	t.Helper()
	return env.SubscribeToStepStatus(flowID, stepID).Wait(t, ctx, timeout)
}

// Filter helpers

func filterFlowEvents(
	flowID api.FlowID, eventTypes ...api.EventType,
) events.EventFilter {
	typeFilter := events.FilterEvents(toTimeboxTypes(eventTypes)...)
	flowFilter := events.FilterFlow(flowID)
	return func(ev *timebox.Event) bool {
		return typeFilter(ev) && flowFilter(ev)
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
		var e api.StepStartedEvent // reuse existing type for ID extraction
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
