package engine

import (
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

// HandleCommitted requests reconciliation for flows whose committed work this
// node could not dispatch, so a replica able to run the step picks it up
func (e *Engine) HandleCommitted(evs ...*timebox.Event) {
	requested := util.Set[api.FlowID]{}
	for _, ev := range evs {
		if api.EventType(ev.Type) != api.EventTypeDispatchDeferred {
			continue
		}
		if fid, ok := events.ParseFlowID(ev.AggregateID); ok {
			requested.Add(fid)
		}
	}

	// No node could run the step, so poll rather than spin
	at := e.Now().Add(localDispatchBackoff)
	for fid := range requested {
		if e.ctx.Err() != nil {
			return
		}
		if err := e.scheduleFlowReconcile(fid, at); err != nil {
			slog.Error("Failed to schedule flow reconciliation",
				log.FlowID(fid), log.Error(err))
		}
	}
}

// RecoverFlow arms whatever the committed state of a flow still calls for. It
// only adds tasks, since a stale one rechecks state and stops, while a
// cancelled one that no path rearms would stall the flow for good
func (e *Engine) RecoverFlow(fid api.FlowID) error {
	return e.flowTx(fid, func(tx *flowTx) error {
		return tx.recoverFlow()
	})
}

func (tx *flowTx) recoverFlow() error {
	fl := tx.Value()
	if fl.ID == "" || !fl.DeactivatedAt.IsZero() {
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
	if err := tx.recoverInFlightWork(tx.Value()); err != nil {
		return err
	}

	for sid := range tx.Value().Executions {
		ex := tx.Value().Executions[sid]
		if !policy.StepActive(ex.Status) || len(ex.WorkItems) != 0 {
			continue
		}
		if err := tx.continueStepWork(sid); err != nil {
			return err
		}
	}
	if policy.FlowTerminal(tx.Value().Status) {
		return tx.maybeDeactivate()
	}

	fl = tx.Value()
	return call.Perform(
		call.WithArg(tx.recoverWorkDispatch, &fl),
		call.WithArg(tx.recoverRetries, fl),
		tx.advanceFlow,
	)
}

// scheduleFlowReconcile arms the reconciliation of a single flow, keyed so that
// duplicate requests collapse into one task
func (e *Engine) scheduleFlowReconcile(fid api.FlowID, at time.Time) error {
	return e.flowStore.Transact(func(tx *timebox.Transaction) error {
		return e.scheduler.Schedule(tx, reconcileKey(fid), at,
			&timebox.Message{
				AggregateID: events.FlowKey(fid),
				Type:        flowReconcileRequested,
			})
	})
}

func reconcileKey(fid api.FlowID) []string {
	return []string{"reconcile", string(fid)}
}
