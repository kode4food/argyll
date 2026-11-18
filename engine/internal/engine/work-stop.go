package engine

import (
	"fmt"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

// CompleteWork marks a work item as successfully completed with the given output values and the
// usage its attempt reported
func (e *Engine) CompleteWork(
	fs api.FlowStep, tkn api.Token, outputs api.Args, usage api.Usage,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		if err := tx.completeWork(fs.StepID, tkn, outputs, usage); err != nil {
			return err
		}
		return tx.forwardUsage(usage)
	})
}

// FailWork marks a work item as failed with the error that ended it and the usage its attempt
// reported
func (e *Engine) FailWork(fs api.FlowStep, tkn api.Token, cause error, usage api.Usage) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		if err := tx.failWork(fs.StepID, tkn, errorText(cause), usage); err != nil {
			return err
		}
		return tx.forwardUsage(usage)
	})
}

// RetryWork marks a work item as not completed with the error that interrupted it and the usage
// its attempt reported, leaving its retry settings to try again
func (e *Engine) RetryWork(fs api.FlowStep, tkn api.Token, cause error, usage api.Usage) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		if err := tx.retryWork(fs.StepID, tkn, errorText(cause), usage); err != nil {
			return err
		}
		return tx.forwardUsage(usage)
	})
}

func (tx *flowTx) completeWork(
	sid api.StepID, tkn api.Token, outputs api.Args, usage api.Usage,
) error {
	err := tx.checkWorkTransition(sid, tkn, api.WorkSucceeded)
	if err != nil {
		return err
	}

	if err := tx.recordMemoized(sid, tkn, outputs); err != nil {
		return err
	}
	if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  tx.flowID,
			StepID:  sid,
			Token:   tkn,
			Outputs: outputs,
			Usage:   usage,
		},
	); err != nil {
		return err
	}

	return tx.settleStep(sid)
}

func (tx *flowTx) failWork(sid api.StepID, tkn api.Token, errMsg string, usage api.Usage) error {
	if err := tx.raiseWorkFailed(sid, tkn, errMsg, usage); err != nil {
		return err
	}
	return tx.settleStep(sid)
}

func (tx *flowTx) retryWork(
	sid api.StepID, tkn api.Token, errMsg string, usage api.Usage,
) error {
	if err := tx.raiseWorkNotCompleted(sid, tkn, errMsg, usage); err != nil {
		return err
	}
	return tx.handleWorkNotCompleted(sid, tkn)
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
		return fmt.Errorf("%w: %s", api.ErrWorkItemNotFound, tkn)
	}

	if !policy.WorkCanTransition(policy.TransitionArgs[api.WorkStatus]{
		From: work.Status,
		To:   toStatus,
	}) {
		return fmt.Errorf("%w: %s -> %s", api.ErrInvalidWorkTransition, work.Status, toStatus)
	}

	return nil
}

func (tx *flowTx) handleWorkNotCompleted(sid api.StepID, tkn api.Token) error {
	if policy.FlowTerminal(tx.Value().Status) {
		return tx.maybeDeactivate()
	}
	if !policy.StepActive(tx.Value().Executions[sid].Status) {
		return nil
	}
	return call.Perform(
		call.WithArgs(tx.scheduleRetry, sid, tkn), call.WithArg(tx.continueStepWork, sid),
		call.WithArg(tx.settleStep, sid),
	)
}

func (tx *flowTx) raiseWorkFailed(
	sid api.StepID, tkn api.Token, errMsg string, usage api.Usage,
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
			Usage:  usage,
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
	sid api.StepID, tkn api.Token, errMsg string, usage api.Usage,
) error {
	if err := tx.checkWorkTransition(sid, tkn, api.WorkNotCompleted); err != nil {
		return err
	}
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkNotCompleted,
		api.WorkNotCompletedEvent{
			FlowID: tx.flowID,
			StepID: sid,
			Token:  tkn,
			Error:  errMsg,
			Usage:  usage,
		},
	)
}
