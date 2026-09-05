package engine

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
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

	e.recoverFlows(ids)

	return nil
}

func (e *Engine) recoverRetries(fl api.FlowState) {
	now := e.Now()
	for sid, ex := range fl.Executions {
		for tkn, work := range ex.WorkItems {
			if retryAt, ok := policy.RecoverableDeadline(ex, work, now); ok {
				e.scheduleRetryTask(api.FlowStep{
					FlowID: fl.ID,
					StepID: sid,
				}, tkn, retryAt)
			}
		}
	}
}

func (e *Engine) listIndexedFlows(status string) ([]api.FlowID, error) {
	store := e.flowExec.GetStore()
	entries, err := store.ListAggregatesByStatus(status)
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
func (e *Engine) recoverFlows(ids []api.FlowID) {
	now := e.Now()
	for _, id := range ids {
		e.scheduleFlowReconcile(id, now)
	}
}
