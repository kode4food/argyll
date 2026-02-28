package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

var (
	ErrListFlowAggregates = errors.New("failed to list flow aggregates")
	ErrLoadPartitionState = errors.New("failed to load partition state")
	ErrGetFlowState       = errors.New("failed to get flow state")
)

// RecoverFlows initiates recovery for all active flows during engine startup
func (e *Engine) RecoverFlows() error {
	ids, err := e.listFlowAggregateIDs()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrListFlowAggregates, err)
	}

	if len(ids) == 0 {
		slog.Info("No flows to recover")
		return nil
	}

	state, err := e.GetPartitionState()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLoadPartitionState, err)
	}

	candidates := pruneRecoveryCandidates(ids, state)
	if len(candidates) == 0 {
		slog.Info("No flows to recover",
			slog.Int("candidate_count", 0))
		return nil
	}

	slog.Info("Recovering flows",
		slog.Int("candidate_count", len(candidates)),
	)

	active := util.Set[api.FlowID]{}
	for flowID := range state.Active {
		active.Add(flowID)
	}
	e.activateMissingFlows(candidates, active)
	e.recoverFlows(candidates)

	return nil
}

// RecoverFlow resumes execution of a specific flow by scheduling optional
// timeout callbacks and any pending work retries
func (e *Engine) RecoverFlow(flowID api.FlowID) error {
	flow, err := e.GetFlowState(flowID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGetFlowState, err)
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
		if exec.WorkItems == nil {
			continue
		}

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
		if exec.WorkItems == nil {
			continue
		}

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

func (e *Engine) listFlowAggregateIDs() ([]api.FlowID, error) {
	store := e.flowExec.GetStore()
	ids, err := store.ListAggregates(e.ctx, events.FlowKey("*"))
	if err != nil {
		return nil, err
	}

	seen := util.Set[api.FlowID]{}
	res := make([]api.FlowID, 0, len(ids))
	for _, id := range ids {
		flowID, ok := flowIDFromAggregateID(id)
		if !ok || seen.Contains(flowID) {
			continue
		}
		seen.Add(flowID)
		res = append(res, flowID)
	}
	slices.Sort(res)
	return res, nil
}

func (e *Engine) activateMissingFlows(
	ids []api.FlowID, active util.Set[api.FlowID],
) {
	for _, id := range ids {
		if active.Contains(id) {
			continue
		}
		flow, err := e.GetFlowState(id)
		if err != nil {
			slog.Error("Failed to load flow for activation repair",
				log.FlowID(id),
				log.Error(err))
			continue
		}
		e.activateFlow(id, flow)
		active.Add(id)
	}
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

func (e *Engine) activateFlow(id api.FlowID, flow *api.FlowState) {
	parentID, _ := api.GetMetaString[api.FlowID](
		flow.Metadata, api.MetaParentFlowID,
	)
	e.EnqueueEvent(api.EventTypeFlowActivated,
		api.FlowActivatedEvent{
			FlowID:       id,
			ParentFlowID: parentID,
			Labels:       flow.Labels,
		},
	)
}

func pruneRecoveryCandidates(
	ids []api.FlowID, state *api.PartitionState,
) []api.FlowID {
	deactivated := util.Set[api.FlowID]{}
	for _, info := range state.Deactivated {
		deactivated.Add(info.FlowID)
	}

	archiving := util.Set[api.FlowID]{}
	for flowID := range state.Archiving {
		archiving.Add(flowID)
	}

	candidates := make([]api.FlowID, 0, len(ids))
	for _, id := range ids {
		if archiving.Contains(id) || deactivated.Contains(id) {
			continue
		}
		candidates = append(candidates, id)
	}
	return candidates
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

func flowIDFromAggregateID(id timebox.AggregateID) (api.FlowID, bool) {
	if len(id) < 2 || id[0] != events.FlowPrefix {
		return "", false
	}
	flowID := api.FlowID(id[1])
	if flowID == "" {
		return "", false
	}
	return flowID, true
}
