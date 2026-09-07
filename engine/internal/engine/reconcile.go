package engine

import (
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// HandleCommitted requests reconciliation for committed flow events except
// attribute updates. Every online replica schedules tasks; none dispatches
// external work directly from the batch
func (e *Engine) HandleCommitted(evs ...*timebox.Event) {
	requested := map[api.FlowID]time.Time{}
	now := e.Now()

	for _, ev := range evs {
		fid, ok := events.ParseFlowID(ev.AggregateID)
		if !ok {
			continue
		}
		at := now
		switch api.EventType(ev.Type) {
		case api.EventTypeAttributeSet:
			continue
		case api.EventTypeDispatchDeferred:
			// No node could run the step, so poll rather than spin
			at = now.Add(localDispatchBackoff)
		}
		if prev, ok := requested[fid]; !ok || at.Before(prev) {
			requested[fid] = at
		}
	}

	for fid, at := range requested {
		e.scheduleFlowReconcile(fid, at)
	}
}

// RecoverFlow arms whatever the committed state of a flow still calls for. It
// only adds tasks, since a stale one rechecks state and stops, while a
// cancelled one that no path rearms would stall the flow for good
func (e *Engine) RecoverFlow(fid api.FlowID) error {
	return e.flowTx(fid, func(tx *flowTx) error {
		fl := tx.Value()
		if fl.ID == "" || !fl.DeactivatedAt.IsZero() {
			tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
				tx.CancelPrefixedTasks(flowTaskPrefix(fid))
			})
			return nil
		}
		if err := validateParentMetadata(fl.Metadata); err != nil {
			return err
		}

		if err := tx.recoverCompensations(); err != nil {
			return err
		}
		// Compensation runs on terminal flows, so bound its attempts before
		// the terminal handling below returns
		tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
			tx.recoverInFlightWork(fl)
		})

		if policy.FlowTerminal(tx.Value().Status) {
			return tx.maybeDeactivate()
		}

		tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
			tx.scheduleTimeouts(fl, tx.Now())
			tx.recoverWorkDispatch(fl)
			tx.recoverRetries(fl)
		})
		return nil
	})
}

// scheduleFlowReconcile arms the reconciliation of a single flow, keyed so that
// duplicate requests collapse into one task
func (e *Engine) scheduleFlowReconcile(fid api.FlowID, at time.Time) {
	e.ScheduleTask(reconcileKey(fid), at, func() error {
		err := e.RecoverFlow(fid)
		if err != nil {
			e.scheduleFlowReconcile(fid, e.Now().Add(localDispatchBackoff))
		}
		return err
	})
}

// flowTaskPrefix covers every task derived from a flow's state, which the
// reconcile task is deliberately keyed outside of
func flowTaskPrefix(fid api.FlowID) []string {
	return []string{string(fid)}
}

func reconcileKey(fid api.FlowID) []string {
	return []string{"reconcile", string(fid)}
}
