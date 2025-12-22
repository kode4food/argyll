package engine

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
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

// ShouldRetry determines if a failed work item should be retried based on
// configured retry limits
func (e *Engine) ShouldRetry(step *api.Step, workItem *api.WorkState) bool {
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

	delayMs := min(
		calculator(config.BackoffMs, retryCount),
		config.MaxBackoffMs,
	)

	return time.Now().Add(time.Duration(delayMs) * time.Millisecond)
}

// Recovery orchestration

// RecoverFlows initiates recovery for all active flows during engine startup
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
		slog.Int("count", len(engineState.ActiveFlows)))

	for flowID := range engineState.ActiveFlows {
		if err := e.RecoverFlow(ctx, flowID); err != nil {
			slog.Error("Failed to recover flow",
				log.FlowID(flowID),
				log.Error(err))
		}
	}

	return nil
}

// RecoverFlow resumes execution of a specific flow by retrying any pending
// work items that are ready for retry
func (e *Engine) RecoverFlow(ctx context.Context, flowID api.FlowID) error {
	flow, err := e.GetFlowState(ctx, flowID)
	if err != nil {
		return fmt.Errorf("failed to get flow state: %w", err)
	}

	if flowTransitions.IsTerminal(flow.Status) {
		return nil
	}

	retryableSteps := e.FindRetrySteps(flow)
	if retryableSteps.IsEmpty() {
		return nil
	}

	now := time.Now()
	for stepID := range retryableSteps {
		exec := flow.Executions[stepID]
		if exec.WorkItems == nil {
			continue
		}

		step, ok := flow.Plan.Steps[stepID]
		if !ok {
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
				} else if isRetryable(workItem, now) {
					shouldRetry = true
				}
			case api.WorkFailed:
				if isRetryable(workItem, now) {
					shouldRetry = true
				}
			}

			if shouldRetry {
				fs := FlowStep{FlowID: flowID, StepID: stepID}
				e.retryWork(ctx, fs, step, token, workItem)
			}
		}
	}

	return nil
}

// FindRetrySteps identifies all steps in a flow that have work items that
// might need recovery (Active, Pending with NextRetryAt, or Failed with
// NextRetryAt)
func (e *Engine) FindRetrySteps(state *api.FlowState) util.Set[api.StepID] {
	retryableSteps := util.Set[api.StepID]{}

	for stepID, exec := range state.Executions {
		if exec.WorkItems == nil {
			continue
		}

		for _, workItem := range exec.WorkItems {
			if workItem.Status == api.WorkActive {
				retryableSteps.Add(stepID)
				break
			}
			if (workItem.Status == api.WorkPending ||
				workItem.Status == api.WorkFailed) &&
				!workItem.NextRetryAt.IsZero() {
				retryableSteps.Add(stepID)
				break
			}
		}
	}

	return retryableSteps
}

func (e *Engine) retryLoop() {
	ticker := time.NewTicker(e.config.RetryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.checkPendingRetries()
		}
	}
}

func (e *Engine) checkPendingRetries() {
	ctx := context.Background()

	engineState, err := e.GetEngineState(ctx)
	if err != nil {
		slog.Error("Failed to get engine state",
			log.Error(err))
		return
	}

	now := time.Now()
	for flowID := range engineState.ActiveFlows {
		flow, err := e.GetFlowState(ctx, flowID)
		if err != nil {
			continue
		}

		for stepID, exec := range flow.Executions {
			if exec.WorkItems == nil {
				continue
			}

			for token, workItem := range exec.WorkItems {
				if workItem.Status == api.WorkPending &&
					isRetryable(workItem, now) {
					if step, ok := flow.Plan.Steps[stepID]; ok {
						fs := FlowStep{FlowID: flowID, StepID: stepID}
						e.retryWork(ctx, fs, step, token, workItem)
					}
				}
			}
		}
	}
}

func (e *Engine) retryWork(
	ctx context.Context, fs FlowStep, step *api.Step, token api.Token,
	workItem *api.WorkState,
) {
	execCtx := &ExecContext{
		engine: e,
		step:   step,
		inputs: workItem.Inputs,
		flowID: fs.FlowID,
		stepID: fs.StepID,
	}

	execCtx.executeWorkItem(ctx, token, workItem)
}

func isRetryable(workItem *api.WorkState, now time.Time) bool {
	return !workItem.NextRetryAt.IsZero() && workItem.NextRetryAt.Before(now)
}
