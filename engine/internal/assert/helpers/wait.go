package helpers

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const DefaultWaitTimeout = wait.DefaultTimeout

// WaitFor runs fn and waits for a matching event
func (e *TestEngineEnv) WaitFor(filter wait.EventFilter, fn func()) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		w := wait.On(e.T, consumer)
		w.ForEvent(filter)
	})
}

// WaitForCount runs fn and waits for count events
func (e *TestEngineEnv) WaitForCount(
	count int, filter wait.EventFilter, fn func(),
) {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		w := wait.On(e.T, consumer)
		w.ForEvents(count, filter)
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
func (e *TestEngineEnv) WaitAfterAll(count int, fn func([]*wait.Wait)) {
	e.T.Helper()

	consumers := make([]*timebox.Consumer, count)
	waits := make([]*wait.Wait, count)
	for i := 0; i < count; i++ {
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
) *api.FlowState {
	e.T.Helper()
	e.WithConsumer(func(consumer *timebox.Consumer) {
		fn()
		w := wait.On(e.T, consumer)
		state, err := e.Engine.GetFlowState(flowID)
		if err == nil && isFlowTerminal(state) {
			return
		}
		w.ForEvent(wait.FlowTerminal(flowID))
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
		w := wait.On(e.T, consumer)
		exec, err := e.getExecutionState(fs.FlowID, fs.StepID)
		if err == nil && exec != nil && isStepStarted(exec.Status) {
			return
		}
		w.ForEvent(wait.StepStarted(fs))
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
		w := wait.On(e.T, consumer)
		exec, err := e.getExecutionState(flowID, stepID)
		if err == nil && exec != nil && isStepTerminal(exec.Status) {
			return
		}
		w.ForEvent(wait.StepTerminal(
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
