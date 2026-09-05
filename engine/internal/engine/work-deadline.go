package engine

import (
	"errors"
	"time"

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
	fl, err := e.GetFlowState(fs.FlowID)
	if err != nil {
		if errors.Is(err, ErrFlowNotFound) {
			return nil
		}
		return err
	}

	at, ok := e.workDeadline(fl, fs.StepID, tkn)
	if !ok {
		return nil
	}
	if at.After(e.Now()) {
		// A newer attempt restamped the start, so wait out its deadline
		e.scheduleWorkDeadlineAt(fs, tkn, at)
		return nil
	}

	work := fl.Executions[fs.StepID].WorkItems[tkn]
	if policy.WorkAwaitsChildFlow(fl.Plan.Steps[fs.StepID], work) {
		return e.settleMissingChildFlow(fs, tkn)
	}

	settle := e.NotCompleteWork
	if policy.WorkCompActive(work.Status) {
		settle = e.NotCompleteCompensation
	}

	err = settle(fs, tkn, ErrWorkDeadlineExceeded.Error())
	if errors.Is(err, ErrInvalidWorkTransition) {
		// The attempt reached its real outcome first, so leave it alone
		return nil
	}
	return err
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
func (e *Engine) settleMissingChildFlow(
	fs api.FlowStep, tkn api.Token,
) error {
	_, err := e.GetFlowState(childFlowID(fs, tkn))
	if !errors.Is(err, ErrFlowNotFound) {
		return err
	}

	err = e.NotCompleteWork(fs, tkn, ErrChildFlowMissing.Error())
	if errors.Is(err, ErrInvalidWorkTransition) {
		return nil
	}
	return err
}

func deadlineKey(fs api.FlowStep, tkn api.Token) []string {
	return []string{
		string(fs.FlowID), "deadline", string(fs.StepID), string(tkn),
	}
}

func deadlinePrefix(fid api.FlowID) []string {
	return []string{string(fid), "deadline"}
}
