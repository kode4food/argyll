package engine

import (
	"fmt"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
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
		return true, events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
			api.StepFailedEvent{
				FlowID: tx.flowID,
				StepID: sid,
				Error:  completion.FailureError,
				Inputs: ex.Inputs,
			},
		)
	}

	st := fl.Plan.Steps[sid]
	outputs := tx.collectStepOutputs(ex.WorkItems, st)
	outputs = tx.consumedOutputs(st, outputs, &fl)
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
	}

	return true, events.Raise(tx.FlowAggregator, api.EventTypeStepCompleted,
		api.StepCompletedEvent{
			FlowID:   tx.flowID,
			StepID:   sid,
			Outputs:  outputs,
			Duration: dur,
		},
	)
}

func (tx *flowTx) consumedOutputs(
	st *api.Step, outputs api.Args, fl *api.FlowState,
) api.Args {
	res := api.Args{}
	goals := keptGoals(fl)
	if slices.Contains(goals, st.ID) {
		for name, value := range outputs {
			if fl.HasInit(name) {
				continue
			}
			res[name] = value
		}
		return res
	}

	scope := goalScope(fl.Plan, goals)
	for name, value := range outputs {
		attr := st.Attributes[name]
		if attr.IsOutput() && tx.needsOutput(name, fl, scope, tx.Now()) {
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

	return tx.advanceFlow()
}

// settleStep records step outcomes and starts failed-step compensation before
// continuing work, advancing the flow, or checking deactivation
func (tx *flowTx) settleStep(sid api.StepID) error {
	var completed bool
	if policy.StepActive(tx.Value().Executions[sid].Status) {
		var err error
		completed, err = tx.checkStepCompletion(sid)
		if err != nil {
			return err
		}
	}
	fl := tx.Value()
	ex := fl.Executions[sid]
	if policy.StepFailed(ex.Status) {
		err := tx.startPendingCompensations(fl.Plan.Steps[sid], ex)
		if err != nil {
			return err
		}
	}
	if policy.FlowTerminal(tx.Value().Status) {
		return tx.maybeDeactivate()
	}
	if completed {
		return tx.advanceFlow()
	}
	if policy.StepActive(ex.Status) {
		return tx.continueStepWork(sid)
	}
	return nil
}
