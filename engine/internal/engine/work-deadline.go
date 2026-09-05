package engine

import (
	"errors"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	ErrWorkDeadlineExceeded = errors.New("work item deadline exceeded")
	ErrChildFlowMissing     = errors.New("child flow was never started")
)

// scheduleWorkDeadlineAt arms the expiry check for an in-flight attempt. Every
// replica arms it, so the attempt outlives the node that started it
func (e *Engine) scheduleWorkDeadlineAt(
	fs api.FlowStep, tkn api.Token, at time.Time,
) {
	e.ScheduleTask(deadlineKey(fs, tkn), at, func() error {
		err := e.runWorkDeadline(fs, tkn)
		if err != nil {
			e.scheduleWorkDeadlineAt(fs, tkn, e.Now().Add(
				localDispatchBackoff,
			))
		}
		return err
	})
}

func (e *Engine) runWorkDeadline(fs api.FlowStep, tkn api.Token) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		fl := tx.Value()
		at, ok := tx.workDeadline(fl, fs.StepID, tkn)
		if !ok {
			return nil
		}
		if at.After(tx.Now()) {
			tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
				tx.scheduleWorkDeadlineAt(fs, tkn, at)
			})
			return nil
		}

		work := fl.Executions[fs.StepID].WorkItems[tkn]
		if policy.WorkAwaitsChildFlow(fl.Plan.Steps[fs.StepID], work) {
			return tx.settleMissingChildFlow(fs, tkn)
		}
		if policy.WorkCompActive(work.Status) {
			return tx.scheduleCompensationRetry(
				fs.StepID, tkn, ErrWorkDeadlineExceeded.Error(),
			)
		}
		if err := tx.raiseWorkNotCompleted(
			fs.StepID, tkn, ErrWorkDeadlineExceeded.Error(),
		); err != nil {
			return err
		}
		return tx.handleWorkNotCompleted(fs.StepID, tkn)
	})
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
func (e *Engine) recoverInFlightWork(fl api.FlowState) {
	for sid, ex := range fl.Executions {
		fs := api.FlowStep{FlowID: fl.ID, StepID: sid}
		for tkn := range ex.WorkItems {
			if at, ok := e.workDeadline(fl, sid, tkn); ok {
				e.scheduleWorkDeadlineAt(fs, tkn, at)
			}
		}
	}
}

// settleMissingChildFlow settles work whose child was never started, so the
// retry path can launch it again; a live child is left alone however long
func (tx *flowTx) settleMissingChildFlow(
	fs api.FlowStep, tkn api.Token,
) error {
	_, err := tx.GetFlowState(childFlowID(fs, tkn))
	if !errors.Is(err, ErrFlowNotFound) {
		return err
	}

	if err := tx.raiseWorkNotCompleted(
		fs.StepID, tkn, ErrChildFlowMissing.Error(),
	); err != nil {
		return err
	}
	return tx.handleWorkNotCompleted(fs.StepID, tkn)
}

func deadlineKey(fs api.FlowStep, tkn api.Token) []string {
	return []string{
		string(fs.FlowID), "deadline", string(fs.StepID), string(tkn),
	}
}

func deadlinePrefix(fid api.FlowID) []string {
	return []string{string(fid), "deadline"}
}
