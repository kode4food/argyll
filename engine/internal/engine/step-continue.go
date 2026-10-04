package engine

import (
	"errors"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util"
)

func (tx *flowTx) scheduleTimeouts(fl api.FlowState, when time.Time) error {
	if !flowHasTimeouts(fl) {
		return nil
	}
	if err := tx.cancelEventPrefix(timeoutFlowPrefix(fl.ID)); err != nil {
		return err
	}
	if policy.FlowTerminal(fl.Status) {
		return nil
	}

	for sid := range fl.Executions {
		if err := tx.scheduleStepTimeouts(fl, sid, when, false); err != nil {
			return err
		}
	}
	return nil
}

func (tx *flowTx) scheduleConsumerTimeouts(
	fl api.FlowState, producerID api.StepID, when time.Time,
) error {
	if policy.FlowTerminal(fl.Status) {
		if flowHasTimeouts(fl) {
			return tx.cancelEventPrefix(timeoutFlowPrefix(fl.ID))
		}
		return nil
	}

	producer, ok := fl.Plan.Steps[producerID]
	if !ok {
		return nil
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
			if err := tx.scheduleStepTimeouts(
				fl, sid, when, true,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tx *flowTx) scheduleStepTimeouts(
	fl api.FlowState, sid api.StepID, when time.Time, clearExisting bool,
) error {
	st, ok := fl.Plan.Steps[sid]
	if !ok || !stepHasTimeouts(st) {
		return nil
	}

	fs := api.FlowStep{FlowID: fl.ID, StepID: sid}
	if clearExisting {
		if err := tx.cancelEventPrefix(timeoutStepPrefix(fs)); err != nil {
			return err
		}
	}

	if policy.FlowTerminal(fl.Status) {
		return nil
	}
	ex, ok := fl.Executions[sid]
	if !ok || !policy.StepPending(ex.Status) {
		return nil
	}

	s := tx.newStepEval(sid, fl, when)
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

	ready, nextAt := tx.canStartStepAt(sid, fl, when)
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
	return tx.skipPendingUnused()
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
