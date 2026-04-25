package helpers

import (
	"testing"
	"time"

	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const waitPollInterval = 10 * time.Millisecond

// WaitFor runs fn and waits for a matching event
func (e *TestEngineEnv) WaitFor(filter wait.EventFilter, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *event.Consumer) {
		w := wait.On(e.T, consumer)
		fn()
		w.ForEvent(filter)
	})
}

// WaitForCount runs fn and waits for count events
func (e *TestEngineEnv) WaitForCount(
	count int, filter wait.EventFilter, fn func(),
) {
	e.T.Helper()
	e.WithConsumer(func(consumer *event.Consumer) {
		w := wait.On(e.T, consumer)
		fn()
		w.ForEvents(count, filter)
	})
}

// WithConsumer provides a scoped event consumer for tests
func (e *TestEngineEnv) WithConsumer(fn func(*event.Consumer)) {
	e.T.Helper()
	consumer := e.EventHub.NewConsumer()
	defer consumer.Close()
	fn(consumer)
}

// WaitAfterAll creates multiple consumers, runs fn, and waits with each
func (e *TestEngineEnv) WaitAfterAll(count int, fn func([]*wait.Wait)) {
	e.T.Helper()

	consumers := make([]*event.Consumer, count)
	waits := make([]*wait.Wait, count)
	for i := range count {
		consumer := e.EventHub.NewConsumer()
		consumers[i] = consumer
		waits[i] = wait.On(e.T, consumer)
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
) api.FlowState {
	e.T.Helper()
	fn()
	return e.WaitForTerminalFlow(flowID)
}

func (e *TestEngineEnv) WaitForTerminalFlow(flowID api.FlowID) api.FlowState {
	e.T.Helper()
	return WaitForTerminalFlowState(e.T, e.Engine, flowID)
}

// WaitForTerminalFlowState waits for a flow to reach a terminal state.
func WaitForTerminalFlowState(
	t *testing.T, eng *engine.Engine, flowID api.FlowID,
) api.FlowState {
	t.Helper()
	return WaitForFlowState(t, eng, flowID, wait.DefaultTimeout, func(st api.FlowState) bool {
		return isFlowTerminal(st)
	})
}

// WaitForTerminalFlows waits for all flows to reach terminal states.
func WaitForTerminalFlows(
	t *testing.T, eng *engine.Engine, flowIDs []api.FlowID,
	timeout time.Duration,
) map[api.FlowID]api.FlowState {
	t.Helper()

	res := make(map[api.FlowID]api.FlowState, len(flowIDs))
	deadline := time.Now().Add(timeout)
	for {
		allTerminal := true
		var lastErr error
		for _, fid := range flowIDs {
			st, err := eng.GetFlowState(fid)
			if err != nil {
				lastErr = err
				allTerminal = false
				break
			}
			res[fid] = st
			if !isFlowTerminal(st) {
				allTerminal = false
			}
		}
		if allTerminal {
			return res
		}
		if time.Now().After(deadline) {
			if lastErr != nil {
				t.Fatalf("failed to fetch terminal flows: %v", lastErr)
			}
			t.Fatalf("flows did not reach terminal state: %v", flowIDs)
		}
		time.Sleep(waitPollInterval)
	}
}

// WaitForFlowExists waits for a flow state to become readable.
func WaitForFlowExists(
	t *testing.T, eng *engine.Engine, flowID api.FlowID,
) api.FlowState {
	t.Helper()
	return WaitForFlowState(t, eng, flowID, wait.DefaultTimeout, nil)
}

// WaitForFlowState waits for a flow state accepted by accept.
func WaitForFlowState(
	t *testing.T, eng *engine.Engine, flowID api.FlowID,
	timeout time.Duration, accept func(api.FlowState) bool,
) api.FlowState {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		st, err := eng.GetFlowState(flowID)
		if err == nil && (accept == nil || accept(st)) {
			return st
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("failed to fetch flow %s: %v", flowID, err)
			}
			t.Fatalf("flow %s did not reach expected state", flowID)
		}
		time.Sleep(waitPollInterval)
	}
}

// WaitForStepStarted waits for a step to start and returns the execution
func (e *TestEngineEnv) WaitForStepStarted(
	fs api.FlowStep, fn func(),
) api.ExecutionState {
	e.T.Helper()
	fn()
	return e.waitForStartedStep(fs.FlowID, fs.StepID)
}

// WaitForStepStatus waits for a step to finish and returns the execution
func (e *TestEngineEnv) WaitForStepStatus(
	flowID api.FlowID, stepID api.StepID, fn func(),
) api.ExecutionState {
	e.T.Helper()
	fn()
	return e.waitForTerminalStep(flowID, stepID)
}

func (e *TestEngineEnv) getExecutionState(
	flowID api.FlowID, stepID api.StepID,
) (api.ExecutionState, error) {
	fl, err := e.Engine.GetFlowState(flowID)
	if err != nil {
		return api.ExecutionState{}, err
	}
	return fl.Executions[stepID], nil
}

func isFlowTerminal(state api.FlowState) bool {
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

func (e *TestEngineEnv) waitForTerminalStep(
	flowID api.FlowID, stepID api.StepID,
) api.ExecutionState {
	e.T.Helper()

	deadline := time.Now().Add(wait.DefaultTimeout)
	for {
		ex, err := e.getExecutionState(flowID, stepID)
		if err == nil && isStepTerminal(ex.Status) {
			return ex
		}
		if time.Now().After(deadline) {
			if err != nil {
				e.T.Fatalf("failed to fetch execution %s: %v", stepID, err)
			}
			e.T.Fatalf("execution %s not terminal after event", stepID)
		}
		time.Sleep(waitPollInterval)
	}
}

func (e *TestEngineEnv) waitForStartedStep(
	flowID api.FlowID, stepID api.StepID,
) api.ExecutionState {
	e.T.Helper()

	deadline := time.Now().Add(wait.DefaultTimeout)
	for {
		ex, err := e.getExecutionState(flowID, stepID)
		if err == nil && isStepStarted(ex.Status) {
			return ex
		}
		if time.Now().After(deadline) {
			if err != nil {
				e.T.Fatalf("failed to fetch execution %s: %v", stepID, err)
			}
			e.T.Fatalf("execution %s not started", stepID)
		}
		time.Sleep(waitPollInterval)
	}
}
