package engine

import (
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
		workConfig = &e.config.Work
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
		config = &e.config.Work
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
func (e *Engine) RecoverFlows() error {
	engineState, err := e.GetEngineState()
	if err != nil {
		return fmt.Errorf("failed to get engine state: %w", err)
	}

	if len(engineState.Active) == 0 {
		slog.Info("No flows to recover")
		return nil
	}

	slog.Info("Recovering flows",
		slog.Int("count", len(engineState.Active)))

	for flowID := range engineState.Active {
		if err := e.RecoverFlow(flowID); err != nil {
			slog.Error("Failed to recover flow",
				log.FlowID(flowID),
				log.Error(err))
		}
	}

	return nil
}

// RecoverFlow resumes execution of a specific flow by queuing any pending
// work items for retry
func (e *Engine) RecoverFlow(flowID api.FlowID) error {
	flow, err := e.GetFlowState(flowID)
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

		for token, workItem := range exec.WorkItems {
			var retryAt time.Time

			switch workItem.Status {
			case api.WorkActive, api.WorkNotCompleted:
				retryAt = now
			case api.WorkPending:
				if exec.Status == api.StepActive {
					retryAt = now
				} else if !workItem.NextRetryAt.IsZero() {
					retryAt = workItem.NextRetryAt
				} else {
					continue
				}
			case api.WorkFailed:
				if !workItem.NextRetryAt.IsZero() {
					retryAt = workItem.NextRetryAt
				} else {
					continue
				}
			default:
				continue
			}

			e.retryQueue.Push(&RetryItem{
				FlowID:      flowID,
				StepID:      stepID,
				Token:       token,
				NextRetryAt: retryAt,
			})
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
			if workItem.Status == api.WorkActive ||
				workItem.Status == api.WorkNotCompleted {
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
	var t retryTimer
	var timerC <-chan time.Time
	notifyC := e.retryQueue.Notify()

	resetTimer := func() {
		if nextTime, ok := e.retryQueue.Peek(); ok {
			timerC = t.Reset(nextTime)
		} else {
			t.Stop()
			timerC = nil
		}
	}

	resetTimer()

	for {
		select {
		case <-e.ctx.Done():
			t.Stop()
			return

		case _, ok := <-notifyC:
			if !ok {
				t.Stop()
				return
			}
			resetTimer()

		case <-timerC:
			e.executeReadyRetries()
			resetTimer()
		}
	}
}

func (e *Engine) executeReadyRetries() {
	now := time.Now()

	items := e.retryQueue.PopReady(now)
	for _, item := range items {
		flow, err := e.GetFlowState(item.FlowID)
		if err != nil {
			slog.Error("Failed to get flow state for retry",
				log.FlowID(item.FlowID),
				log.Error(err))
			continue
		}

		exec, ok := flow.Executions[item.StepID]
		if !ok || exec.WorkItems == nil {
			continue
		}

		workItem, ok := exec.WorkItems[item.Token]
		if !ok {
			continue
		}

		step, ok := flow.Plan.Steps[item.StepID]
		if !ok {
			continue
		}

		fs := FlowStep{FlowID: item.FlowID, StepID: item.StepID}
		e.retryWork(fs, step, item.Token, workItem, flow.Metadata)
	}
}

func (e *Engine) retryWork(
	fs FlowStep, step *api.Step, token api.Token, workItem *api.WorkState,
	meta api.Metadata,
) {
	execCtx := &ExecContext{
		engine: e,
		step:   step,
		inputs: workItem.Inputs,
		flowID: fs.FlowID,
		stepID: fs.StepID,
		meta:   meta,
	}

	execCtx.executeWorkItem(token, workItem)
}
