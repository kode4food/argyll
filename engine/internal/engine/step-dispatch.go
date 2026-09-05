package engine

import (
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util"
)

const localDispatchBackoff = 1 * time.Second

func (e *Engine) recoverWorkDispatch(fl api.FlowState) {
	steps := e.findWorkDispatchSteps(fl)
	if steps.IsEmpty() {
		return
	}

	now := e.Now()
	for sid := range steps {
		e.scheduleWorkDispatch(api.FlowStep{
			FlowID: fl.ID,
			StepID: sid,
		}, now)
	}
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

func (e *Engine) scheduleWorkDispatch(fs api.FlowStep, at time.Time) {
	e.ScheduleTask(workDispatchKey(fs), at, func() error {
		err := e.dispatchWork(fs)
		if err != nil {
			e.scheduleWorkDispatch(fs, e.Now().Add(localDispatchBackoff))
		}
		return err
	})
}

func (e *Engine) dispatchWork(fs api.FlowStep) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		fl := tx.Value()
		if fl.ID == "" || policy.FlowTerminal(fl.Status) {
			return nil
		}
		if !fl.DeactivatedAt.IsZero() {
			return nil
		}

		ex := fl.Executions[fs.StepID]
		if !policy.StepActive(ex.Status) {
			return nil
		}

		st := fl.Plan.Steps[fs.StepID]
		inputs := ex.Inputs
		meta := fl.Metadata

		if policy.WorkReadyToDispatch(st, ex, tx.Now()) &&
			!tx.canDispatchLocally(st.ID) {
			// An event per poll would never settle, so keep this local
			tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
				tx.scheduleWorkDispatch(fs, tx.Now().Add(
					localDispatchBackoff,
				))
			})
			return nil
		}

		started, err := tx.startPendingWork(st)
		if err != nil {
			return err
		}
		if len(started) == 0 {
			return nil
		}

		tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
			tx.executeStartedWork(st, inputs, meta, started)
		})
		return nil
	})
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
