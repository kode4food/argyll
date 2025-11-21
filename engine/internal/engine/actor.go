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
	flowID       timebox.ID
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
	workCompleted := timebox.MakeHandler(wa.handleWorkCompleted)
	workFailed := timebox.MakeHandler(wa.handleWorkFailed)

	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		api.EventTypeFlowStarted:   wa.handleProcessFlow,
		api.EventTypeAttributeSet:  wa.handleProcessFlow,
		api.EventTypeStepCompleted: wa.handleProcessFlow,
		api.EventTypeStepFailed:    wa.handleProcessFlow,
		api.EventTypeWorkCompleted: workCompleted,
		api.EventTypeWorkFailed:    workFailed,
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

func (wa *flowActor) handleWorkCompleted(
	_ *timebox.Event, data api.WorkCompletedEvent,
) error {
	flow, err := wa.GetFlowState(wa.ctx, wa.flowID)
	if err != nil {
		return nil
	}

	exec, ok := flow.Executions[data.StepID]
	if !ok || exec.Status != api.StepActive {
		return nil
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
	return nil
}

func (wa *flowActor) handleWorkFailed(
	_ *timebox.Event, data api.WorkFailedEvent,
) error {
	return wa.handleWorkCompleted(nil, api.WorkCompletedEvent{
		FlowID: data.FlowID,
		StepID: data.StepID,
	})
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

func (wa *flowActor) launchReadySteps(ready []timebox.ID) {
	for _, stepID := range ready {
		wa.wg.Add(1)
		go func(stepID timebox.ID) {
			defer wa.wg.Done()
			wa.executeStep(wa.ctx, FlowStep{
				FlowID: wa.flowID,
				StepID: stepID,
			})
		}(stepID)
	}
}

func (wa *flowActor) findReadySteps(flow *api.FlowState) []timebox.ID {
	visited := util.Set[timebox.ID]{}
	var ready []timebox.ID

	for _, goalID := range flow.Plan.Goals {
		wa.findReadyStepsFromGoal(goalID, flow, visited, &ready)
	}

	return ready
}

func (wa *flowActor) findReadyStepsFromGoal(
	stepID timebox.ID, flow *api.FlowState, visited util.Set[timebox.ID],
	ready *[]timebox.ID,
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
	stepID timebox.ID, flow *api.FlowState,
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

func (wa *flowActor) isStepReady(
	stepID timebox.ID, flow *api.FlowState,
) bool {
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
