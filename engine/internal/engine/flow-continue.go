package engine

import (
	"errors"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// checkUnreachable finds and fails all pending steps that can no longer
// complete because their required inputs cannot be satisfied
func (tx *flowTx) checkUnreachable() error {
	for {
		failedAny := false
		flow := tx.Value()

		for stepID, exec := range flow.Executions {
			if exec.Status != api.StepPending {
				continue
			}
			if tx.canStepComplete(stepID, flow) {
				continue
			}

			if err := events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
				api.StepFailedEvent{
					FlowID: tx.flowID,
					StepID: stepID,
					Error:  "required input no longer available",
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
		flow := tx.Value()

		for stepID, exec := range flow.Executions {
			if exec.Status != api.StepPending {
				continue
			}
			if tx.areOutputsNeeded(stepID, flow) {
				continue
			}

			if err := events.Raise(tx.FlowAggregator, api.EventTypeStepSkipped,
				api.StepSkippedEvent{
					FlowID: tx.flowID,
					StepID: stepID,
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
	if flowTransitions.IsTerminal(tx.Value().Status) {
		return nil
	}

	for {
		flow := tx.Value()
		now := tx.Now()
		startedAny := false

		for stepID, exec := range flow.Executions {
			if exec.Status != api.StepPending {
				continue
			}

			ready, _ := tx.canStartStepAt(stepID, flow, now)
			if !ready {
				continue
			}

			if err := tx.prepareStep(stepID); err != nil {
				if errors.Is(err, ErrStepAlreadyPending) {
					continue
				}
				return err
			}
			startedAny = true
			break
		}

		if !startedAny || flowTransitions.IsTerminal(tx.Value().Status) {
			return nil
		}
	}
}
