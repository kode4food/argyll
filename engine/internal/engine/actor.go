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

func (wa *flowActor) run() {
	defer wa.wg.Done()
	defer wa.flows.Delete(wa.flowID)

	idleTimer := time.NewTimer(100 * time.Millisecond)
	defer idleTimer.Stop()

	for {
		select {
		case event := <-wa.events:
			wa.handleEvent(event)
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(100 * time.Millisecond)

		case <-idleTimer.C:
			select {
			case event := <-wa.events:
				wa.handleEvent(event)
				idleTimer.Reset(100 * time.Millisecond)
			default:
				return
			}

		case <-wa.ctx.Done():
			return
		}
	}
}

func (wa *flowActor) createEventHandler() timebox.Handler {
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
		flowStarted:      wa.handleProcessFlow,
		stepCompleted:    wa.handleProcessFlow,
		stepFailed:       wa.handleProcessFlow,
		workSucceeded:    timebox.MakeHandler(wa.handleWorkSucceeded),
		workFailed:       timebox.MakeHandler(wa.handleWorkFailed),
		workNotCompleted: timebox.MakeHandler(wa.handleWorkNotCompleted),
		retryScheduled:   timebox.MakeHandler(wa.handleRetryScheduled),
	})
}

func (wa *flowActor) handleEvent(event *timebox.Event) {
	if err := wa.eventHandler(event); err != nil {
		slog.Error("Failed to handle flow event",
			slog.Any("flow_id", wa.flowID),
			slog.Any("event_type", event.Type),
			slog.Any("error", err))
	}
}

func (wa *flowActor) handleProcessFlow(_ *timebox.Event) error {
	wa.processFlow()
	return nil
}

func (wa *flowActor) handleWorkSucceeded(
	_ *timebox.Event, data api.WorkSucceededEvent,
) error {
	wa.checkIfStepDone(data.StepID)
	return nil
}

func (wa *flowActor) handleWorkFailed(
	_ *timebox.Event, data api.WorkFailedEvent,
) error {
	wa.checkIfStepDone(data.StepID)
	return nil
}

func (wa *flowActor) handleWorkNotCompleted(
	_ *timebox.Event, data api.WorkNotCompletedEvent,
) error {
	flow, err := wa.GetFlowState(wa.ctx, wa.flowID)
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

	fs := FlowStep{FlowID: wa.flowID, StepID: data.StepID}
	if wa.ShouldRetry(step, workItem) {
		_ = wa.ScheduleRetry(wa.ctx, fs, data.Token, data.Error)
	} else {
		_ = wa.FailWork(wa.ctx, fs, data.Token, data.Error)
	}

	return nil
}

func (wa *flowActor) checkIfStepDone(stepID api.StepID) {
	flow, err := wa.GetFlowState(wa.ctx, wa.flowID)
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
		wa.checkCompletableSteps(wa.ctx, wa.flowID, flow)
	}
}

func (wa *flowActor) handleRetryScheduled(
	_ *timebox.Event, data api.RetryScheduledEvent,
) error {
	flow, err := wa.GetFlowState(wa.ctx, wa.flowID)
	if err != nil {
		return nil
	}

	exec, ok := flow.Executions[data.StepID]
	if !ok || exec.Status != api.StepActive {
		return nil
	}

	wa.checkCompletableSteps(wa.ctx, wa.flowID, flow)
	return nil
}

func (wa *flowActor) processFlow() {
	flow, ok := wa.GetActiveFlow(wa.flowID)
	if !ok {
		return
	}

	if !wa.ensureScriptsCompiled(wa.flowID, flow) {
		return
	}

	wa.evaluateFlowState(wa.ctx, wa.flowID, flow)
	wa.checkCompletableSteps(wa.ctx, wa.flowID, flow)

	flow, ok = wa.GetActiveFlow(wa.flowID)
	if !ok {
		return
	}

	ready := wa.findReadySteps(flow)
	if len(ready) == 0 {
		wa.handleTerminalState(flow)
		return
	}

	wa.launchReadySteps(ready)
}

func (wa *flowActor) handleTerminalState(flow *api.FlowState) {
	if wa.isFlowComplete(flow) {
		wa.completeFlow(wa.ctx, wa.flowID, flow)
		return
	}

	if wa.IsFlowFailed(flow) {
		wa.evaluateFlowState(wa.ctx, wa.flowID, flow)
		wa.failFlow(wa.ctx, wa.flowID, flow)
	}
}

func (wa *flowActor) launchReadySteps(ready []api.StepID) {
	for _, stepID := range ready {
		wa.wg.Add(1)
		go func(stepID api.StepID) {
			defer wa.wg.Done()
			wa.executeStep(wa.ctx, FlowStep{
				FlowID: wa.flowID,
				StepID: stepID,
			})
		}(stepID)
	}
}

func (wa *flowActor) findReadySteps(flow *api.FlowState) []api.StepID {
	visited := util.Set[api.StepID]{}
	var ready []api.StepID

	for _, goalID := range flow.Plan.Goals {
		wa.findReadyStepsFromGoal(goalID, flow, visited, &ready)
	}

	return ready
}

func (wa *flowActor) findReadyStepsFromGoal(
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
			wa.findReadyStepsFromGoal(providerID, flow, visited, ready)
		}
	}

	if wa.isStepReadyForExec(stepID, flow) {
		*ready = append(*ready, stepID)
	}
}

func (wa *flowActor) isStepReadyForExec(
	stepID api.StepID, flow *api.FlowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return false
	}
	if !wa.isStepReady(stepID, flow) {
		return false
	}
	return wa.areOutputsNeeded(stepID, flow)
}

func (wa *flowActor) isStepReady(stepID api.StepID, flow *api.FlowState) bool {
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
