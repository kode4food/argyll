package engine

import (
	"fmt"
	"slices"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

// checkStepCompletion checks if a specific step can complete (all work items
// done) and raises appropriate completion or failure events
func (tx *flowTx) checkStepCompletion(sid api.StepID) (bool, error) {
	fl := tx.Value()
	ex, ok := fl.Executions[sid]
	if !ok || !policy.StepActive(ex.Status) {
		return false, fmt.Errorf("%w: expected %s to be active, got %s",
			ErrInvariantViolated, sid, ex.Status)
	}

	completion := policy.StepWorkCompletion(ex.WorkItems)
	if !completion.Done {
		return false, nil
	}

	if completion.Failed {
		if err := events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
			api.StepFailedEvent{
				FlowID: tx.flowID,
				StepID: sid,
				Error:  completion.FailureError,
				Inputs: ex.Inputs,
			},
		); err != nil {
			return true, err
		}
		st := fl.Plan.Steps[sid]
		return true, tx.startPendingCompensations(st, ex)
	}

	st := fl.Plan.Steps[sid]
	outputs := tx.collectStepOutputs(ex.WorkItems, st)
	outputs = tx.consumedOutputs(st, outputs, fl)
	dur := max(tx.Now().Sub(ex.StartedAt).Milliseconds(), int64(0))

	for key, value := range outputs {
		if !isOutputAttribute(st, key) {
			continue
		}
		if err := events.Raise(tx.FlowAggregator, api.EventTypeAttributeSet,
			api.AttributeSetEvent{
				FlowID: tx.flowID,
				StepID: sid,
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
			StepID:   sid,
			Outputs:  outputs,
			Duration: dur,
		},
	); err != nil {
		return true, err
	}
	if tx.Value().Status == api.FlowActive {
		tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
			tx.scheduleConsumerTimeouts(fl, sid, tx.Now())
		})
	}
	return true, nil
}

func (tx *flowTx) consumedOutputs(
	st *api.Step, outputs api.Args, fl api.FlowState,
) api.Args {
	if slices.Contains(fl.Plan.Goals, st.ID) {
		return outputs
	}

	res := api.Args{}
	for name, value := range outputs {
		attr := st.Attributes[name]
		if tx.needsOutput(name, attr, fl) {
			res[name] = value
		}
	}
	return res
}

func (tx *flowTx) handlePredicateFailure(
	sid api.StepID, inputs api.Args, err error,
) error {
	if raiseErr := events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
		api.StepFailedEvent{
			FlowID: tx.flowID,
			StepID: sid,
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
func (tx *flowTx) handleStepFailure(sid api.StepID) error {
	if policy.FlowTerminal(tx.Value().Status) {
		return tx.handleTerminalWork(sid)
	}
	if !policy.StepActive(tx.Value().Executions[sid].Status) {
		return nil
	}

	completed, err := tx.checkStepCompletion(sid)
	if err != nil || !completed {
		if err != nil {
			return err
		}
		return tx.continueStepWork(sid, false)
	}

	return call.Perform(
		tx.checkUnreachable,
		tx.checkTerminal,
		tx.startReadyPendingSteps,
	)
}

// handleTerminalWork handles work completion when the flow is already
// terminal: checks step completion then deactivates if no work remains
func (tx *flowTx) handleTerminalWork(sid api.StepID) error {
	// A late result on a settled step is recorded for the audit trail, but
	// there is no step outcome left to decide
	if policy.StepActive(tx.Value().Executions[sid].Status) {
		if _, err := tx.checkStepCompletion(sid); err != nil {
			return err
		}
	}
	return tx.maybeDeactivate()
}
