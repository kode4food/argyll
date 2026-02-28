package engine

import (
	"fmt"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

// scheduleRetry handles retry decision for a specific work item
func (tx *flowTx) scheduleRetry(stepID api.StepID, token api.Token) error {
	exec, ok := tx.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return fmt.Errorf("%w: expected %s to be active, got %s",
			ErrInvariantViolated, stepID, exec.Status)
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
		if err := events.Raise(tx.FlowAggregator, api.EventTypeRetryScheduled,
			api.RetryScheduledEvent{
				FlowID:      tx.flowID,
				StepID:      stepID,
				Token:       token,
				RetryCount:  work.RetryCount + 1,
				NextRetryAt: nextRetryAt,
				Error:       work.Error,
			},
		); err != nil {
			return err
		}
		tx.OnSuccess(func(*api.FlowState) {
			tx.handleRetryScheduled(stepID, token, nextRetryAt)
		})
		return nil
	}

	// Permanent failure
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkFailed,
		api.WorkFailedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  token,
			Error:  work.Error,
		},
	)
}

func (tx *flowTx) handleWorkSucceeded(stepID api.StepID) error {
	// Terminal flows only record step completions for audit
	if flowTransitions.IsTerminal(tx.Value().Status) {
		if _, err := tx.checkStepCompletion(stepID); err != nil {
			return err
		}
		return tx.maybeDeactivate()
	}

	completed, err := tx.checkStepCompletion(stepID)
	if err != nil {
		return err
	}
	if !completed {
		return tx.handleWorkContinuation(stepID)
	}

	return call.Perform(
		tx.skipPendingUnused,
		tx.startReadyPendingSteps,
		tx.checkTerminal,
	)
}

func (tx *flowTx) handleWorkContinuation(stepID api.StepID) error {
	return tx.continueStepWork(stepID, true)
}

func (tx *flowTx) handleWorkFailed(stepID api.StepID) error {
	return call.Perform(
		call.WithArgs(tx.continueStepWork, stepID, true),
		call.WithArg(tx.handleStepFailure, stepID),
	)
}

func (tx *flowTx) handleWorkNotCompleted(
	stepID api.StepID, token api.Token,
) error {
	if flowTransitions.IsTerminal(tx.Value().Status) {
		return tx.maybeDeactivate()
	}
	return call.Perform(
		call.WithArgs(tx.scheduleRetry, stepID, token),
		call.WithArgs(tx.continueStepWork, stepID, true),
		call.WithArg(tx.handleStepFailure, stepID),
	)
}

func (tx *flowTx) startPendingWork(step *api.Step) (api.WorkItems, error) {
	stepID := step.ID
	exec, ok := tx.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil, fmt.Errorf("%w: expected %s to be active, got %s",
			ErrInvariantViolated, stepID, exec.Status)
	}

	limit := stepParallelism(step)
	active := countActiveWorkItems(exec.WorkItems)

	remaining := limit - active
	if remaining <= 0 {
		return nil, nil
	}

	now := tx.Now()
	started := api.WorkItems{}
	for token, work := range exec.WorkItems {
		if remaining == 0 {
			break
		}
		shouldStart, err := tx.shouldStartPendingWorkItem(step, work, now)
		if err != nil {
			return nil, err
		}
		if !shouldStart {
			continue
		}

		if step.Memoizable {
			if cached, ok := tx.Engine.memoCache.Get(step, work.Inputs); ok {
				err := tx.handleMemoCacheHit(stepID, token, cached)
				if err != nil {
					return nil, err
				}
				remaining--
				continue
			}
		}

		if err := tx.raiseWorkStarted(stepID, token, work.Inputs); err != nil {
			return nil, err
		}
		exec = tx.Value().Executions[stepID]
		started[token] = exec.WorkItems[token]
		remaining--
	}

	return started, nil
}

func (tx *flowTx) startRetryWorkItem(
	step *api.Step, token api.Token,
) (api.WorkItems, error) {
	stepID := step.ID
	exec, ok := tx.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil, nil
	}

	work, ok := exec.WorkItems[token]
	if !ok {
		return nil, nil
	}

	now := tx.Now()
	shouldStart := false
	switch work.Status {
	case api.WorkPending:
		var err error
		if shouldStart, err = tx.shouldStartRetryPending(
			step, work, exec.WorkItems, now,
		); err != nil {
			return nil, err
		}
	case api.WorkFailed:
		if work.NextRetryAt.IsZero() || work.NextRetryAt.After(now) {
			return nil, nil
		}
		shouldStart = true
	case api.WorkActive, api.WorkNotCompleted:
		shouldStart = true
	default:
		return nil, nil
	}
	if !shouldStart {
		return nil, nil
	}

	if err := tx.raiseWorkStarted(stepID, token, work.Inputs); err != nil {
		return nil, err
	}
	exec = tx.Value().Executions[stepID]
	started := api.WorkItems{}
	started[token] = exec.WorkItems[token]
	return started, nil
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
	tx.OnSuccess(func(flow *api.FlowState) {
		exec := flow.Executions[stepID]
		if clearRetryEntries {
			for token := range started {
				tx.Engine.CancelTask(
					retryKey(api.FlowStep{
						FlowID: tx.flowID,
						StepID: stepID,
					}, token),
				)
			}
		}
		tx.handleWorkItemsExecution(
			step, exec.Inputs, flow.Metadata, started,
		)
	})
	return nil
}

func (tx *flowTx) shouldStartPendingWorkItem(
	step *api.Step, work *api.WorkState, now time.Time,
) (bool, error) {
	stepID := step.ID
	if work.Status != api.WorkPending {
		return false, nil
	}
	if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(now) {
		return false, nil
	}
	shouldStart, err := tx.evaluateStepPredicate(step, work.Inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(stepID, err)
	}
	return shouldStart, nil
}

func (tx *flowTx) shouldStartRetryPending(
	step *api.Step, work *api.WorkState, items api.WorkItems, now time.Time,
) (bool, error) {
	stepID := step.ID
	if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(now) {
		return false, nil
	}
	limit := stepParallelism(step)
	active := countActiveWorkItems(items)
	if active >= limit {
		return false, nil
	}
	shouldStart, err := tx.evaluateStepPredicate(step, work.Inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(stepID, err)
	}
	return shouldStart, nil
}

func (tx *flowTx) raiseWorkStarted(
	stepID api.StepID, token api.Token, inputs api.Args,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkStarted,
		api.WorkStartedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  token,
			Inputs: inputs,
		},
	)
}

// handleMemoCacheHit processes a memo cache hit by emitting WorkSucceeded
func (tx *flowTx) handleMemoCacheHit(
	stepID api.StepID, token api.Token, outputs api.Args,
) error {
	if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  tx.flowID,
			StepID:  stepID,
			Token:   token,
			Outputs: outputs,
		},
	); err != nil {
		return err
	}
	tx.OnSuccess(func(flow *api.FlowState) {
		fs := api.FlowStep{FlowID: tx.flowID, StepID: stepID}
		tx.handleWorkSucceededCleanup(fs, token)
	})
	return tx.handleWorkSucceeded(stepID)
}

func (tx *flowTx) handleRetryScheduled(
	stepID api.StepID, token api.Token, nextRetryAt time.Time,
) {
	tx.Engine.scheduleRetryTask(api.FlowStep{
		FlowID: tx.flowID,
		StepID: stepID,
	}, token, nextRetryAt)
}

func stepParallelism(step *api.Step) int {
	if step.WorkConfig == nil || step.WorkConfig.Parallelism <= 0 {
		return 1
	}
	return step.WorkConfig.Parallelism
}

func countActiveWorkItems(items api.WorkItems) int {
	active := 0
	for _, work := range items {
		if work.Status == api.WorkActive {
			active++
		}
	}
	return active
}
