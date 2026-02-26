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

func (e *Engine) recoverTimeoutScans(flow *api.FlowState) {
	e.scheduleTimeouts(flow, time.Now())
}

func (e *Engine) recoverRetryWork(flow *api.FlowState) {
	retryableSteps := e.FindRetrySteps(flow)
	if retryableSteps.IsEmpty() {
		return
	}

	now := time.Now()
	for stepID := range retryableSteps {
		exec := flow.Executions[stepID]
		if exec.WorkItems == nil {
			continue
		}

		for token, workItem := range exec.WorkItems {
			retryAt, ok := recoverRetryDeadline(exec, workItem, now)
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

func recoverRetryDeadline(
	exec *api.ExecutionState, workItem *api.WorkState, now time.Time,
) (time.Time, bool) {
	switch workItem.Status {
	case api.WorkActive, api.WorkNotCompleted:
		return now, true
	case api.WorkPending:
		if !workItem.NextRetryAt.IsZero() {
			return workItem.NextRetryAt, true
		}
		if exec.Status == api.StepActive {
			return now, true
		}
		return time.Time{}, false
	case api.WorkFailed:
		if !workItem.NextRetryAt.IsZero() {
			return workItem.NextRetryAt, true
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
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
		started, err = tx.startRetryWorkItem(step, token)
		if err != nil {
			return err
		}
		if len(started) == 0 {
			return nil
		}
		tx.OnSuccess(func(*api.FlowState) {
			tx.handleWorkItemsExecution(
				step, inputs, meta, started,
			)
		})
		return nil
	})
	return err
}

func (e *Engine) scheduleRetryTask(
	fs api.FlowStep, token api.Token, retryAt time.Time,
) {
	e.ScheduleTaskKeyed(retryTaskKey(fs, token),
		func() error {
			err := e.runRetryTask(fs, token)
			if err != nil {
				e.scheduleRetryTask(fs, token,
					time.Now().Add(retryDispatchBackoff),
				)
			}
			return err
		},
		retryAt,
	)
}

func (e *Engine) runRetryTask(fs api.FlowStep, token api.Token) error {
	flow, err := e.GetFlowState(fs.FlowID)
	if err != nil {
		if errors.Is(err, ErrFlowNotFound) {
			return nil
		}
		return err
	}
	if flowTransitions.IsTerminal(flow.Status) {
		return nil
	}

	exec, ok := flow.Executions[fs.StepID]
	if !ok || exec.WorkItems == nil {
		return nil
	}
	if _, ok := exec.WorkItems[token]; !ok {
		return nil
	}

	step, ok := flow.Plan.Steps[fs.StepID]
	if !ok {
		return nil
	}

	if err := e.retryWork(fs, step, token, flow.Metadata); err != nil {
		return err
	}
	return nil
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

func retryTaskKey(fs api.FlowStep, token api.Token) string {
	return fmt.Sprintf("retry/%s/%s/%s", fs.FlowID, fs.StepID, token)
}

func retryTaskPrefix(flowID api.FlowID) string {
	return fmt.Sprintf("retry/%s/", flowID)
}
