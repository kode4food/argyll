package engine

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

type backoffCalculator func(baseDelayMs int64, retryCount int) int64

var backoffCalculators = map[string]backoffCalculator{
	api.BackoffTypeFixed: func(base int64, _ int) int64 {
		return base
	},
	api.BackoffTypeLinear: func(base int64, count int) int64 {
		return base * int64(count+1)
	},
	api.BackoffTypeExponential: func(base int64, count int) int64 {
		multiplier := math.Pow(2, float64(count))
		return int64(float64(base) * multiplier)
	},
}

// Retry logic

func (e *Engine) ShouldRetry(step *api.Step, workItem *api.WorkState) bool {
	if !isRetryableError(workItem.Error) {
		return false
	}

	workConfig := step.WorkConfig
	if workConfig == nil {
		workConfig = &e.config.WorkConfig
	}

	if workConfig.MaxRetries == 0 {
		return false
	}

	if workConfig.MaxRetries < 0 {
		return true
	}

	return workItem.RetryCount < workConfig.MaxRetries
}

func isRetryableError(errorStr string) bool {
	if errorStr == "" {
		return true
	}

	if strings.Contains(errorStr, "step returned success=false") {
		return false
	}

	if strings.Contains(errorStr, "HTTP 4") {
		return false
	}

	return true
}

func (e *Engine) CalculateNextRetry(
	config *api.WorkConfig, retryCount int,
) time.Time {
	if config == nil {
		config = &e.config.WorkConfig
	}

	calculator, ok := backoffCalculators[config.BackoffType]
	if !ok {
		calculator = backoffCalculators[api.BackoffTypeFixed]
	}

	delayMs := calculator(config.BackoffMs, retryCount)

	if delayMs > config.MaxBackoffMs {
		delayMs = config.MaxBackoffMs
	}

	return time.Now().Add(time.Duration(delayMs) * time.Millisecond)
}

func (e *Engine) ScheduleRetry(
	ctx context.Context, flowID, stepID timebox.ID, token api.Token,
	errMsg string,
) error {
	flow, err := e.GetWorkflowState(ctx, flowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow state: %w", err)
	}

	step := flow.Plan.GetStep(stepID)
	if step == nil {
		return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
	}

	exec, ok := flow.Executions[stepID]
	if !ok {
		return fmt.Errorf("execution state not found for step: %s", stepID)
	}

	workItem, ok := exec.WorkItems[token]
	if !ok {
		return fmt.Errorf("work item not found: %s", token)
	}

	newRetryCount := workItem.RetryCount + 1
	nextRetryAt := e.CalculateNextRetry(step.WorkConfig, workItem.RetryCount)

	cmd := func(st *api.WorkflowState, ag *WorkflowAggregator) error {
		return util.Raise(ag, api.EventTypeRetryScheduled,
			api.RetryScheduledEvent{
				FlowID:      flowID,
				StepID:      stepID,
				Token:       token,
				RetryCount:  newRetryCount,
				NextRetryAt: nextRetryAt,
				Error:       errMsg,
			},
		)
	}

	_, err = e.workflowExec.Exec(ctx, workflowKey(flowID), cmd)
	if err != nil {
		return fmt.Errorf("failed to schedule retry: %w", err)
	}

	slog.Info("Retry scheduled",
		slog.Any("flow_id", flowID),
		slog.Any("step_id", stepID),
		slog.Any("token", token),
		slog.Int("retry_count", newRetryCount),
		slog.Any("next_retry_at", nextRetryAt))

	return nil
}

// Recovery orchestration

func (e *Engine) RecoverWorkflows(ctx context.Context) error {
	engineState, err := e.GetEngineState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get engine state: %w", err)
	}

	if len(engineState.ActiveWorkflows) == 0 {
		slog.Info("No workflows to recover")
		return nil
	}

	slog.Info("Recovering workflows",
		slog.Int("count", len(engineState.ActiveWorkflows)),
	)

	for flowID := range engineState.ActiveWorkflows {
		if err := e.RecoverWorkflow(ctx, flowID); err != nil {
			slog.Error("Failed to recover workflow",
				slog.Any("flow_id", flowID),
				slog.Any("error", err))
		}
	}

	return nil
}

func (e *Engine) RecoverWorkflow(ctx context.Context, flowID timebox.ID) error {
	flow, err := e.GetWorkflowState(ctx, flowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow state: %w", err)
	}

	if workflowTransitions.IsTerminal(flow.Status) {
		return nil
	}

	retriableSteps := e.FindRetrySteps(flow)
	if retriableSteps.IsEmpty() {
		return nil
	}

	slog.Info("Recovering workflow",
		slog.Any("flow_id", flowID),
		slog.Int("retriable_steps", retriableSteps.Len()))

	now := time.Now()
	for stepID := range retriableSteps {
		exec := flow.Executions[stepID]
		if exec.WorkItems == nil {
			continue
		}

		step := flow.Plan.GetStep(stepID)
		if step == nil {
			continue
		}

		for token, workItem := range exec.WorkItems {
			if workItem.Status == api.WorkPending &&
				!workItem.NextRetryAt.IsZero() &&
				workItem.NextRetryAt.Before(now) {
				slog.Info("Retrying work item",
					slog.Any("flow_id", flowID),
					slog.Any("step_id", stepID),
					slog.Any("token", token),
					slog.Int("retry_count", workItem.RetryCount))

				e.retryWork(ctx, flowID, stepID, step, workItem.Inputs)
			}
		}
	}

	return nil
}

func (e *Engine) FindRetrySteps(state *api.WorkflowState) util.Set[timebox.ID] {
	retriableSteps := util.Set[timebox.ID]{}

	for stepID, exec := range state.Executions {
		if exec.WorkItems == nil {
			continue
		}

		for _, workItem := range exec.WorkItems {
			if workItem.Status == api.WorkPending &&
				!workItem.NextRetryAt.IsZero() &&
				workItem.RetryCount > 0 {
				retriableSteps.Add(stepID)
				break
			}
		}
	}

	return retriableSteps
}
