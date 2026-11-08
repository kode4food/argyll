package engine

import (
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// lookupMemoized returns the outputs of an earlier successful work item for a
// memoized Step definition and inputs
func (tx *flowTx) lookupMemoized(
	st *api.Step, inputs api.Args,
) (api.Args, bool, error) {
	if st.DefaultedHandling() != api.HandlingMemoized {
		return nil, false, nil
	}
	state, err := tx.memoTx(st, inputs, func(*MemoAggregator) error {
		return nil
	})
	if err != nil || state == nil {
		return nil, false, err
	}
	return state.Outputs, true, nil
}

// applyMemoized succeeds a work item with memoized outputs
func (tx *flowTx) applyMemoized(
	sid api.StepID, tkn api.Token, outputs api.Args,
) error {
	if err := tx.checkWorkTransition(
		sid, tkn, api.WorkSucceeded,
	); err != nil {
		return err
	}
	if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  tx.flowID,
			StepID:  sid,
			Token:   tkn,
			Outputs: outputs,
		},
	); err != nil {
		return err
	}
	return tx.settleStep(sid)
}

// recordMemoized retains a memoized Step's first successful work item for its
// definition and inputs
func (tx *flowTx) recordMemoized(data api.WorkSucceededEvent) error {
	fl := tx.Value()
	st := fl.Plan.Steps[data.StepID]
	if st.DefaultedHandling() != api.HandlingMemoized {
		return nil
	}
	ex := fl.Executions[data.StepID]
	inputs := ex.Inputs.Apply(ex.WorkItems[data.Token].Inputs)
	_, err := tx.memoTx(st, inputs, func(ag *MemoAggregator) error {
		if ag.Value() != nil {
			return nil
		}
		return events.Raise(ag, api.EventTypeWorkSucceeded, data)
	})
	return err
}

// memoTx runs fn against one memoized result's aggregate inside the current
// store transaction
func (tx storeTx) memoTx(
	st *api.Step, inputs api.Args, fn func(*MemoAggregator) error,
) (*api.WorkSucceededEvent, error) {
	key, err := events.MemoKey(st, inputs)
	if err != nil {
		return nil, err
	}
	return tx.Exec(tx.memoExec, key,
		func(_ *api.WorkSucceededEvent, ag *MemoAggregator) error {
			return fn(ag)
		},
	)
}
