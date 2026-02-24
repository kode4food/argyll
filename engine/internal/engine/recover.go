package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type backoffCalculator func(baseDelay int64, retryCount int) int64

var (
	ErrListFlowAggregates = errors.New("failed to list flow aggregates")
	ErrLoadPartitionState = errors.New("failed to load partition state")
	ErrGetFlowState       = errors.New("failed to get flow state")
)

const retryDispatchBackoff = 1 * time.Second

var backoffCalculators = map[string]backoffCalculator{
	api.BackoffTypeFixed: func(base int64, _ int) int64 {
		return base
	},
	api.BackoffTypeLinear: func(base int64, count int) int64 {
		return base * int64(count+1)
	},
	api.BackoffTypeExponential: func(base int64, count int) int64 {
		multiplier := math.Pow(2, float64(count))
		return int64(float64(base) * multiplier)
	},
}

// Retry logic

// ShouldRetry determines if a failed work item should be retried based on
// configured retry limits
func (e *Engine) ShouldRetry(step *api.Step, workItem *api.WorkState) bool {
	workConfig := e.resolveRetryConfig(step.WorkConfig)

	if workConfig.MaxRetries == 0 {
		return false
	}

	if workConfig.MaxRetries < 0 {
		return true
	}

	return workItem.RetryCount < workConfig.MaxRetries
}

// CalculateNextRetry calculates the next retry time using the configured
// backoff strategy (fixed, linear, or exponential)
func (e *Engine) CalculateNextRetry(
	config *api.WorkConfig, retryCount int,
) time.Time {
	config = e.resolveRetryConfig(config)

	calculator, ok := backoffCalculators[config.BackoffType]
	if !ok {
		calculator = backoffCalculators[api.BackoffTypeFixed]
	}

	delay := min(
		calculator(config.InitBackoff, retryCount), config.MaxBackoff,
	)

	return time.Now().Add(time.Duration(delay) * time.Millisecond)
}

// Recovery orchestration

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

// RecoverFlow resumes execution of a specific flow by queuing any pending work
// items for retry
func (e *Engine) RecoverFlow(flowID api.FlowID) error {
	flow, err := e.GetFlowState(flowID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGetFlowState, err)
	}

	if flowTransitions.IsTerminal(flow.Status) {
		return nil
	}

	retryableSteps := e.FindRetrySteps(flow)
	if retryableSteps.IsEmpty() {
		return nil
	}

	now := time.Now()
	for stepID := range retryableSteps {
		exec := flow.Executions[stepID]
		if exec.WorkItems == nil {
			continue
		}

		for token, workItem := range exec.WorkItems {
			var retryAt time.Time

			switch workItem.Status {
			case api.WorkActive, api.WorkNotCompleted:
				retryAt = now
			case api.WorkPending:
				if !workItem.NextRetryAt.IsZero() {
					retryAt = workItem.NextRetryAt
				} else if exec.Status == api.StepActive {
					retryAt = now
				} else {
					continue
				}
			case api.WorkFailed:
				if !workItem.NextRetryAt.IsZero() {
					retryAt = workItem.NextRetryAt
				} else {
					continue
				}
			default:
				continue
			}

			e.retryQueue.Push(&RetryItem{
				FlowID:      flowID,
				StepID:      stepID,
				Token:       token,
				NextRetryAt: retryAt,
			})
			e.RegisterTask(e.retryTask, retryAt)
		}
	}

	return nil
}

// FindRetrySteps identifies all steps in a flow that have work items that
// might need recovery (Active, Pending with NextRetryAt, or Failed with
// NextRetryAt)
func (e *Engine) FindRetrySteps(state *api.FlowState) util.Set[api.StepID] {
	retryableSteps := util.Set[api.StepID]{}

	for stepID, exec := range state.Executions {
		if exec.WorkItems == nil {
			continue
		}

		for _, workItem := range exec.WorkItems {
			if workItem.Status == api.WorkActive ||
				workItem.Status == api.WorkNotCompleted {
				retryableSteps.Add(stepID)
				break
			}
			if (workItem.Status == api.WorkPending ||
				workItem.Status == api.WorkFailed) &&
				!workItem.NextRetryAt.IsZero() {
				retryableSteps.Add(stepID)
				break
			}
		}
	}

	return retryableSteps
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
		if archiving.Contains(id) {
			continue
		}
		if deactivated.Contains(id) {
			continue
		}
		candidates = append(candidates, id)
	}
	return candidates
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

func (e *Engine) executeReadyRetries() {
	now := time.Now()

	items := e.retryQueue.PopReady(now)
	for _, item := range items {
		flow, err := e.GetFlowState(item.FlowID)
		if err != nil {
			if errors.Is(err, ErrFlowNotFound) {
				continue
			}
			e.requeueRetryItem(item)
			slog.Error("Failed to get flow state for retry",
				log.FlowID(item.FlowID),
				log.Error(err))
			continue
		}

		exec, ok := flow.Executions[item.StepID]
		if !ok || exec.WorkItems == nil {
			continue
		}

		if _, ok := exec.WorkItems[item.Token]; !ok {
			continue
		}

		step, ok := flow.Plan.Steps[item.StepID]
		if !ok {
			continue
		}

		fs := api.FlowStep{FlowID: item.FlowID, StepID: item.StepID}
		if err := e.retryWork(fs, step, item.Token, flow.Metadata); err != nil {
			e.requeueRetryItem(item)
			slog.Error("Failed to retry work item",
				log.FlowID(fs.FlowID),
				log.StepID(fs.StepID),
				log.Token(item.Token),
				log.Error(err))
		}
	}
}

func (e *Engine) retryWork(
	fs api.FlowStep, step *api.Step, token api.Token, meta api.Metadata,
) error {
	var started api.WorkItems
	var inputs api.Args

	err := e.flowTx(fs.FlowID, func(tx *flowTx) error {
		exec, ok := tx.Value().Executions[fs.StepID]
		if ok {
			inputs = exec.Inputs
		}
		var err error
		started, err = tx.startRetryWorkItem(fs.StepID, step, token)
		if err != nil {
			return err
		}
		if len(started) == 0 {
			return nil
		}
		tx.OnSuccess(func(*api.FlowState) {
			tx.handleWorkItemsExecution(
				fs.StepID, step, inputs, meta, started,
			)
		})
		return nil
	})
	return err
}

func (e *Engine) requeueRetryItem(item *RetryItem) {
	nextRetryAt := time.Now().Add(retryDispatchBackoff)
	e.retryQueue.Push(&RetryItem{
		FlowID:      item.FlowID,
		StepID:      item.StepID,
		Token:       item.Token,
		NextRetryAt: nextRetryAt,
	})
	e.RegisterTask(e.retryTask, nextRetryAt)
}

func (e *Engine) resolveRetryConfig(config *api.WorkConfig) *api.WorkConfig {
	res := e.config.Work
	if config == nil {
		return &res
	}

	if config.MaxRetries != 0 {
		res.MaxRetries = config.MaxRetries
	}
	if config.InitBackoff > 0 {
		res.InitBackoff = config.InitBackoff
	}
	if config.MaxBackoff > 0 {
		res.MaxBackoff = config.MaxBackoff
	}
	if config.BackoffType != "" {
		res.BackoffType = config.BackoffType
	}

	return &res
}
