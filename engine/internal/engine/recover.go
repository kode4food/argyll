package engine

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
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

// RecoverFlow resumes execution of a specific flow by scheduling optional
// timeout callbacks and any pending work retries
func (e *Engine) RecoverFlow(fid api.FlowID) error {
	fl, err := e.GetFlowState(fid)
	if err != nil {
		return errors.Join(ErrGetFlowState, err)
	}
	if err := validateParentMetadata(fl.Metadata); err != nil {
		return err
	}

	e.recoverCompensations(fl)

	if policy.FlowTerminal(fl.Status) {
		// The parent's release may have been missed while this engine was down
		return e.flowTx(fid, func(tx *flowTx) error {
			return tx.maybeDeactivate()
		})
	}

	e.recoverWork(fl)
	return nil
}

func (e *Engine) recoverWork(fl api.FlowState) {
	e.recoverTimeouts(fl)
	e.recoverWorkDispatch(fl)
	e.recoverRetries(fl)
}

// FindRetrySteps identifies all steps in a flow that have work items that
// might need recovery
func (*Engine) FindRetrySteps(fl api.FlowState) util.Set[api.StepID] {
	steps := util.Set[api.StepID]{}

	for sid, ex := range fl.Executions {
		for _, work := range ex.WorkItems {
			if !policy.Recoverable(ex, work) {
				continue
			}
			steps.Add(sid)
			break
		}
	}

	return steps
}

func (e *Engine) recoverTimeouts(fl api.FlowState) {
	e.scheduleTimeouts(fl, e.Now())
}

func (e *Engine) recoverRetries(fl api.FlowState) {
	steps := e.FindRetrySteps(fl)
	if steps.IsEmpty() {
		return
	}

	now := e.Now()
	for sid := range steps {
		ex := fl.Executions[sid]
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

func (e *Engine) recoverFlows(ids []api.FlowID) {
	for _, id := range ids {
		if err := e.RecoverFlow(id); err != nil {
			slog.Error("Failed to recover flow",
				log.FlowID(id),
				log.Error(err))
		}
	}
}
