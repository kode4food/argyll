package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

var (
	ErrListActiveFlows = errors.New("failed to list active flows")
	ErrGetFlowState    = errors.New("failed to get flow state")
)

// RecoverFlows initiates recovery for all active flows during engine startup
func (e *Engine) RecoverFlows() error {
	ids, err := e.listIndexedFlows(events.FlowStatusActive)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrListActiveFlows, err)
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
func (e *Engine) RecoverFlow(flowID api.FlowID) error {
	flow, err := e.GetFlowState(flowID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGetFlowState, err)
	}
	if err := validateParentMetadata(flow.Metadata); err != nil {
		return err
	}

	if flowTransitions.IsTerminal(flow.Status) {
		return nil
	}

	e.recoverTimeoutScans(flow)
	e.recoverRetryWork(flow)
	return nil
}

// FindRetrySteps identifies all steps in a flow that have work items that
// might need recovery
func (e *Engine) FindRetrySteps(state *api.FlowState) util.Set[api.StepID] {
	steps := util.Set[api.StepID]{}

	for stepID, exec := range state.Executions {
		for _, work := range exec.WorkItems {
			if !isRecoverable(exec, work) {
				continue
			}
			steps.Add(stepID)
			break
		}
	}

	return steps
}

func (e *Engine) recoverTimeoutScans(flow *api.FlowState) {
	e.scheduleTimeouts(flow, e.Now())
}

func (e *Engine) recoverRetryWork(flow *api.FlowState) {
	steps := e.FindRetrySteps(flow)
	if steps.IsEmpty() {
		return
	}

	now := e.Now()
	for stepID := range steps {
		exec := flow.Executions[stepID]
		for token, work := range exec.WorkItems {
			retryAt, ok := recoverableDeadline(exec, work, now)
			if !ok {
				continue
			}
			e.scheduleRetryTask(api.FlowStep{
				FlowID: flow.ID,
				StepID: stepID,
			}, token, retryAt)
		}
	}
}

func (e *Engine) listIndexedFlows(status string) ([]api.FlowID, error) {
	store := e.flowExec.GetStore()
	entries, err := store.ListAggregatesByStatus(e.ctx, status)
	if err != nil {
		return nil, err
	}

	seen := util.Set[api.FlowID]{}
	res := make([]api.FlowID, 0, len(entries))
	for _, entry := range entries {
		flowID, ok := events.ParseFlowID(entry.ID)
		if !ok || seen.Contains(flowID) {
			continue
		}
		seen.Add(flowID)
		res = append(res, flowID)
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

func isRecoverable(exec *api.ExecutionState, work *api.WorkState) bool {
	switch work.Status {
	case api.WorkActive, api.WorkNotCompleted:
		return true
	case api.WorkPending:
		if exec.Status == api.StepActive {
			return true
		}
		return !work.NextRetryAt.IsZero()
	case api.WorkFailed:
		return !work.NextRetryAt.IsZero()
	default:
		return false
	}
}

func recoverableDeadline(
	exec *api.ExecutionState, work *api.WorkState, now time.Time,
) (time.Time, bool) {
	switch work.Status {
	case api.WorkActive, api.WorkNotCompleted:
		return now, true
	case api.WorkPending:
		if !work.NextRetryAt.IsZero() {
			return work.NextRetryAt, true
		}
		if exec.Status == api.StepActive {
			return now, true
		}
		return time.Time{}, false
	case api.WorkFailed:
		if !work.NextRetryAt.IsZero() {
			return work.NextRetryAt, true
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}
