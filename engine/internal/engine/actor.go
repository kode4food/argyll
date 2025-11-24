package engine

import (
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

type flowActor struct {
	*Engine
	flowID       api.FlowID
	events       chan *timebox.Event
	eventHandler timebox.Handler
}

func (a *flowActor) run() {
	defer a.wg.Done()
	defer a.flows.Delete(a.flowID)

	idleTimer := time.NewTimer(100 * time.Millisecond)
	defer idleTimer.Stop()

	for {
		select {
		case event := <-a.events:
			a.handleEvent(event)
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(100 * time.Millisecond)

		case <-idleTimer.C:
			select {
			case event := <-a.events:
				a.handleEvent(event)
				idleTimer.Reset(100 * time.Millisecond)
			default:
				return
			}

		case <-a.ctx.Done():
			return
		}
	}
}

func (a *flowActor) createEventHandler() timebox.Handler {
	const (
		flowStarted      = timebox.EventType(api.EventTypeFlowStarted)
		stepCompleted    = timebox.EventType(api.EventTypeStepCompleted)
		stepFailed       = timebox.EventType(api.EventTypeStepFailed)
		workSucceeded    = timebox.EventType(api.EventTypeWorkSucceeded)
		workFailed       = timebox.EventType(api.EventTypeWorkFailed)
		workNotCompleted = timebox.EventType(api.EventTypeWorkNotCompleted)
		retryScheduled   = timebox.EventType(api.EventTypeRetryScheduled)
	)

	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		flowStarted:      a.handleProcessFlow,
		stepCompleted:    a.handleProcessFlow,
		stepFailed:       a.handleProcessFlow,
		workSucceeded:    timebox.MakeHandler(a.handleWorkSucceeded),
		workFailed:       timebox.MakeHandler(a.handleWorkFailed),
		workNotCompleted: timebox.MakeHandler(a.handleWorkNotCompleted),
		retryScheduled:   timebox.MakeHandler(a.handleRetryScheduled),
	})
}

func (a *flowActor) handleEvent(event *timebox.Event) {
	if err := a.eventHandler(event); err != nil {
		slog.Error("Failed to handle flow event",
			slog.Any("flow_id", a.flowID),
			slog.Any("event_type", event.Type),
			slog.Any("error", err))
	}
}

func (a *flowActor) handleProcessFlow(_ *timebox.Event) error {
	a.processFlow()
	return nil
}

func (a *flowActor) handleWorkSucceeded(
	_ *timebox.Event, data api.WorkSucceededEvent,
) error {
	a.checkIfStepDone(data.StepID)
	return nil
}

func (a *flowActor) handleWorkFailed(
	_ *timebox.Event, data api.WorkFailedEvent,
) error {
	a.checkIfStepDone(data.StepID)
	return nil
}

func (a *flowActor) handleWorkNotCompleted(
	_ *timebox.Event, data api.WorkNotCompletedEvent,
) error {
	flow, err := a.GetFlowState(a.ctx, a.flowID)
	if err != nil {
		return nil
	}

	exec, ok := flow.Executions[data.StepID]
	if !ok {
		return nil
	}

	workItem := exec.WorkItems[data.Token]
	if workItem == nil {
		return nil
	}

	step := flow.Plan.GetStep(data.StepID)
	if step == nil {
		return nil
	}

	fs := FlowStep{FlowID: a.flowID, StepID: data.StepID}
	if a.ShouldRetry(step, workItem) {
		_ = a.ScheduleRetry(a.ctx, fs, data.Token, data.Error)
	} else {
		_ = a.FailWork(a.ctx, fs, data.Token, data.Error)
	}

	return nil
}

func (a *flowActor) checkIfStepDone(stepID api.StepID) {
	flow, err := a.GetFlowState(a.ctx, a.flowID)
	if err != nil {
		return
	}

	exec, ok := flow.Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return
	}

	allDone := true
	for _, item := range exec.WorkItems {
		if !workTransitions.IsTerminal(item.Status) {
			allDone = false
			break
		}
	}

	if allDone {
		a.checkCompletableSteps(a.ctx, a.flowID, flow)
	}
}

func (a *flowActor) handleRetryScheduled(
	_ *timebox.Event, data api.RetryScheduledEvent,
) error {
	flow, err := a.GetFlowState(a.ctx, a.flowID)
	if err != nil {
		return nil
	}

	exec, ok := flow.Executions[data.StepID]
	if !ok || exec.Status != api.StepActive {
		return nil
	}

	a.checkCompletableSteps(a.ctx, a.flowID, flow)
	return nil
}

func (a *flowActor) processFlow() {
	flow, ok := a.GetActiveFlow(a.flowID)
	if !ok {
		return
	}

	if !a.ensureScriptsCompiled(a.flowID, flow) {
		return
	}

	a.evaluateFlowState(a.ctx, a.flowID, flow)
	a.checkCompletableSteps(a.ctx, a.flowID, flow)

	flow, ok = a.GetActiveFlow(a.flowID)
	if !ok {
		return
	}

	ready := a.findReadySteps(flow)
	if len(ready) == 0 {
		a.handleTerminalState(flow)
		return
	}

	for _, stepID := range ready {
		fs := FlowStep{FlowID: a.flowID, StepID: stepID}
		execCtx := a.PrepareStepExecution(a.ctx, fs)
		if execCtx == nil {
			continue
		}

		execCtx.execute(a.ctx)
	}
}

func (a *flowActor) handleTerminalState(flow *api.FlowState) {
	if a.isFlowComplete(flow) {
		a.completeFlow(a.ctx, a.flowID, flow)
		return
	}

	if a.IsFlowFailed(flow) {
		a.evaluateFlowState(a.ctx, a.flowID, flow)
		a.failFlow(a.ctx, a.flowID, flow)
	}
}

func (a *flowActor) findReadySteps(flow *api.FlowState) []api.StepID {
	visited := util.Set[api.StepID]{}
	var ready []api.StepID

	for _, goalID := range flow.Plan.Goals {
		a.findReadyStepsFromGoal(goalID, flow, visited, &ready)
	}

	return ready
}

func (a *flowActor) findReadyStepsFromGoal(
	stepID api.StepID, flow *api.FlowState, visited util.Set[api.StepID],
	ready *[]api.StepID,
) {
	if visited.Contains(stepID) {
		return
	}
	visited.Add(stepID)

	exec, ok := flow.Executions[stepID]
	if !ok || exec.Status != api.StepPending {
		return
	}

	step := flow.Plan.GetStep(stepID)
	if step == nil {
		return
	}

	for name, attr := range step.Attributes {
		if !attr.IsRequired() {
			continue
		}

		if _, hasAttr := flow.Attributes[name]; hasAttr {
			continue
		}

		deps := flow.Plan.Attributes[name]
		if deps == nil || len(deps.Providers) == 0 {
			continue
		}

		for _, providerID := range deps.Providers {
			a.findReadyStepsFromGoal(providerID, flow, visited, ready)
		}
	}

	if a.isStepReadyForExec(stepID, flow) {
		*ready = append(*ready, stepID)
	}
}

func (a *flowActor) isStepReadyForExec(
	stepID api.StepID, flow *api.FlowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return false
	}
	if !a.isStepReady(stepID, flow) {
		return false
	}
	return a.areOutputsNeeded(stepID, flow)
}

func (a *flowActor) isStepReady(stepID api.StepID, flow *api.FlowState) bool {
	step := flow.Plan.GetStep(stepID)
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			if _, ok := flow.Attributes[name]; !ok {
				return false
			}
		}
	}
	return true
}
