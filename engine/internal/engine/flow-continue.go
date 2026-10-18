package engine

import (
	"maps"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

// advanceFlow settles pending steps, starts ready ones, and arms the rest's
// deadlines. Unused steps skip both before and after unreachable ones fail
func (tx *flowTx) advanceFlow() error {
	return call.Perform(
		tx.skipPendingUnused,
		tx.checkUnreachable,
		tx.skipPendingUnused,
		tx.checkTerminal,
		tx.startReadyPendingSteps,
		tx.scheduleTimeouts,
	)
}

// checkUnreachable finds and fails all pending steps that can no longer
// complete because their required inputs cannot be satisfied
func (tx *flowTx) checkUnreachable() error {
	for {
		failedAny := false
		fl := tx.Value()

		for sid, ex := range fl.Executions {
			if !policy.StepPending(ex.Status) {
				continue
			}
			if tx.canStepComplete(sid, &fl) {
				continue
			}

			st := fl.Plan.Steps[sid]
			reason := "required input no longer available"
			inputs, err := tx.collectStepInputs(st, &fl)
			if err != nil {
				reason = err.Error()
				inputs = ex.Inputs
			}

			if err := events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
				api.StepFailedEvent{
					FlowID: tx.flowID,
					StepID: sid,
					Error:  reason,
					Inputs: inputs,
				},
			); err != nil {
				return err
			}
			failedAny = true
			break
		}

		if !failedAny {
			return nil
		}
	}
}

func (tx *flowTx) skipPendingUnused() error {
	for {
		skip := false
		fl := tx.Value()

		for sid, ex := range fl.Executions {
			if !policy.StepPending(ex.Status) {
				continue
			}
			if tx.areOutputsNeeded(sid, &fl, keptGoals(&fl)) {
				continue
			}

			if err := events.Raise(tx.FlowAggregator, api.EventTypeStepSkipped,
				api.StepSkippedEvent{
					FlowID: tx.flowID,
					StepID: sid,
					Reason: "outputs not needed",
				},
			); err != nil {
				return err
			}
			skip = true
			break
		}

		if !skip {
			return nil
		}
	}
}

func (tx *flowTx) startReadyPendingSteps() error {
	for !policy.FlowTerminal(tx.Value().Status) {
		ready := tx.readyPendingSteps()
		if len(ready) == 0 {
			return nil
		}
		for _, sid := range ready {
			fl := tx.Value()
			if policy.FlowTerminal(fl.Status) {
				return nil
			}
			if !policy.StepPending(fl.Executions[sid].Status) {
				continue
			}
			if err := tx.prepareStep(sid); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tx *flowTx) readyPendingSteps() []api.StepID {
	fl := tx.Value()
	now := tx.Now()
	var res []api.StepID
	for _, sid := range slices.Sorted(maps.Keys(fl.Executions)) {
		if !policy.StepPending(fl.Executions[sid].Status) {
			continue
		}
		if ready, _ := tx.canStartStepAt(sid, &fl, now); ready {
			res = append(res, sid)
		}
	}
	return res
}
