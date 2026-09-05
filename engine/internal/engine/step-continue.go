package engine

import (
	"errors"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

func (e *Engine) scheduleTimeouts(fl api.FlowState, when time.Time) {
	if !flowHasTimeouts(fl) {
		return
	}
	e.CancelPrefixedTasks(timeoutFlowPrefix(fl.ID))
	if policy.FlowTerminal(fl.Status) {
		return
	}

	for sid := range fl.Executions {
		e.scheduleStepTimeouts(fl, sid, when, false)
	}
}

func (e *Engine) scheduleConsumerTimeouts(
	fl api.FlowState, producerID api.StepID, when time.Time,
) {
	if policy.FlowTerminal(fl.Status) {
		if flowHasTimeouts(fl) {
			e.CancelPrefixedTasks(timeoutFlowPrefix(fl.ID))
		}
		return
	}

	producer, ok := fl.Plan.Steps[producerID]
	if !ok {
		return
	}

	seen := util.Set[api.StepID]{}
	for name, attr := range producer.Attributes {
		if !attr.IsOutput() {
			continue
		}
		deps, ok := fl.Plan.Attributes[name]
		if !ok {
			continue
		}
		for _, sid := range deps.Consumers {
			if seen.Contains(sid) {
				continue
			}
			seen.Add(sid)
			e.scheduleStepTimeouts(fl, sid, when, true)
		}
	}
}

func (e *Engine) scheduleStepTimeouts(
	fl api.FlowState, sid api.StepID, when time.Time, clearExisting bool,
) {
	st, ok := fl.Plan.Steps[sid]
	if !ok || !stepHasTimeouts(st) {
		return
	}

	fs := api.FlowStep{FlowID: fl.ID, StepID: sid}
	if clearExisting {
		e.CancelPrefixedTasks(timeoutStepPrefix(fs))
	}

	if policy.FlowTerminal(fl.Status) {
		return
	}
	ex, ok := fl.Executions[sid]
	if !ok || !policy.StepPending(ex.Status) {
		return
	}

	s := e.newStepEval(sid, fl, when)
	anchor, err := s.requiredReadyAt()
	if err != nil {
		return
	}
	if anchor.IsZero() {
		return
	}

	for name, attr := range s.step.Attributes {
		if !attr.IsOptional() || attr.OptionalDeadline() <= 0 {
			continue
		}
		dec := s.optionalDecisionAt(name, attr, anchor)
		if dec.ready {
			e.scheduleTimeoutTask(fs, name, when)
			continue
		}
		if dec.nextAt.IsZero() {
			continue
		}
		e.scheduleTimeoutTask(fs, name, dec.nextAt)
	}
}

func (e *Engine) scheduleTimeoutTask(
	fs api.FlowStep, name api.Name, at time.Time,
) {
	e.ScheduleTask(timeoutKey(fs, name), at, func() error {
		return e.runTimeoutTaskAt(fs, name, e.Now())
	})
}

func (e *Engine) runTimeoutTaskAt(
	fs api.FlowStep, name api.Name, when time.Time,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		fl := tx.Value()
		if policy.FlowTerminal(fl.Status) {
			return nil
		}

		ex, ok := fl.Executions[fs.StepID]
		if !ok || !policy.StepPending(ex.Status) {
			return nil
		}

		ready, nextAt := tx.canStartStepAt(fs.StepID, fl, when)
		if !ready {
			if !nextAt.IsZero() {
				tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
					tx.scheduleTimeoutTask(fs, name, nextAt)
				})
			}
			return nil
		}

		err := tx.prepareStep(fs.StepID)
		if err != nil {
			if errors.Is(err, ErrStepAlreadyPending) {
				return nil
			}
			return err
		}
		return tx.skipPendingUnused()
	})
}

func flowHasTimeouts(fl api.FlowState) bool {
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

func timeoutFlowPrefix(fid api.FlowID) []string {
	return []string{string(fid), "timeout"}
}

func timeoutStepPrefix(fs api.FlowStep) []string {
	return []string{string(fs.FlowID), "timeout", string(fs.StepID)}
}
