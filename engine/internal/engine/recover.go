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

// ShouldRetry determines if a failed work item should be retried based on the
// error type and configured retry limits
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

// CalculateNextRetry calculates the next retry time using the configured
// backoff strategy (fixed, linear, or exponential)
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

// ScheduleRetry schedules a failed work item for retry at a calculated future
// time based on the backoff configuration
func (e *Engine) ScheduleRetry(
	ctx context.Context, fs FlowStep, token api.Token, errMsg string,
) error {
	flow, err := e.GetFlowState(ctx, fs.FlowID)
	if err != nil {
		return fmt.Errorf("failed to get flow state: %w", err)
	}

	step := flow.Plan.GetStep(fs.StepID)
	if step == nil {
		return fmt.Errorf("%w: %s", ErrStepNotInPlan, fs.StepID)
	}

	exec, ok := flow.Executions[fs.StepID]
	if !ok {
		return fmt.Errorf("execution state not found for step: %s", fs.StepID)
	}

	workItem, ok := exec.WorkItems[token]
	if !ok {
		return fmt.Errorf("work item not found: %s", token)
	}

	newRetryCount := workItem.RetryCount + 1
	nextRetryAt := e.CalculateNextRetry(step.WorkConfig, workItem.RetryCount)

	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		return util.Raise(ag, api.EventTypeRetryScheduled,
			api.RetryScheduledEvent{
				FlowID:      fs.FlowID,
				StepID:      fs.StepID,
				Token:       token,
				RetryCount:  newRetryCount,
				NextRetryAt: nextRetryAt,
				Error:       errMsg,
			},
		)
	}

	_, err = e.flowExec.Exec(ctx, flowKey(fs.FlowID), cmd)
	if err != nil {
		return fmt.Errorf("failed to schedule retry: %w", err)
	}

	slog.Info("Retry scheduled",
		slog.Any("flow_id", fs.FlowID),
		slog.Any("step_id", fs.StepID),
		slog.Any("token", token),
		slog.Int("retry_count", newRetryCount),
		slog.Any("next_retry_at", nextRetryAt))

	return nil
}

// Recovery orchestration

// RecoverFlows initiates recovery for all active flows during engine
// startup
func (e *Engine) RecoverFlows(ctx context.Context) error {
	engineState, err := e.GetEngineState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get engine state: %w", err)
	}

	if len(engineState.ActiveFlows) == 0 {
		slog.Info("No flows to recover")
		return nil
	}

	slog.Info("Recovering flows",
		slog.Int("count", len(engineState.ActiveFlows)),
	)

	for flowID := range engineState.ActiveFlows {
		if err := e.RecoverFlow(ctx, flowID); err != nil {
			slog.Error("Failed to recover flow",
				slog.Any("flow_id", flowID),
				slog.Any("error", err))
		}
	}

	return nil
}

// RecoverFlow resumes execution of a specific flow by retrying any
// pending work items that are ready for retry
func (e *Engine) RecoverFlow(ctx context.Context, flowID timebox.ID) error {
	flow, err := e.GetFlowState(ctx, flowID)
	if err != nil {
		return fmt.Errorf("failed to get flow state: %w", err)
	}

	if flowTransitions.IsTerminal(flow.Status) {
		return nil
	}

	retriableSteps := e.FindRetrySteps(flow)
	if retriableSteps.IsEmpty() {
		return nil
	}

	slog.Info("Recovering flow",
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
			shouldRetry := false

			switch workItem.Status {
			case api.WorkActive:
				shouldRetry = true
			case api.WorkPending:
				if exec.Status == api.StepActive {
					shouldRetry = true
				} else if !workItem.NextRetryAt.IsZero() &&
					workItem.NextRetryAt.Before(now) {
					shouldRetry = true
				}
			case api.WorkFailed:
				if !workItem.NextRetryAt.IsZero() &&
					workItem.NextRetryAt.Before(now) {
					shouldRetry = true
				}
			}

			if shouldRetry {
				slog.Info("Retrying work item",
					slog.Any("flow_id", flowID),
					slog.Any("step_id", stepID),
					slog.Any("token", token),
					slog.String("status", string(workItem.Status)),
					slog.Int("retry_count", workItem.RetryCount))

				fs := FlowStep{FlowID: flowID, StepID: stepID}
				e.retryWork(ctx, fs, step, token, workItem)
			}
		}
	}

	return nil
}

// FindRetrySteps identifies all steps in a flow that have work items
// scheduled for retry
func (e *Engine) FindRetrySteps(state *api.FlowState) util.Set[timebox.ID] {
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
