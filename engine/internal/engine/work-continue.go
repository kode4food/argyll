package engine

import (
	"errors"
	"math"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type backoffCalculator func(baseDelay int64, retryCount int) int64

const retryDispatchBackoff = 1 * time.Second

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

// ShouldRetry determines if a failed work item should be retried based on
// configured retry limits
func (e *Engine) ShouldRetry(step *api.Step, work *api.WorkState) bool {
	workConfig := e.resolveRetryConfig(step.WorkConfig)

	if workConfig.MaxRetries == 0 {
		return false
	}
	if workConfig.MaxRetries < 0 {
		return true
	}

	return work.RetryCount < workConfig.MaxRetries
}

// CalculateNextRetry calculates the next retry time using the configured
// backoff strategy
func (e *Engine) CalculateNextRetry(
	config *api.WorkConfig, retryCount int,
) time.Time {
	return e.calculateNextRetryAt(e.Now(), config, retryCount)
}

func (e *Engine) calculateNextRetryAt(
	now time.Time, config *api.WorkConfig, retryCount int,
) time.Time {
	config = e.resolveRetryConfig(config)

	calculator, ok := backoffCalculators[config.BackoffType]
	if !ok {
		calculator = backoffCalculators[api.BackoffTypeFixed]
	}

	delay := min(
		calculator(config.InitBackoff, retryCount), config.MaxBackoff,
	)

	return now.Add(time.Duration(delay) * time.Millisecond)
}

func (tx *flowTx) scheduleRetry(stepID api.StepID, token api.Token) error {
	exec, ok := tx.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil
	}

	work, ok := exec.WorkItems[token]
	if !ok || work.Status != api.WorkNotCompleted {
		return nil
	}

	step := tx.Value().Plan.Steps[stepID]
	if tx.ShouldRetry(step, work) {
		nextRetryAt := tx.calculateNextRetryAt(
			tx.Now(), step.WorkConfig, work.RetryCount,
		)
		if err := tx.raiseRetryScheduled(stepID, token, work, nextRetryAt); err != nil {
			return err
		}
		tx.OnSuccess(func(*api.FlowState) {
			tx.handleRetryScheduled(stepID, token, nextRetryAt)
		})
		return nil
	}

	return tx.raiseWorkFailed(stepID, token, work.Error)
}

func (tx *flowTx) continueStepWork(
	stepID api.StepID, clearRetryEntries bool,
) error {
	step := tx.Value().Plan.Steps[stepID]
	started, err := tx.startPendingWork(step)
	if err != nil {
		return err
	}
	if len(started) == 0 {
		return nil
	}
	if clearRetryEntries {
		tx.OnSuccess(func(*api.FlowState) {
			for token := range started {
				tx.CancelTask(
					retryKey(api.FlowStep{
						FlowID: tx.flowID,
						StepID: stepID,
					}, token),
				)
			}
		})
	}
	return tx.startContinuedWork(stepID, step, started)
}

func (tx *flowTx) handleWorkContinuation(stepID api.StepID) error {
	return tx.continueStepWork(stepID, true)
}

func (tx *flowTx) handleRetryScheduled(
	stepID api.StepID, token api.Token, nextRetryAt time.Time,
) {
	tx.scheduleRetryTask(api.FlowStep{
		FlowID: tx.flowID,
		StepID: stepID,
	}, token, nextRetryAt)
}

func (tx *flowTx) startContinuedWork(
	stepID api.StepID, step *api.Step, started api.WorkItems,
) error {
	tx.OnSuccess(func(flow *api.FlowState) {
		exec := flow.Executions[stepID]
		tx.handleWorkItemsExecution(step, exec.Inputs, flow.Metadata, started)
	})
	return nil
}

func (e *Engine) retryWork(
	fs api.FlowStep, step *api.Step, token api.Token, meta api.Metadata,
) error {
	var started api.WorkItems
	var inputs api.Args

	err := e.flowTx(fs.FlowID, func(tx *flowTx) error {
		exec, ok := tx.Value().Executions[fs.StepID]
		if ok {
			inputs = exec.Inputs
		}
		var err error
		started, err = tx.startRetryWorkItem(step, token)
		if err != nil {
			return err
		}
		if len(started) == 0 {
			return nil
		}
		tx.OnSuccess(func(*api.FlowState) {
			tx.handleWorkItemsExecution(step, inputs, meta, started)
		})
		return nil
	})
	return err
}

func (e *Engine) scheduleRetryTask(
	fs api.FlowStep, token api.Token, retryAt time.Time,
) {
	e.ScheduleTask(retryKey(fs, token), retryAt, func() error {
		err := e.runRetryTask(fs, token)
		if err != nil {
			e.scheduleRetryTask(fs, token,
				e.Now().Add(retryDispatchBackoff),
			)
		}
		return err
	})
}

func (e *Engine) runRetryTask(fs api.FlowStep, token api.Token) error {
	flow, err := e.GetFlowState(fs.FlowID)
	if err != nil {
		if errors.Is(err, ErrFlowNotFound) {
			return nil
		}
		return err
	}
	if flowTransitions.IsTerminal(flow.Status) {
		return nil
	}

	exec, ok := flow.Executions[fs.StepID]
	if !ok {
		return nil
	}
	if _, ok := exec.WorkItems[token]; !ok {
		return nil
	}

	step, ok := flow.Plan.Steps[fs.StepID]
	if !ok {
		return nil
	}

	return e.retryWork(fs, step, token, flow.Metadata)
}

func (e *Engine) resolveRetryConfig(config *api.WorkConfig) *api.WorkConfig {
	res := e.config.Work
	if config == nil {
		return &res
	}

	if config.MaxRetries != 0 {
		res.MaxRetries = config.MaxRetries
	}
	if config.InitBackoff > 0 {
		res.InitBackoff = config.InitBackoff
	}
	if config.MaxBackoff > 0 {
		res.MaxBackoff = config.MaxBackoff
	}
	if config.BackoffType != "" {
		res.BackoffType = config.BackoffType
	}

	return &res
}

func retryKey(fs api.FlowStep, token api.Token) []string {
	return []string{
		"retry", string(fs.FlowID), string(fs.StepID), string(token),
	}
}

func retryPrefix(flowID api.FlowID) []string {
	return []string{"retry", string(flowID)}
}
