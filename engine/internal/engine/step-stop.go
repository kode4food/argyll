package engine

import (
	"fmt"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

// checkStepCompletion checks if a specific step can complete (all work items
// done) and raises appropriate completion or failure events
func (tx *flowTx) checkStepCompletion(stepID api.StepID) (bool, error) {
	fl := tx.Value()
	ex, ok := fl.Executions[stepID]
	if !ok || !policy.StepActive(ex.Status) {
		return false, fmt.Errorf("%w: expected %s to be active, got %s",
			ErrInvariantViolated, stepID, ex.Status)
	}

	completion := policy.StepWorkCompletion(ex.WorkItems)
	if !completion.Done {
		return false, nil
	}

	if completion.Failed {
		return true, events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
			api.StepFailedEvent{
				FlowID: tx.flowID,
				StepID: stepID,
				Error:  completion.FailureError,
				Inputs: ex.Inputs,
			},
		)
	}

	st := fl.Plan.Steps[stepID]
	outputs := tx.collectStepOutputs(ex.WorkItems, st)
	dur := max(tx.Now().Sub(ex.StartedAt).Milliseconds(), int64(0))

	for key, value := range outputs {
		if !isOutputAttribute(st, key) {
			continue
		}
		if err := events.Raise(tx.FlowAggregator, api.EventTypeAttributeSet,
			api.AttributeSetEvent{
				FlowID: tx.flowID,
				StepID: stepID,
				Key:    key,
				Value:  value,
			},
		); err != nil {
			return false, err
		}
		fl = tx.Value()
	}

	if err := events.Raise(tx.FlowAggregator, api.EventTypeStepCompleted,
		api.StepCompletedEvent{
			FlowID:   tx.flowID,
			StepID:   stepID,
			Outputs:  outputs,
			Duration: dur,
		},
	); err != nil {
		return true, err
	}
	if tx.Value().Status == api.FlowActive {
		tx.OnSuccess(func(flow api.FlowState, _ []*timebox.Event) {
			tx.scheduleConsumerTimeouts(flow, stepID, tx.Now())
		})
	}
	return true, nil
}

func (tx *flowTx) handlePredicateFailure(
	stepID api.StepID, inputs api.Args, err error,
) error {
	if raiseErr := events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
		api.StepFailedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Error:  err.Error(),
			Inputs: inputs,
		},
	); raiseErr != nil {
		return raiseErr
	}

	return call.Perform(
		tx.checkUnreachable,
		tx.checkTerminal,
	)
}

// handleStepFailure handles common failure logic for work failure paths,
// checking step completion and propagating failures
func (tx *flowTx) handleStepFailure(stepID api.StepID) error {
	if policy.FlowTerminal(tx.Value().Status) {
		return tx.handleTerminalWork(stepID)
	}

	completed, err := tx.checkStepCompletion(stepID)
	if err != nil || !completed {
		if err != nil {
			return err
		}
		return tx.continueStepWork(stepID, false)
	}

	return call.Perform(
		tx.checkUnreachable,
		tx.checkTerminal,
		tx.startReadyPendingSteps,
	)
}

// handleTerminalWork handles work completion when the flow is already
// terminal: checks step completion then deactivates if no work remains
func (tx *flowTx) handleTerminalWork(stepID api.StepID) error {
	if _, err := tx.checkStepCompletion(stepID); err != nil {
		return err
	}
	return tx.maybeDeactivate()
}
