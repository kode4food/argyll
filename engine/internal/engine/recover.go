package engine

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

var (
	ErrListActiveFlows        = errors.New("failed to list active flows")
	ErrGetFlowState           = errors.New("failed to get flow state")
	ErrInvalidFlowStatusEntry = errors.New("invalid flow status entry")
)

// recoverPageSize bounds each read of the active flow index during recovery
const recoverPageSize = 500

// RecoverFlows initiates recovery for all active root and child flows during engine startup,
// reading each active index a page at a time
func (e *Engine) RecoverFlows() error {
	now := e.Now()
	count := 0
	for _, status := range []string{
		events.FlowStatusActive,
		events.ChildStatusPrefix + events.FlowStatusActive,
	} {
		q := timebox.StatusQuery{
			Status: status,
			Type:   events.FlowPrefix,
			Limit:  recoverPageSize,
		}
		for {
			entries, err := e.flowStore.ListAggregatesByStatus(q)
			if err != nil {
				return errors.Join(ErrListActiveFlows, err)
			}
			for _, entry := range entries {
				fid, ok := events.ParseFlowID(entry.ID)
				if !ok {
					return errors.Join(ErrListActiveFlows,
						fmt.Errorf("%w: %s", ErrInvalidFlowStatusEntry, entry.ID.String()),
					)
				}
				// Reconcile through the same task committed-event wake-ups use, so a transient
				// failure is retried
				if err := e.scheduleFlowReconcile(fid, now); err != nil {
					return err
				}
			}
			count += len(entries)
			if len(entries) < q.Limit {
				break
			}
			q.After = &entries[len(entries)-1]
		}
	}

	if count == 0 {
		slog.Info("No flows to recover")
		return nil
	}
	slog.Info("Recovering flows", slog.Int("candidate_count", count))
	return nil
}

func (tx *flowTx) recoverRetries(fl api.FlowState) error {
	now := tx.Now()
	for sid, ex := range fl.Executions {
		for tkn, work := range ex.WorkItems {
			if retryAt, ok := policy.RecoverableDeadline(&ex, &work, now); ok {
				if err := tx.scheduleRetryTask(sid, tkn, retryAt); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
