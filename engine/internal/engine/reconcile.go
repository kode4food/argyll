package engine

import (
	"errors"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// HandleCommitted requests reconciliation for every flow named by a committed
// batch, on every online replica. It never acts directly, since that would
// broadcast the same external work to all of them
func (e *Engine) HandleCommitted(evs ...*timebox.Event) {
	requested := map[api.FlowID]time.Time{}
	now := e.Now()

	for _, ev := range evs {
		fid, ok := events.ParseFlowID(ev.AggregateID)
		if !ok {
			continue
		}
		at := now
		if api.EventType(ev.Type) == api.EventTypeDispatchDeferred {
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
	fl, err := e.GetFlowState(fid)
	if err != nil {
		if errors.Is(err, ErrFlowNotFound) {
			e.CancelPrefixedTasks(flowTaskPrefix(fid))
			return nil
		}
		return errors.Join(ErrGetFlowState, err)
	}
	if err := validateParentMetadata(fl.Metadata); err != nil {
		return err
	}

	if !fl.DeactivatedAt.IsZero() {
		e.CancelPrefixedTasks(flowTaskPrefix(fid))
		e.releaseChildFlows(fl)
		return nil
	}

	e.recoverCompensations(fl)
	// Compensation runs on terminal flows, so bound its attempts before the
	// terminal handling below returns
	e.recoverInFlightWork(fl)

	if policy.FlowTerminal(fl.Status) {
		return e.reconcileTerminalFlow(fl)
	}

	e.scheduleTimeouts(fl, e.Now())
	e.recoverWorkDispatch(fl)
	e.recoverRetries(fl)
	return nil
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

// reconcileTerminalFlow settles a terminal flow with its parent and its
// children. Each part rechecks state, so a repeat changes nothing
func (e *Engine) reconcileTerminalFlow(fl api.FlowState) error {
	if err := e.completeParentWork(fl); err != nil {
		return err
	}
	return e.flowTx(fl.ID, func(tx *flowTx) error {
		return tx.maybeDeactivate()
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
