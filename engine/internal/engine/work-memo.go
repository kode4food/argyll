package engine

import (
	"encoding/json"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// memoForeverYear is the deadline year of a memoized result that never expires
const memoForeverYear = 9999

// lookupMemoized returns the outputs of an earlier successful work item for a
// memoized Step definition and inputs
func (tx *flowTx) lookupMemoized(
	st *api.Step, inputs api.Args,
) (api.Args, bool, error) {
	if st.DefaultedHandling() != api.HandlingMemoized {
		return nil, false, nil
	}
	path, err := memoPath(st, inputs)
	if err != nil {
		return nil, false, err
	}
	sched, err := tx.pendingMemo(path)
	if err != nil || sched == nil {
		return nil, false, err
	}
	outputs, err := sched.Message.GetValue[api.Args]()
	if err != nil {
		return nil, false, err
	}
	return outputs, true, nil
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

// recordMemoized retains a memoized Step's first successful outputs for its
// definition and inputs until the Step's memo TTL elapses
func (tx *flowTx) recordMemoized(
	sid api.StepID, tkn api.Token, outputs api.Args,
) error {
	fl := tx.Value()
	st := fl.Plan.Steps[sid]
	if st.DefaultedHandling() != api.HandlingMemoized {
		return nil
	}
	ex := fl.Executions[sid]
	path, err := memoPath(st, ex.Inputs.Apply(ex.WorkItems[tkn].Inputs))
	if err != nil {
		return err
	}
	sched, err := tx.pendingMemo(path)
	if err != nil || sched != nil {
		return err
	}
	data, err := json.Marshal(outputs)
	if err != nil {
		return err
	}
	return scheduler.Schedule(tx.storeTx.Transaction, path,
		memoDeadline(st, tx.Now()), &timebox.Message{
			AggregateID: events.FlowKey(tx.flowID),
			Type:        memoExpired,
			Data:        data,
		},
	)
}

// pendingMemo returns the unexpired memoized result at path, read within the
// current transaction so its commit is checked against it
func (tx *flowTx) pendingMemo(path []string) (*timebox.Schedule, error) {
	sched, err := scheduler.Load(tx.storeTx.Transaction, path)
	if err != nil || sched == nil || !sched.At.After(tx.Now()) {
		return nil, err
	}
	return sched, nil
}

func memoDeadline(st *api.Step, now time.Time) time.Time {
	if st.WorkConfig == nil || st.WorkConfig.MemoTTL == 0 {
		return time.Date(memoForeverYear, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	return now.Add(time.Duration(st.WorkConfig.MemoTTL) * time.Millisecond)
}

func memoPath(st *api.Step, inputs api.Args) ([]string, error) {
	stepKey, err := st.HashKey()
	if err != nil {
		return nil, err
	}
	inputKey, err := inputs.HashKey()
	if err != nil {
		return nil, err
	}
	return []string{"memo", stepKey, inputKey}, nil
}
