package engine

import (
	"errors"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util"
)

const localDispatchBackoff = 1 * time.Second

var (
	ErrDispatchUnavailable = errors.New("dispatch unavailable")
)

func (tx *flowTx) recoverWorkDispatch(fl api.FlowState) error {
	steps := tx.findWorkDispatchSteps(fl)
	if steps.IsEmpty() {
		return nil
	}

	now := tx.Now()
	for sid := range steps {
		if err := tx.scheduleWorkDispatch(sid, now); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) findWorkDispatchSteps(fl api.FlowState) util.Set[api.StepID] {
	steps := util.Set[api.StepID]{}
	now := e.Now()

	for sid, ex := range fl.Executions {
		if !policy.StepActive(ex.Status) {
			continue
		}
		st, ok := fl.Plan.Steps[sid]
		if !ok {
			continue
		}
		if policy.WorkReadyToDispatch(st, ex, now) {
			steps.Add(sid)
		}
	}

	return steps
}

func (tx *flowTx) scheduleWorkDispatch(sid api.StepID, at time.Time) error {
	fs := api.FlowStep{FlowID: tx.flowID, StepID: sid}
	return tx.scheduleEvent(workDispatchKey(fs), at,
		workDispatchRequested, sid)
}

func (tx *flowTx) handleWorkDispatch(sid api.StepID) error {
	fl := tx.Value()
	if fl.ID == "" || policy.FlowTerminal(fl.Status) {
		return nil
	}
	if !fl.DeactivatedAt.IsZero() {
		return nil
	}

	ex := fl.Executions[sid]
	if !policy.StepActive(ex.Status) {
		return nil
	}

	st := fl.Plan.Steps[sid]

	if policy.WorkReadyToDispatch(st, ex, tx.Now()) &&
		!tx.canDispatchLocally(st.ID) {
		return ErrDispatchUnavailable
	}

	started, err := tx.startPendingWork(st)
	if err != nil {
		return err
	}
	if len(started) == 0 {
		return nil
	}

	return tx.startContinuedWork(sid, st, started)
}

func (tx *flowTx) raiseDispatchDeferred(sid api.StepID) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeDispatchDeferred,
		api.DispatchDeferredEvent{
			FlowID: tx.flowID,
			StepID: sid,
		},
	)
}

func workDispatchKey(fs api.FlowStep) []string {
	return []string{string(fs.FlowID), "dispatch", string(fs.StepID)}
}
