package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

var (
	ErrWorkItemNotFound      = errors.New("work item not found")
	ErrInvalidWorkTransition = errors.New("invalid work state transition")
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
func (e *Engine) FailWork(fs api.FlowStep, tkn api.Token, errMsg string) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		return tx.failWork(fs.StepID, tkn, errMsg)
	})
}

// NotCompleteWork marks a work item as not completed with specified error
func (e *Engine) NotCompleteWork(
	fs api.FlowStep, tkn api.Token, errMsg string,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		if err := tx.raiseWorkNotCompleted(fs.StepID, tkn, errMsg); err != nil {
			return err
		}

		return tx.handleWorkNotCompleted(fs.StepID, tkn)
	})
}

func (tx *flowTx) completeWork(
	sid api.StepID, tkn api.Token, outputs api.Args,
) error {
	err := tx.checkWorkTransition(sid, tkn, api.WorkSucceeded)
	if err != nil {
		return err
	}

	tx.memoizeWorkOutput(sid, tkn, outputs)

	if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  tx.flowID,
			StepID:  sid,
			Token:   tkn,
			Outputs: outputs,
		},
	); err != nil {
		return err
	}

	tx.clearRetryTask(sid, tkn)

	return tx.handleWorkSucceeded(sid)
}

func (tx *flowTx) clearRetryTask(sid api.StepID, tkn api.Token) {
	tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
		if !hasRetryTask(fl, sid, tkn) {
			return
		}
		tx.CancelTask(
			retryKey(api.FlowStep{FlowID: tx.flowID, StepID: sid}, tkn),
		)
	})
}

func (tx *flowTx) memoizeWorkOutput(
	sid api.StepID, tkn api.Token, outputs api.Args,
) {
	fl := tx.Value()
	st := fl.Plan.Steps[sid]
	if st.DefaultedHandling() != api.HandlingMemoized {
		return
	}

	ex := fl.Executions[sid]
	work := ex.WorkItems[tkn]
	inputs := ex.Inputs.Apply(work.Inputs)
	if err := tx.memoCache.Put(st, inputs, outputs); err != nil {
		slog.Warn("memo cache put failed",
			log.FlowID(tx.flowID), log.StepID(sid),
			log.Error(err))
	}
}

func (tx *flowTx) failWork(
	sid api.StepID, tkn api.Token, errMsg string,
) error {
	if err := tx.raiseWorkFailed(sid, tkn, errMsg); err != nil {
		return err
	}
	return tx.handleWorkFailed(sid)
}

func (tx *flowTx) checkWorkTransition(
	sid api.StepID, tkn api.Token, toStatus api.WorkStatus,
) error {
	fl := tx.Value()
	ex, ok := fl.Executions[sid]
	if !ok {
		return fmt.Errorf("%w: %s", ErrStepNotInPlan, sid)
	}

	work, ok := ex.WorkItems[tkn]
	if !ok {
		return fmt.Errorf("%w: %s", ErrWorkItemNotFound, tkn)
	}

	if !policy.WorkCanTransition(work.Status, toStatus) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidWorkTransition,
			work.Status, toStatus)
	}

	return nil
}

func (tx *flowTx) handleWorkSucceeded(sid api.StepID) error {
	if policy.FlowTerminal(tx.Value().Status) {
		return tx.handleTerminalWork(sid)
	}
	if !policy.StepActive(tx.Value().Executions[sid].Status) {
		return nil
	}

	completed, err := tx.checkStepCompletion(sid)
	if err != nil {
		return err
	}
	if !completed {
		return tx.handleWorkContinuation(sid)
	}

	return call.Perform(
		tx.skipPendingUnused,
		tx.startReadyPendingSteps,
		tx.checkTerminal,
	)
}

func (tx *flowTx) handleWorkFailed(sid api.StepID) error {
	return tx.handleStepFailure(sid)
}

func (tx *flowTx) handleWorkNotCompleted(
	sid api.StepID, tkn api.Token,
) error {
	if policy.FlowTerminal(tx.Value().Status) {
		return tx.maybeDeactivate()
	}
	if !policy.StepActive(tx.Value().Executions[sid].Status) {
		return nil
	}
	return call.Perform(
		call.WithArgs(tx.scheduleRetry, sid, tkn),
		call.WithArgs(tx.continueStepWork, sid, true),
		call.WithArg(tx.handleStepFailure, sid),
	)
}

// handleMemoCacheHit processes a memo cache hit by emitting WorkSucceeded
func (tx *flowTx) handleMemoCacheHit(
	sid api.StepID, tkn api.Token, outputs api.Args,
) error {
	if err := tx.checkWorkTransition(
		sid, tkn, api.WorkSucceeded,
	); err != nil {
		return err
	}
	if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  tx.flowID,
			StepID:  sid,
			Token:   tkn,
			Outputs: outputs,
		},
	); err != nil {
		return err
	}
	tx.clearRetryTask(sid, tkn)
	return tx.handleWorkSucceeded(sid)
}

func (tx *flowTx) raiseWorkFailed(
	sid api.StepID, tkn api.Token, errMsg string,
) error {
	if err := tx.checkWorkTransition(sid, tkn, api.WorkFailed); err != nil {
		return err
	}
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkFailed,
		api.WorkFailedEvent{
			FlowID: tx.flowID,
			StepID: sid,
			Token:  tkn,
			Error:  errMsg,
		},
	)
}

func (tx *flowTx) raiseRetryScheduled(
	sid api.StepID, tkn api.Token, work api.WorkState, nextRetryAt time.Time,
) error {
	if err := tx.checkWorkTransition(sid, tkn, api.WorkPending); err != nil {
		return err
	}
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkRetryScheduled,
		api.WorkRetryScheduledEvent{
			FlowID:      tx.flowID,
			StepID:      sid,
			Token:       tkn,
			RetryCount:  work.RetryCount + 1,
			NextRetryAt: nextRetryAt,
			Error:       work.Error,
		},
	)
}

func (tx *flowTx) raiseWorkNotCompleted(
	sid api.StepID, tkn api.Token, errMsg string,
) error {
	if err := tx.checkWorkTransition(
		sid, tkn, api.WorkNotCompleted,
	); err != nil {
		return err
	}
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkNotCompleted,
		api.WorkNotCompletedEvent{
			FlowID: tx.flowID,
			StepID: sid,
			Token:  tkn,
			Error:  errMsg,
		},
	)
}

func hasRetryTask(fl api.FlowState, sid api.StepID, tkn api.Token) bool {
	ex, ok := fl.Executions[sid]
	if !ok {
		return false
	}
	work, ok := ex.WorkItems[tkn]
	if !ok {
		return false
	}
	return !work.NextRetryAt.IsZero()
}
