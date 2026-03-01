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
	fs api.FlowStep, tkn api.Token, outputs api.Args,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		return tx.completeWork(fs.StepID, tkn, outputs)
	})
}

// FailWork marks a work item as failed with the specified error message
func (e *Engine) FailWork(
	fs api.FlowStep, tkn api.Token, errMsg string,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		return tx.failWork(fs.StepID, tkn, errMsg)
	})
}

// NotCompleteWork marks a work item as not completed with specified error
func (e *Engine) NotCompleteWork(
	fs api.FlowStep, tkn api.Token, errMsg string,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		err := tx.checkWorkTransition(fs.StepID, tkn, api.WorkNotCompleted)
		if err != nil {
			return err
		}

		var retryTkn api.Token
		if exec, ok := tx.Value().Executions[fs.StepID]; ok {
			if work := exec.WorkItems[tkn]; work != nil {
				step := tx.Value().Plan.Steps[fs.StepID]
				if step != nil && !step.Memoizable && work.RetryCount > 0 {
					retryTkn = api.Token(uuid.New().String())
				}
			}
		}

		if err := tx.raiseWorkNotCompleted(
			fs.StepID, tkn, retryTkn, errMsg,
		); err != nil {
			return err
		}

		actualToken := tkn
		if retryTkn != "" {
			actualToken = retryTkn
		}
		return tx.handleWorkNotCompleted(fs.StepID, actualToken)
	})
}

func (tx *flowTx) handleWorkSucceededCleanup(fs api.FlowStep, tkn api.Token) {
	tx.CancelTask(retryKey(fs, tkn))
}

func (tx *flowTx) completeWork(
	stepID api.StepID, tkn api.Token, outputs api.Args,
) error {
	err := tx.checkWorkTransition(stepID, tkn, api.WorkSucceeded)
	if err != nil {
		return err
	}

	if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  tx.flowID,
			StepID:  stepID,
			Token:   tkn,
			Outputs: outputs,
		},
	); err != nil {
		return err
	}

	tx.OnSuccess(func(flow *api.FlowState) {
		if hasRetryTask(flow, stepID, tkn) {
			tx.handleWorkSucceededCleanup(api.FlowStep{
				FlowID: tx.flowID,
				StepID: stepID,
			}, tkn)
		}
		step := flow.Plan.Steps[stepID]
		if step != nil && step.Memoizable {
			exec := flow.Executions[stepID]
			work := flow.Executions[stepID].WorkItems[tkn]
			if exec != nil && work != nil {
				inputs := exec.Inputs.Apply(work.Inputs)
				err := tx.memoCache.Put(step, inputs, outputs)
				if err != nil {
					slog.Warn("memo cache put failed",
						log.FlowID(tx.flowID), log.StepID(stepID),
						log.Error(err))
				}
			}
		}
	})

	return tx.handleWorkSucceeded(stepID)
}

func (tx *flowTx) failWork(
	stepID api.StepID, tkn api.Token, errMsg string,
) error {
	if err := tx.checkWorkTransition(stepID, tkn, api.WorkFailed); err != nil {
		return err
	}
	if err := tx.raiseWorkFailed(stepID, tkn, errMsg); err != nil {
		return err
	}
	return tx.handleWorkFailed(stepID)
}

func (tx *flowTx) checkWorkTransition(
	stepID api.StepID, tkn api.Token, toStatus api.WorkStatus,
) error {
	flow := tx.Value()
	exec, ok := flow.Executions[stepID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
	}

	work, ok := exec.WorkItems[tkn]
	if !ok {
		return fmt.Errorf("%w: %s", ErrWorkItemNotFound, tkn)
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
	stepID api.StepID, tkn api.Token,
) error {
	if flowTransitions.IsTerminal(tx.Value().Status) {
		return tx.maybeDeactivate()
	}
	return call.Perform(
		call.WithArgs(tx.scheduleRetry, stepID, tkn),
		call.WithArgs(tx.continueStepWork, stepID, true),
		call.WithArg(tx.handleStepFailure, stepID),
	)
}

// handleMemoCacheHit processes a memo cache hit by emitting WorkSucceeded
func (tx *flowTx) handleMemoCacheHit(
	stepID api.StepID, tkn api.Token, outputs api.Args,
) error {
	if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  tx.flowID,
			StepID:  stepID,
			Token:   tkn,
			Outputs: outputs,
		},
	); err != nil {
		return err
	}
	tx.OnSuccess(func(flow *api.FlowState) {
		if hasRetryTask(flow, stepID, tkn) {
			fs := api.FlowStep{FlowID: tx.flowID, StepID: stepID}
			tx.handleWorkSucceededCleanup(fs, tkn)
		}
	})
	return tx.handleWorkSucceeded(stepID)
}

func (tx *flowTx) raiseWorkFailed(
	stepID api.StepID, tkn api.Token, errMsg string,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkFailed,
		api.WorkFailedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  tkn,
			Error:  errMsg,
		},
	)
}

func hasRetryTask(
	flow *api.FlowState, stepID api.StepID, tkn api.Token,
) bool {
	exec, ok := flow.Executions[stepID]
	if !ok {
		return false
	}
	work, ok := exec.WorkItems[tkn]
	if !ok || work == nil {
		return false
	}
	return !work.NextRetryAt.IsZero()
}

func (tx *flowTx) raiseRetryScheduled(
	stepID api.StepID, tkn api.Token, work *api.WorkState,
	nextRetryAt time.Time,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeRetryScheduled,
		api.RetryScheduledEvent{
			FlowID:      tx.flowID,
			StepID:      stepID,
			Token:       tkn,
			RetryCount:  work.RetryCount + 1,
			NextRetryAt: nextRetryAt,
			Error:       work.Error,
		},
	)
}

func (tx *flowTx) raiseWorkNotCompleted(
	stepID api.StepID, tkn, retryTkn api.Token, errMsg string,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkNotCompleted,
		api.WorkNotCompletedEvent{
			FlowID:     tx.flowID,
			StepID:     stepID,
			Token:      tkn,
			RetryToken: retryTkn,
			Error:      errMsg,
		},
	)
}
