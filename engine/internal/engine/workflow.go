package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

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
			WorkflowID: flowID,
			Result:     result,
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
			WorkflowID: flowID,
			Error:      errMsg,
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

func (e *Engine) StartStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, inputs api.Args,
) error {
	return e.transitionStepExecution(
		ctx, flowID, stepID, api.StepActive, "start",
		api.EventTypeStepStarted,
		api.StepStartedEvent{
			WorkflowID: flowID,
			StepID:     stepID,
			Inputs:     inputs,
		},
	)
}

func (e *Engine) CompleteStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, outputs api.Args, dur int64,
) error {
	return e.transitionStepExecution(
		ctx, flowID, stepID, api.StepCompleted, "complete",
		api.EventTypeStepCompleted,
		api.StepCompletedEvent{
			WorkflowID: flowID,
			StepID:     stepID,
			Outputs:    outputs,
			Duration:   dur,
		},
	)
}

func (e *Engine) FailStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, errMsg string,
) error {
	return e.transitionStepExecution(
		ctx, flowID, stepID, api.StepFailed, "fail",
		api.EventTypeStepFailed,
		api.StepFailedEvent{
			WorkflowID: flowID,
			StepID:     stepID,
			Error:      errMsg,
		},
	)
}

func (e *Engine) SkipStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, reason string,
) error {
	return e.transitionStepExecution(
		ctx, flowID, stepID, api.StepSkipped, "skip",
		api.EventTypeStepSkipped,
		api.StepSkippedEvent{
			WorkflowID: flowID,
			StepID:     stepID,
			Reason:     reason,
		},
	)
}

func (e *Engine) StartWork(
	ctx context.Context, flowID, stepID timebox.ID, token api.Token,
	inputs api.Args,
) error {
	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		ev, err := json.Marshal(api.WorkStartedEvent{
			WorkflowID: flowID,
			StepID:     stepID,
			Token:      token,
			Inputs:     inputs,
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
			WorkflowID: flowID,
			StepID:     stepID,
			Token:      token,
			Outputs:    outputs,
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
			WorkflowID: flowID,
			StepID:     stepID,
			Token:      token,
			Error:      errMsg,
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
			return fmt.Errorf("%s: %s", ErrAttributeAlreadySet, attr)
		}

		ev, err := json.Marshal(api.AttributeSetEvent{
			WorkflowID: flowID,
			StepID:     stepID,
			Key:        attr,
			Value:      value,
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

func (e *Engine) processWorkflow(flowID timebox.ID) {
	flow, ok := e.GetActiveWorkflow(flowID)
	if !ok {
		return
	}

	if !e.ensureScriptsCompiled(flowID, flow) {
		return
	}

	e.evaluateWorkflowState(e.ctx, flowID, flow)

	e.checkCompletableSteps(e.ctx, flowID, flow)

	ready := e.findReadySteps(flow)
	if len(ready) == 0 {
		e.handleTerminalState(flowID, flow)
		return
	}

	e.launchReadySteps(flowID, ready)
}

func (e *Engine) checkCompletableSteps(
	ctx context.Context, flowID timebox.ID, flow *api.WorkflowState,
) {
	for stepID, exec := range flow.Executions {
		if exec.Status != api.StepActive || exec.WorkItems == nil {
			continue
		}

		allDone := true
		hasFailed := false
		var failureError string

		for _, item := range exec.WorkItems {
			switch item.Status {
			case api.WorkFailed:
				hasFailed = true
				if failureError == "" {
					failureError = item.Error
				}
			case api.WorkPending:
				if !item.NextRetryAt.IsZero() {
					allDone = false
				}
			case api.WorkActive:
				allDone = false
			}
		}

		if !allDone {
			continue
		}

		if hasFailed {
			if failureError == "" {
				failureError = "work item failed"
			}
			_ = e.FailStepExecution(ctx, flowID, stepID, failureError)
		} else {
			outputs := aggregateWorkItemOutputs(
				exec.WorkItems, flow.ExecutionPlan.GetStep(stepID),
			)
			dur := time.Since(exec.StartedAt).Milliseconds()
			e.EnqueueStepResult(flowID, stepID, outputs, dur)
		}
	}
}

func aggregateWorkItemOutputs(
	items map[api.Token]*api.WorkState, step *api.Step,
) api.Args {
	completed := make([]*api.WorkState, 0, len(items))
	for _, item := range items {
		if item.Status == api.WorkCompleted {
			completed = append(completed, item)
		}
	}

	switch len(completed) {
	case 0:
		return api.Args{}
	case 1:
		return completed[0].Outputs
	default:
		aggregated := map[api.Name][]map[string]any{}
		var multiArgNames []api.Name
		if step != nil {
			multiArgNames = step.MultiArgNames()
		}

		for _, item := range completed {
			for outputName, outputValue := range item.Outputs {
				entry := map[string]any{}
				for _, argName := range multiArgNames {
					if val, ok := item.Inputs[argName]; ok {
						entry[string(argName)] = val
					}
				}
				entry["value"] = outputValue

				aggregated[outputName] = append(aggregated[outputName], entry)
			}
		}

		outputs := api.Args{}
		for name, values := range aggregated {
			outputs[name] = values
		}
		return outputs
	}
}

func (e *Engine) startWorkflow(
	ctx context.Context, flowID timebox.ID, plan *api.ExecutionPlan,
	initState api.Args, meta api.Metadata,
) error {
	existing, err := e.GetWorkflowState(ctx, flowID)
	if err == nil && existing.ID != "" {
		return ErrWorkflowExists
	}

	if err := plan.ValidateInputs(initState); err != nil {
		return err
	}

	if err := e.scripts.CompilePlan(plan); err != nil {
		return fmt.Errorf("%s: %w", ErrScriptCompileFailed, err)
	}

	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		ev, err := json.Marshal(api.WorkflowStartedEvent{
			WorkflowID:    flowID,
			ExecutionPlan: plan,
			InitialState:  initState,
			Metadata:      meta,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeWorkflowStarted, ev)
		return nil
	}

	_, err = e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
}

func (e *Engine) GetActiveWorkflow(
	flowID timebox.ID,
) (*api.WorkflowState, bool) {
	flow, err := e.GetWorkflowState(e.ctx, flowID)
	if err != nil {
		slog.Error("Failed to get workflow state",
			slog.Any("workflow_id", flowID),
			slog.Any("error", err))
		return nil, false
	}

	if flow.Status == api.WorkflowCompleted ||
		flow.Status == api.WorkflowFailed {
		return nil, false
	}

	return flow, true
}

func (e *Engine) ensureScriptsCompiled(
	flowID timebox.ID, flow *api.WorkflowState,
) bool {
	if !flow.ExecutionPlan.NeedsCompilation() {
		return true
	}

	if err := e.scripts.CompilePlan(flow.ExecutionPlan); err != nil {
		slog.Error("Failed to compile scripts",
			slog.Any("workflow_id", flowID),
			slog.Any("error", err))
		return false
	}

	return true
}

func (e *Engine) handleTerminalState(
	flowID timebox.ID, flow *api.WorkflowState,
) {
	if e.isWorkflowComplete(flow) {
		e.completeWorkflow(e.ctx, flowID, flow)
		return
	}

	if e.IsWorkflowFailed(flow) {
		e.failWorkflow(e.ctx, flowID, flow)
	}
}

func (e *Engine) launchReadySteps(flowID timebox.ID, ready []timebox.ID) {
	for _, stepID := range ready {
		e.wg.Add(1)
		go func(stepID timebox.ID) {
			defer e.wg.Done()
			e.executeStep(e.ctx, flowID, stepID)
		}(stepID)
	}
}

func (e *Engine) findReadySteps(flow *api.WorkflowState) []timebox.ID {
	var res []timebox.ID
	for _, step := range flow.ExecutionPlan.Steps {
		if e.isStepReadyForExec(step.ID, flow) {
			res = append(res, step.ID)
		}
	}
	return res
}

func (e *Engine) isStepReadyForExec(
	stepID timebox.ID, flow *api.WorkflowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return false
	}
	return e.isStepReady(stepID, flow)
}

func (e *Engine) isStepReady(stepID timebox.ID, flow *api.WorkflowState) bool {
	step := flow.ExecutionPlan.GetStep(stepID)
	for name, attr := range step.Attributes {
		if attr.Role == api.RoleRequired {
			if _, ok := flow.Attributes[name]; !ok {
				return false
			}
		}
	}
	return true
}

func (e *Engine) isWorkflowComplete(flow *api.WorkflowState) bool {
	for _, step := range flow.ExecutionPlan.Steps {
		if !e.isStepComplete(step.ID, flow) {
			return false
		}
	}
	return true
}

func (e *Engine) isStepComplete(
	stepID timebox.ID, flow *api.WorkflowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if !ok {
		return false
	}
	return exec.Status == api.StepCompleted || exec.Status == api.StepSkipped
}

func (e *Engine) evaluateWorkflowState(
	ctx context.Context, flowID timebox.ID, flow *api.WorkflowState,
) {
	for _, step := range flow.ExecutionPlan.Steps {
		exec, ok := flow.Executions[step.ID]
		if !ok || exec.Status != api.StepPending {
			continue
		}
		e.maybeSkipStep(ctx, flowID, step.ID, flow)
	}
}

func (e *Engine) maybeSkipStep(
	ctx context.Context, flowID, stepID timebox.ID, flow *api.WorkflowState,
) {
	if e.canStepComplete(stepID, flow) {
		return
	}

	err := e.SkipStepExecution(ctx, flowID, stepID,
		"step unreachable: required inputs cannot be satisfied",
	)
	if err != nil {
		slog.Error("Failed to skip step",
			slog.Any("step_id", stepID),
			slog.Any("error", err))
	}
}

func (e *Engine) IsWorkflowFailed(flow *api.WorkflowState) bool {
	for _, goalID := range flow.ExecutionPlan.GoalSteps {
		if !e.canStepComplete(goalID, flow) {
			return true
		}
	}
	return false
}

func (e *Engine) canStepComplete(
	stepID timebox.ID, flow *api.WorkflowState,
) bool {
	exec, ok := flow.Executions[stepID]
	if !ok {
		return false
	}

	if !hasCompletableStatus(exec.Status) {
		return exec.Status == api.StepCompleted
	}

	step := flow.ExecutionPlan.GetStep(stepID)
	if step == nil {
		return false
	}

	for requiredInputName, attr := range step.Attributes {
		if attr.Role == api.RoleRequired {
			if _, hasAttr := flow.Attributes[requiredInputName]; hasAttr {
				continue
			}
			if !e.HasInputProvider(stepID, requiredInputName, flow) {
				return false
			}
		}
	}

	return true
}

func (e *Engine) transitionStepExecution(
	ctx context.Context, flowID, stepID timebox.ID, toStatus api.StepStatus,
	action string, eventType timebox.EventType, eventData any,
) error {
	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		exec, ok := st.Executions[stepID]
		if !ok {
			return fmt.Errorf("%s: %s", ErrStepNotInPlan, stepID)
		}

		if can, exp := canTransitionTo(exec.Status, toStatus); !can {
			return fmt.Errorf(
				"%s: step %s cannot %s (status=%s, expected=%s)",
				ErrInvalidTransition, stepID, action, exec.Status, exp)
		}

		ev, err := json.Marshal(eventData)
		if err != nil {
			return err
		}
		ag.Raise(eventType, ev)
		return nil
	}

	_, err := e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	return err
}

func (e *Engine) failWorkflow(
	ctx context.Context, flowID timebox.ID, flow *api.WorkflowState,
) {
	var failed []string

	for _, step := range flow.ExecutionPlan.Steps {
		if exec, ok := flow.Executions[step.ID]; ok {
			failed = e.appendFailedStep(failed, step.ID, exec)
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

func (e *Engine) appendFailedStep(
	failed []string, stepID timebox.ID, exec *api.ExecutionState,
) []string {
	if exec.Status != api.StepFailed {
		return failed
	}

	if exec.Error == "" {
		return append(failed, string(stepID))
	}
	return append(failed, fmt.Sprintf("%s (%s)", stepID, exec.Error))
}

func (e *Engine) completeWorkflow(
	ctx context.Context, flowID timebox.ID, flow *api.WorkflowState,
) {
	result := api.Args{}

	for _, goalID := range flow.ExecutionPlan.GoalSteps {
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

func hasCompletableStatus(status api.StepStatus) bool {
	return status != api.StepCompleted &&
		status != api.StepFailed &&
		status != api.StepSkipped
}

func (e *Engine) HasInputProvider(
	stepID timebox.ID, name api.Name, flow *api.WorkflowState,
) bool {
	for _, planStep := range flow.ExecutionPlan.Steps {
		if planStep.ID == stepID {
			continue
		}

		if e.StepProvidesInput(planStep, name, flow) {
			return true
		}
	}
	return false
}

func (e *Engine) StepProvidesInput(
	step *api.Step, name api.Name, flow *api.WorkflowState,
) bool {
	for attrName, attr := range step.Attributes {
		if attrName == name && attr.Role == api.RoleOutput {
			return e.canStepComplete(step.ID, flow)
		}
	}
	return false
}

func workflowKey(flowID timebox.ID) timebox.AggregateID {
	return timebox.NewAggregateID("workflow", flowID)
}
