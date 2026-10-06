package engine

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util"
)

var (
	ErrListActiveFlows        = errors.New("failed to list active flows")
	ErrGetFlowState           = errors.New("failed to get flow state")
	ErrInvalidFlowStatusEntry = errors.New("invalid flow status entry")
)

// RecoverFlows initiates recovery for all active flows during engine startup
func (e *Engine) RecoverFlows() error {
	ids, err := e.listIndexedFlows(events.FlowStatusActive)
	if err != nil {
		return errors.Join(ErrListActiveFlows, err)
	}

	if len(ids) == 0 {
		slog.Info("No flows to recover")
		return nil
	}

	slog.Info("Recovering flows",
		slog.Int("candidate_count", len(ids)),
	)

	return e.recoverFlows(ids)
}

func (tx *flowTx) recoverRetries(fl api.FlowState) error {
	now := tx.Now()
	for sid, ex := range fl.Executions {
		for tkn, work := range ex.WorkItems {
			if retryAt, ok := policy.RecoverableDeadline(ex, work, now); ok {
				if err := tx.scheduleRetryTask(sid, tkn, retryAt); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (e *Engine) listIndexedFlows(status string) ([]api.FlowID, error) {
	entries, err := e.flowStore.ListAggregatesByStatus(timebox.StatusQuery{
		Status: status,
		Type:   events.FlowPrefix,
	})
	if err != nil {
		return nil, err
	}

	seen := util.Set[api.FlowID]{}
	res := make([]api.FlowID, 0, len(entries))
	for _, entry := range entries {
		fid, ok := events.ParseFlowID(entry.ID)
		if !ok {
			return nil, errors.Join(
				ErrListActiveFlows,
				fmt.Errorf("%w: %s", ErrInvalidFlowStatusEntry,
					entry.ID.String()),
			)
		}
		if seen.Contains(fid) {
			continue
		}
		seen.Add(fid)
		res = append(res, fid)
	}
	return res, nil
}

// recoverFlows reconciles the indexed flows through the same tasks the
// committed-event wake-ups use, so a transient failure is retried
func (e *Engine) recoverFlows(ids []api.FlowID) error {
	now := e.Now()
	for _, id := range ids {
		if err := e.scheduleFlowReconcile(id, now); err != nil {
			return err
		}
	}
	return nil
}
