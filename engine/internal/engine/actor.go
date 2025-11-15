package engine

import (
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type workflowActor struct {
	*Engine
	flowID       timebox.ID
	events       chan *timebox.Event
	eventHandler timebox.Handler
}

func (wa *workflowActor) run() {
	defer wa.wg.Done()
	defer wa.workflows.Delete(wa.flowID)

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

func (wa *workflowActor) createEventHandler() timebox.Handler {
	workCompleted := timebox.MakeHandler(wa.handleWorkCompleted)
	workFailed := timebox.MakeHandler(wa.handleWorkFailed)

	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		api.EventTypeWorkflowStarted: wa.handleProcessWorkflow,
		api.EventTypeAttributeSet:    wa.handleProcessWorkflow,
		api.EventTypeStepCompleted:   wa.handleProcessWorkflow,
		api.EventTypeStepFailed:      wa.handleProcessWorkflow,
		api.EventTypeWorkCompleted:   workCompleted,
		api.EventTypeWorkFailed:      workFailed,
	})
}

func (wa *workflowActor) handleEvent(event *timebox.Event) {
	if err := wa.eventHandler(event); err != nil {
		slog.Error("Failed to handle workflow event",
			slog.Any("flow_id", wa.flowID),
			slog.Any("event_type", event.Type),
			slog.Any("error", err))
	}
}

func (wa *workflowActor) handleProcessWorkflow(_ *timebox.Event) error {
	wa.processWorkflow()
	return nil
}

func (wa *workflowActor) handleWorkCompleted(
	_ *timebox.Event, data api.WorkCompletedEvent,
) error {
	flow, err := wa.GetWorkflowState(wa.ctx, wa.flowID)
	if err != nil {
		return nil
	}

	exec, ok := flow.Executions[data.StepID]
	if !ok || exec.Status != api.StepActive {
		return nil
	}

	allDone := true
	for _, item := range exec.WorkItems {
		if !isWorkTerminal(item.Status) {
			allDone = false
			break
		}
	}

	if allDone {
		wa.checkCompletableSteps(wa.ctx, wa.flowID, flow)
	}
	return nil
}

func (wa *workflowActor) handleWorkFailed(
	_ *timebox.Event, data api.WorkFailedEvent,
) error {
	return wa.handleWorkCompleted(nil, api.WorkCompletedEvent{
		FlowID: data.FlowID,
		StepID: data.StepID,
	})
}

func (wa *workflowActor) processWorkflow() {
	flow, ok := wa.GetActiveWorkflow(wa.flowID)
	if !ok {
		return
	}

	if !wa.ensureScriptsCompiled(wa.flowID, flow) {
		return
	}

	wa.evaluateWorkflowState(wa.ctx, wa.flowID, flow)
	wa.checkCompletableSteps(wa.ctx, wa.flowID, flow)

	flow, ok = wa.GetActiveWorkflow(wa.flowID)
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

func (wa *workflowActor) handleTerminalState(flow *api.WorkflowState) {
	if wa.isWorkflowComplete(flow) {
		wa.completeWorkflow(wa.ctx, wa.flowID, flow)
		return
	}

	if wa.IsWorkflowFailed(flow) {
		wa.evaluateWorkflowState(wa.ctx, wa.flowID, flow)
		wa.failWorkflow(wa.ctx, wa.flowID, flow)
	}
}

func (wa *workflowActor) launchReadySteps(ready []timebox.ID) {
	for _, stepID := range ready {
		wa.wg.Add(1)
		go func(stepID timebox.ID) {
			defer wa.wg.Done()
			wa.executeStep(wa.ctx, wa.flowID, stepID)
		}(stepID)
	}
}

func (wa *workflowActor) findReadySteps(flow *api.WorkflowState) []timebox.ID {
	visited := make(map[timebox.ID]bool)
	var ready []timebox.ID

	for _, goalID := range flow.Plan.Goals {
		wa.findReadyStepsFromGoal(goalID, flow, visited, &ready)
	}

	return ready
}

func (wa *workflowActor) findReadyStepsFromGoal(
	stepID timebox.ID, flow *api.WorkflowState, visited map[timebox.ID]bool,
	ready *[]timebox.ID,
) {
	if visited[stepID] {
		return
	}
	visited[stepID] = true

	exec, ok := flow.Executions[stepID]
	if !ok || exec.Status != api.StepPending {
		return
	}

	step := flow.Plan.GetStep(stepID)
	if step == nil {
		return
	}

	for name, attr := range step.Attributes {
		if attr.Role != api.RoleRequired {
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

func (wa *workflowActor) isStepReadyForExec(
	stepID timebox.ID, flow *api.WorkflowState,
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

func (wa *workflowActor) isStepReady(
	stepID timebox.ID, flow *api.WorkflowState,
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
