package engine

import (
	"errors"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

// scheduleTimeouts arms the steps anchored during this transaction, since an
// anchor never moves once set and earlier ones were armed when they appeared
func (tx *flowTx) scheduleTimeouts() error {
	return tx.armTimeouts(tx.anchoredAtStart)
}

// recoverTimeouts arms the steps scheduleTimeouts skips, re-arming deadlines
// that were anchored before this transaction began
func (tx *flowTx) recoverTimeouts() error {
	return tx.armTimeouts(func(sid api.StepID) bool {
		return !tx.anchoredAtStart(sid)
	})
}

func (tx *flowTx) armTimeouts(skip func(api.StepID) bool) error {
	fl := tx.Value()
	if !flowHasTimeouts(&fl) || policy.FlowTerminal(fl.Status) {
		return nil
	}

	when := tx.Now()
	for sid := range fl.Executions {
		if skip(sid) {
			continue
		}
		if err := tx.scheduleStepTimeouts(fl, sid, when); err != nil {
			return err
		}
	}
	return nil
}

func (tx *flowTx) anchoredAtStart(sid api.StepID) bool {
	start := tx.start
	ex, ok := start.Executions[sid]
	if !ok || !policy.StepPending(ex.Status) {
		return false
	}
	anchor, err := tx.newStepEval(sid, &start, tx.Now()).requiredReadyAt()
	return err == nil && !anchor.IsZero()
}

func (tx *flowTx) scheduleStepTimeouts(
	fl api.FlowState, sid api.StepID, when time.Time,
) error {
	st, ok := fl.Plan.Steps[sid]
	if !ok || !stepHasTimeouts(st) {
		return nil
	}

	ex, ok := fl.Executions[sid]
	if !ok || !policy.StepPending(ex.Status) {
		return nil
	}

	s := tx.newStepEval(sid, &fl, when)
	anchor, err := s.requiredReadyAt()
	if err != nil {
		return nil
	}
	if anchor.IsZero() {
		return nil
	}

	for name, attr := range s.step.Attributes {
		if !attr.IsOptional() || attr.OptionalDeadline() <= 0 {
			continue
		}
		dec := s.optionalDecisionAt(name, attr, anchor)
		if dec.ready {
			if err := tx.scheduleTimeoutTask(sid, name, when); err != nil {
				return err
			}
			continue
		}
		if dec.nextAt.IsZero() {
			continue
		}
		if err := tx.scheduleTimeoutTask(sid, name, dec.nextAt); err != nil {
			return err
		}
	}
	return nil
}

func (tx *flowTx) scheduleTimeoutTask(
	sid api.StepID, name api.Name, at time.Time,
) error {
	fs := api.FlowStep{FlowID: tx.flowID, StepID: sid}
	return tx.scheduleEvent(timeoutKey(fs, name), at, stepTimeoutElapsed,
		scheduledTimeout{StepID: sid, Name: name})
}

func (tx *flowTx) handleStepTimeout(
	sid api.StepID, name api.Name, when time.Time,
) error {
	fl := tx.Value()
	if policy.FlowTerminal(fl.Status) {
		return nil
	}

	ex, ok := fl.Executions[sid]
	if !ok || !policy.StepPending(ex.Status) {
		return nil
	}

	ready, nextAt := tx.canStartStepAt(sid, &fl, when)
	if !ready {
		if !nextAt.IsZero() {
			return tx.scheduleTimeoutTask(sid, name, nextAt)
		}
		return nil
	}

	err := tx.prepareStep(sid)
	if err != nil {
		if errors.Is(err, ErrStepAlreadyPending) {
			return nil
		}
		return err
	}
	return tx.advanceFlow()
}

func flowHasTimeouts(fl *api.FlowState) bool {
	for _, st := range fl.Plan.Steps {
		if stepHasTimeouts(st) {
			return true
		}
	}
	return false
}

func stepHasTimeouts(st *api.Step) bool {
	for _, attr := range st.Attributes {
		if attr.IsOptional() && attr.OptionalDeadline() > 0 {
			return true
		}
	}
	return false
}

func timeoutKey(fs api.FlowStep, name api.Name) []string {
	return []string{string(fs.FlowID), "timeout", string(fs.StepID),
		string(name)}
}
