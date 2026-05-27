package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/internal/engine/step"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// ExecContext holds the context for a single step execution
	ExecContext struct {
		engine *Engine
		step   *api.Step
		child  *api.ExecutionPlan
		inputs api.Args
		meta   api.Metadata
		flowID api.FlowID
		stepID api.StepID
	}

	// MultiArgs maps attribute names to value arrays for parallel execution
	MultiArgs map[api.Name][]any
)

var _ step.Runtime = (*ExecContext)(nil)

var (
	ErrStepAlreadyPending     = errors.New("step not pending")
	ErrUnsupportedStepType    = errors.New("unsupported step type")
	ErrPredicateCompileFailed = errors.New("predicate compilation failed")
	ErrScriptEnvFailed        = errors.New("failed to get script environment")
	ErrPredicateEvalFailed    = errors.New("predicate evaluation failed")
	ErrMatchEvalFailed        = errors.New("match evaluation failed")
)

func (e *ExecContext) FlowID() api.FlowID {
	return e.flowID
}

func (e *ExecContext) StepID() api.StepID {
	return e.stepID
}

func (e *ExecContext) Metadata() api.Metadata {
	return e.meta
}

func (e *ExecContext) WebhookURL(tkn api.Token) string {
	return e.engine.config.WebhookBaseURL + "/webhook/" +
		string(e.flowID) + "/" + string(e.stepID) + "/" + string(tkn)
}

func (e *ExecContext) CompleteWork(tkn api.Token, outputs api.Args) error {
	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}
	return e.engine.CompleteWork(fs, tkn, outputs)
}

func (e *ExecContext) StartChildFlow(
	tkn api.Token, init api.InitArgs,
) (api.FlowID, error) {
	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}
	return e.engine.StartChildFlow(fs, tkn, e.child, init, e.meta)
}

func (e *ExecContext) UpdateHealth(s api.HealthStatus, msg string) error {
	return e.engine.UpdateStepHealth(e.stepID, s, msg)
}

func (tx *flowTx) executeStartedWork(
	step *api.Step, inputs api.Args, meta api.Metadata, items api.WorkItems,
) {
	execCtx := &ExecContext{
		engine: tx.Engine,
		flowID: tx.flowID,
		stepID: step.ID,
		step:   step,
		child:  tx.Value().Plan.Children[step.ID],
		inputs: inputs,
		meta:   meta,
	}
	execCtx.executeWorkItems(items)
}

func (e *ExecContext) executeWorkItems(items api.WorkItems) {
	for tkn, work := range items {
		if !policy.WorkActive(work.Status) {
			continue
		}

		go e.performWorkItem(tkn, work)
	}
}

func (e *ExecContext) performWorkItem(tkn api.Token, work api.WorkState) {
	inputs := e.inputs.Apply(work.Inputs)
	if err := e.performWork(inputs, tkn); err != nil {
		e.handleWorkItemFailure(tkn, err)
	}
}

func (e *ExecContext) handleWorkItemFailure(tkn api.Token, err error) {
	fs := api.FlowStep{FlowID: e.flowID, StepID: e.stepID}

	if errors.Is(err, ErrInvalidWorkTransition) {
		return
	}

	if errors.Is(err, api.ErrWorkNotCompleted) {
		recErr := e.engine.NotCompleteWork(fs, tkn, err.Error())
		if recErr != nil {
			slog.Error("Failed to record work not completed",
				log.FlowID(e.flowID),
				log.StepID(e.stepID),
				log.Error(recErr))
		}
		return
	}

	recErr := e.engine.FailWork(fs, tkn, err.Error())
	if recErr != nil {
		slog.Error("Failed to record work failure",
			log.FlowID(e.flowID),
			log.StepID(e.stepID),
			log.Error(recErr))
	}
}

func (e *ExecContext) performWork(inputs api.Args, tkn api.Token) error {
	handler, err := e.engine.steps.Lookup(e.step.Type)
	if err != nil {
		return errors.Join(ErrUnsupportedStepType, err)
	}
	return handler.Execute(e, e.step, inputs, tkn)
}

func (tx *flowTx) startPendingWork(step *api.Step) (api.WorkItems, error) {
	sid := step.ID
	ex := tx.Value().Executions[sid]
	if !policy.StepActive(ex.Status) {
		return nil, fmt.Errorf("%w: expected %s to be active, got %s",
			ErrInvariantViolated, sid, ex.Status)
	}

	limit := policy.StepParallelism(step)
	active := policy.CountActiveWorkItems(ex.WorkItems)
	remaining := limit - active
	if remaining <= 0 {
		return nil, nil
	}

	now := tx.Now()
	started := api.WorkItems{}
	canDispatch := tx.canDispatchLocally(step.ID)
	for tkn, work := range ex.WorkItems {
		if remaining == 0 {
			break
		}
		shouldStart, err := tx.shouldStartPendingWorkItem(
			step, ex.Inputs, work, now,
		)
		if err != nil {
			return nil, err
		}
		if !shouldStart {
			continue
		}

		inputs := ex.Inputs.Apply(work.Inputs)
		if step.Memoizable {
			if cached, ok := tx.memoCache.Get(step, inputs); ok {
				err := tx.handleMemoCacheHit(sid, tkn, cached)
				if err != nil {
					return nil, err
				}
				remaining--
				continue
			}
		}
		if !canDispatch {
			continue
		}

		if err := tx.raiseWorkStarted(sid, tkn, inputs); err != nil {
			return nil, err
		}
		ex = tx.Value().Executions[sid]
		started[tkn] = ex.WorkItems[tkn]
		remaining--
	}

	return started, nil
}

func (tx *flowTx) startRetryWorkItem(
	step *api.Step, tkn api.Token,
) (api.WorkItems, time.Time, error) {
	sid := step.ID
	ex := tx.Value().Executions[sid]
	if !policy.StepActive(ex.Status) {
		return nil, time.Time{}, nil
	}

	work, ok := ex.WorkItems[tkn]
	if !ok {
		return nil, time.Time{}, nil
	}

	now := tx.Now()
	shouldStart := false
	action, nextAt := policy.RetryStartDecision(work, now)
	switch action {
	case policy.RetryStartWait:
		return nil, nextAt, nil
	case policy.RetryStartCheckPending:
		var err error
		if shouldStart, err = tx.shouldStartRetryPending(
			step, ex.Inputs, work, ex.WorkItems, now,
		); err != nil {
			return nil, time.Time{}, err
		}
	case policy.RetryStartNow:
		shouldStart = true
	default:
		return nil, time.Time{}, nil
	}
	if !shouldStart {
		return nil, time.Time{}, nil
	}

	inputs := ex.Inputs.Apply(work.Inputs)
	if err := tx.raiseWorkStarted(sid, tkn, inputs); err != nil {
		return nil, time.Time{}, err
	}
	ex = tx.Value().Executions[sid]
	started := api.WorkItems{}
	started[tkn] = ex.WorkItems[tkn]
	return started, time.Time{}, nil
}

func (tx *flowTx) shouldStartPendingWorkItem(
	step *api.Step, base api.Args, work api.WorkState, when time.Time,
) (bool, error) {
	sid := step.ID
	if !policy.WorkPending(work.Status) {
		return false, nil
	}
	if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(when) {
		return false, nil
	}
	inputs := base.Apply(work.Inputs)
	shouldStart, err := tx.evaluateStepPredicate(step, inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(sid, base, err)
	}
	return shouldStart, nil
}

func (tx *flowTx) shouldStartRetryPending(
	step *api.Step, base api.Args, work api.WorkState, items api.WorkItems,
	when time.Time,
) (bool, error) {
	sid := step.ID
	if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(when) {
		return false, nil
	}
	limit := policy.StepParallelism(step)
	active := policy.CountActiveWorkItems(items)
	if active >= limit {
		return false, nil
	}
	inputs := base.Apply(work.Inputs)
	shouldStart, err := tx.evaluateStepPredicate(step, inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(sid, base, err)
	}
	return shouldStart, nil
}

func (tx *flowTx) raiseWorkStarted(
	stepID api.StepID, tkn api.Token, inputs api.Args,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkStarted,
		api.WorkStartedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  tkn,
			Inputs: inputs,
		},
	)
}
