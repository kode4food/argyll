package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

var workflowTransitions = map[api.WorkflowStatus]map[api.WorkflowStatus]bool{
	api.WorkflowActive: {
		api.WorkflowCompleted: true,
		api.WorkflowFailed:    true,
	},
	api.WorkflowCompleted: {},
	api.WorkflowFailed:    {},
}

func (e *Engine) GetWorkflowState(
	ctx context.Context, flowID timebox.ID,
) (*api.WorkflowState, error) {
	state, err := e.workflowExec.Exec(ctx, workflowKey(flowID),
		func(st *api.WorkflowState, ag *WorkflowAggregator) error {
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	if state.ID == "" {
		return nil, ErrWorkflowNotFound
	}

	return state, nil
}

func (e *Engine) CompleteWorkflow(
	ctx context.Context, flowID timebox.ID, result api.Args,
) error {
	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		ev, err := json.Marshal(api.WorkflowCompletedEvent{
			FlowID: flowID,
			Result: result,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkflowCompleted, ev)
		return nil
	}

	_, err := e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
}

func (e *Engine) FailWorkflow(
	ctx context.Context, flowID timebox.ID, errMsg string,
) error {
	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		ev, err := json.Marshal(api.WorkflowFailedEvent{
			FlowID: flowID,
			Error:  errMsg,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkflowFailed, ev)
		return nil
	}

	_, err := e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
}

func (e *Engine) StartWork(
	ctx context.Context, flowID, stepID timebox.ID, token api.Token,
	inputs api.Args,
) error {
	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		ev, err := json.Marshal(api.WorkStartedEvent{
			FlowID: flowID,
			StepID: stepID,
			Token:  token,
			Inputs: inputs,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkStarted, ev)
		return nil
	}

	_, err := e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
}

func (e *Engine) CompleteWork(
	ctx context.Context, flowID, stepID timebox.ID, token api.Token,
	outputs api.Args,
) error {
	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		ev, err := json.Marshal(api.WorkCompletedEvent{
			FlowID:  flowID,
			StepID:  stepID,
			Token:   token,
			Outputs: outputs,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkCompleted, ev)
		return nil
	}

	_, err := e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
}

func (e *Engine) FailWork(
	ctx context.Context, flowID, stepID timebox.ID, token api.Token,
	errMsg string,
) error {
	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		ev, err := json.Marshal(api.WorkFailedEvent{
			FlowID: flowID,
			StepID: stepID,
			Token:  token,
			Error:  errMsg,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkFailed, ev)
		return nil
	}

	_, err := e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
}

func (e *Engine) SetAttribute(
	ctx context.Context, flowID, stepID timebox.ID, attr api.Name, value any,
) error {
	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		if _, ok := st.Attributes[attr]; ok {
			return fmt.Errorf("%w: %s", ErrAttributeAlreadySet, attr)
		}

		ev, err := json.Marshal(api.AttributeSetEvent{
			FlowID: flowID,
			StepID: stepID,
			Key:    attr,
			Value:  value,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeAttributeSet, ev)
		return nil
	}

	_, err := e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
}

func (e *Engine) GetAttribute(
	ctx context.Context, flowID timebox.ID, attr api.Name,
) (any, bool, error) {
	flow, err := e.GetWorkflowState(ctx, flowID)
	if err != nil {
		return nil, false, err
	}

	if av, ok := flow.Attributes[attr]; ok {
		return av.Value, true, nil
	}
	return nil, false, nil
}

func (e *Engine) GetAttributes(
	ctx context.Context, flowID timebox.ID,
) (api.Args, error) {
	flow, err := e.GetWorkflowState(ctx, flowID)
	if err != nil {
		return nil, err
	}

	return flow.GetAttributeArgs(), nil
}

func (e *Engine) GetWorkflowEvents(
	ctx context.Context, flowID timebox.ID, fromSeq int64,
) ([]*timebox.Event, error) {
	id := timebox.NewAggregateID("workflow", flowID)
	return e.workflowExec.GetStore().GetEvents(ctx, id, fromSeq)
}

func (e *Engine) ListWorkflows(
	ctx context.Context,
) ([]*api.WorkflowDigest, error) {
	ids, err := e.workflowExec.GetStore().ListAggregates(
		ctx, timebox.NewAggregateID("workflow", "*"),
	)
	if err != nil {
		return nil, err
	}

	var digests []*api.WorkflowDigest
	for _, id := range ids {
		if digest := e.buildWorkflowDigest(ctx, id); digest != nil {
			digests = append(digests, digest)
		}
	}

	return digests, nil
}

func (e *Engine) buildWorkflowDigest(
	ctx context.Context, id timebox.AggregateID,
) *api.WorkflowDigest {
	if len(id) < 2 || id[0] != "workflow" {
		return nil
	}

	flowID := id[1]
	flow, err := e.GetWorkflowState(ctx, flowID)
	if err != nil {
		return nil
	}

	return &api.WorkflowDigest{
		ID:          flow.ID,
		Status:      flow.Status,
		CreatedAt:   flow.CreatedAt,
		CompletedAt: flow.CompletedAt,
		Error:       flow.Error,
	}
}

func (e *Engine) areOutputsNeeded(
	stepID timebox.ID, flow *api.WorkflowState,
) bool {
	step := flow.Plan.GetStep(stepID)
	if step == nil {
		return false
	}

	for _, goalID := range flow.Plan.Goals {
		if goalID == stepID {
			return true
		}
	}

	for name, attr := range step.Attributes {
		if attr.Role != api.RoleOutput {
			continue
		}

		if _, alreadySatisfied := flow.Attributes[name]; alreadySatisfied {
			continue
		}

		attrDeps, ok := flow.Plan.Attributes[name]
		if !ok || len(attrDeps.Consumers) == 0 {
			continue
		}

		for _, consumerID := range attrDeps.Consumers {
			consumerExec, ok := flow.Executions[consumerID]
			if !ok {
				continue
			}

			if consumerExec.Status == api.StepPending {
				return true
			}
		}
	}

	return false
}

func (e *Engine) isWorkflowComplete(flow *api.WorkflowState) bool {
	for stepID := range flow.Plan.Steps {
		if !e.isStepComplete(stepID, flow) {
			return false
		}
	}
	return true
}

func (e *Engine) evaluateWorkflowState(
	ctx context.Context, flowID timebox.ID, flow *api.WorkflowState,
) {
	for stepID := range flow.Plan.Steps {
		exec, ok := flow.Executions[stepID]
		if !ok || exec.Status != api.StepPending {
			continue
		}
		e.maybeSkipStep(ctx, flowID, stepID, flow)
	}
}

func (e *Engine) maybeSkipStep(
	ctx context.Context, flowID, stepID timebox.ID, flow *api.WorkflowState,
) {
	if e.canStepComplete(stepID, flow) {
		if !e.areOutputsNeeded(stepID, flow) {
			err := e.SkipStepExecution(ctx, flowID, stepID, "outputs not needed")
			if err != nil {
				slog.Error("Failed to skip step",
					slog.Any("step_id", stepID),
					slog.Any("error", err))
			}
		}
		return
	}

	err := e.FailStepExecution(ctx, flowID, stepID, "required inputs cannot be satisfied")
	if err != nil {
		slog.Error("Failed to fail step",
			slog.Any("step_id", stepID),
			slog.Any("error", err))
	}
}

func (e *Engine) IsWorkflowFailed(flow *api.WorkflowState) bool {
	for _, goalID := range flow.Plan.Goals {
		if !e.canStepComplete(goalID, flow) {
			return true
		}
	}
	return false
}

func (e *Engine) failWorkflow(
	ctx context.Context, flowID timebox.ID, flow *api.WorkflowState,
) {
	var failed []string

	for stepID := range flow.Plan.Steps {
		if exec, ok := flow.Executions[stepID]; ok {
			failed = e.appendFailedStep(failed, stepID, exec)
		}
	}

	errMsg := fmt.Sprintf(
		"goal unreachable: failed steps: %v",
		failed,
	)

	if err := e.FailWorkflow(ctx, flowID, errMsg); err != nil {
		slog.Error("Failed to record failure",
			slog.Any("flow_id", flowID),
			slog.Any("error", err))
	}
}

func (e *Engine) completeWorkflow(
	ctx context.Context, flowID timebox.ID, flow *api.WorkflowState,
) {
	result := api.Args{}

	for _, goalID := range flow.Plan.Goals {
		if goal := flow.Executions[goalID]; goal != nil {
			maps.Copy(result, goal.Outputs)
		}
	}

	if err := e.CompleteWorkflow(ctx, flowID, result); err != nil {
		slog.Error("Failed to complete workflow",
			slog.Any("flow_id", flowID),
			slog.Any("error", err))
	}
}

func (e *Engine) HasInputProvider(name api.Name, flow *api.WorkflowState) bool {
	deps := flow.Plan.Attributes[name]
	if deps == nil {
		return false
	}

	if len(deps.Providers) == 0 {
		return true
	}

	for _, providerID := range deps.Providers {
		if e.canStepComplete(providerID, flow) {
			return true
		}
	}
	return false
}

func workflowKey(flowID timebox.ID) timebox.AggregateID {
	return timebox.NewAggregateID("workflow", flowID)
}

func (e *Engine) GetActiveWorkflow(
	flowID timebox.ID,
) (*api.WorkflowState, bool) {
	flow, err := e.GetWorkflowState(e.ctx, flowID)
	if err != nil {
		slog.Error("Failed to get workflow state",
			slog.Any("flow_id", flowID),
			slog.Any("error", err))
		return nil, false
	}

	if isWorkflowTerminal(flow.Status) {
		return nil, false
	}

	return flow, true
}

func (e *Engine) ensureScriptsCompiled(
	flowID timebox.ID, flow *api.WorkflowState,
) bool {
	if !flow.Plan.NeedsCompilation() {
		return true
	}

	if err := e.scripts.CompilePlan(flow.Plan); err != nil {
		slog.Error("Failed to compile scripts",
			slog.Any("flow_id", flowID),
			slog.Any("error", err))
		return false
	}

	return true
}

func isWorkflowTerminal(status api.WorkflowStatus) bool {
	transitions, ok := workflowTransitions[status]
	return ok && len(transitions) == 0
}
