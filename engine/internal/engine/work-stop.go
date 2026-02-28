package engine

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

// CompleteWork marks a work item as successfully completed with the given
// output values
func (e *Engine) CompleteWork(
	fs api.FlowStep, token api.Token, outputs api.Args,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		err := tx.checkWorkTransition(fs.StepID, token, api.WorkSucceeded)
		if err != nil {
			return err
		}

		if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
			api.WorkSucceededEvent{
				FlowID:  fs.FlowID,
				StepID:  fs.StepID,
				Token:   token,
				Outputs: outputs,
			},
		); err != nil {
			return err
		}
		tx.OnSuccess(func(flow *api.FlowState) {
			tx.handleWorkSucceededCleanup(fs, token)
			step := flow.Plan.Steps[fs.StepID]
			if step != nil && step.Memoizable {
				work := flow.Executions[fs.StepID].WorkItems[token]
				if work != nil {
					err := e.memoCache.Put(step, work.Inputs, outputs)
					if err != nil {
						slog.Warn("memo cache put failed",
							log.FlowID(fs.FlowID), log.StepID(fs.StepID),
							log.Error(err))
					}
				}
			}
		})
		return tx.handleWorkSucceeded(fs.StepID)
	})
}

// FailWork marks a work item as failed with the specified error message
func (e *Engine) FailWork(
	fs api.FlowStep, token api.Token, errMsg string,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		err := tx.checkWorkTransition(fs.StepID, token, api.WorkFailed)
		if err != nil {
			return err
		}

		if err := tx.raiseWorkFailed(fs.StepID, token, errMsg); err != nil {
			return err
		}
		return tx.handleWorkFailed(fs.StepID)
	})
}

// NotCompleteWork marks a work item as not completed with specified error
func (e *Engine) NotCompleteWork(
	fs api.FlowStep, token api.Token, errMsg string,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		err := tx.checkWorkTransition(fs.StepID, token, api.WorkNotCompleted)
		if err != nil {
			return err
		}

		var retryToken api.Token
		if exec, ok := tx.Value().Executions[fs.StepID]; ok {
			if work := exec.WorkItems[token]; work != nil {
				step := tx.Value().Plan.Steps[fs.StepID]
				if step != nil && !step.Memoizable && work.RetryCount > 0 {
					retryToken = api.Token(uuid.New().String())
				}
			}
		}

		if err := tx.raiseWorkNotCompleted(
			fs.StepID, token, retryToken, errMsg,
		); err != nil {
			return err
		}

		actualToken := token
		if retryToken != "" {
			actualToken = retryToken
		}
		return tx.handleWorkNotCompleted(fs.StepID, actualToken)
	})
}

func (tx *flowTx) handleWorkSucceededCleanup(fs api.FlowStep, token api.Token) {
	tx.Engine.CancelTask(retryKey(fs, token))
}

func (tx *flowTx) checkWorkTransition(
	stepID api.StepID, token api.Token, toStatus api.WorkStatus,
) error {
	flow := tx.Value()
	exec, ok := flow.Executions[stepID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
	}

	work, ok := exec.WorkItems[token]
	if !ok {
		return fmt.Errorf("%w: %s", ErrWorkItemNotFound, token)
	}

	if !workTransitions.CanTransition(work.Status, toStatus) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidWorkTransition,
			work.Status, toStatus)
	}

	return nil
}

func (tx *flowTx) handleWorkSucceeded(stepID api.StepID) error {
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

func (tx *flowTx) raiseWorkFailed(
	stepID api.StepID, token api.Token, errMsg string,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkFailed,
		api.WorkFailedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  token,
			Error:  errMsg,
		},
	)
}

func (tx *flowTx) raiseRetryScheduled(
	stepID api.StepID, token api.Token, work *api.WorkState,
	nextRetryAt time.Time,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeRetryScheduled,
		api.RetryScheduledEvent{
			FlowID:      tx.flowID,
			StepID:      stepID,
			Token:       token,
			RetryCount:  work.RetryCount + 1,
			NextRetryAt: nextRetryAt,
			Error:       work.Error,
		},
	)
}

func (tx *flowTx) raiseWorkNotCompleted(
	stepID api.StepID, token, retryToken api.Token, errMsg string,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkNotCompleted,
		api.WorkNotCompletedEvent{
			FlowID:     tx.flowID,
			StepID:     stepID,
			Token:      token,
			RetryToken: retryToken,
			Error:      errMsg,
		},
	)
}
