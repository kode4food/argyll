package engine

import (
	"errors"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

var (
	ErrWorkDeadlineExceeded = errors.New("work item deadline exceeded")
)

func (tx *flowTx) scheduleWorkDeadlineAt(
	sid api.StepID, tkn api.Token, at time.Time,
) error {
	fs := api.FlowStep{FlowID: tx.flowID, StepID: sid}
	return tx.scheduleEvent(deadlineKey(fs, tkn), at, workDeadlineElapsed,
		scheduledWork{StepID: sid, Token: tkn})
}

func (tx *flowTx) clearWorkDeadline(sid api.StepID, tkn api.Token) error {
	fs := api.FlowStep{FlowID: tx.flowID, StepID: sid}
	return tx.cancelEvent(deadlineKey(fs, tkn))
}

func (tx *flowTx) handleWorkDeadline(sid api.StepID, tkn api.Token) error {
	fl := tx.Value()
	at, ok := tx.workDeadline(fl, sid, tkn)
	if !ok {
		return nil
	}
	if at.After(tx.Now()) {
		return tx.scheduleWorkDeadlineAt(sid, tkn, at)
	}

	work := fl.Executions[sid].WorkItems[tkn]
	if policy.WorkCompActive(work.Status) {
		return tx.scheduleCompensationRetry(
			sid, tkn, ErrWorkDeadlineExceeded.Error(),
		)
	}
	if err := tx.raiseWorkNotCompleted(
		sid, tkn, ErrWorkDeadlineExceeded.Error(),
	); err != nil {
		return err
	}
	return tx.handleWorkNotCompleted(sid, tkn)
}

func (e *Engine) workDeadline(
	fl api.FlowState, sid api.StepID, tkn api.Token,
) (time.Time, bool) {
	work, ok := fl.Executions[sid].WorkItems[tkn]
	if !ok {
		return time.Time{}, false
	}
	return policy.WorkDeadline(
		fl.Plan.Steps[sid], work, e.defaultWorkTimeout(),
	)
}

func (e *Engine) defaultWorkTimeout() time.Duration {
	return time.Duration(e.config.StepTimeout) * time.Millisecond
}

// recoverInFlightWork bounds every in-flight attempt with the deadline its
// step implies, so an attempt outlives the node that claimed it
func (tx *flowTx) recoverInFlightWork(fl api.FlowState) error {
	for sid, ex := range fl.Executions {
		for tkn := range ex.WorkItems {
			if at, ok := tx.workDeadline(fl, sid, tkn); ok {
				if err := tx.scheduleWorkDeadlineAt(sid, tkn, at); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func deadlineKey(fs api.FlowStep, tkn api.Token) []string {
	return []string{
		string(fs.FlowID), "deadline", string(fs.StepID), string(tkn),
	}
}

func deadlinePrefix(fid api.FlowID) []string {
	return []string{string(fid), "deadline"}
}
