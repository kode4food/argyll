package engine

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/step"
)

type (
	// ExecContext holds the context for a single step execution
	ExecContext struct {
		engine *Engine
		step   *api.Step
		inputs api.Args
		meta   api.Metadata
		flowID api.FlowID
		stepID api.StepID
	}

	// MultiArgs maps attribute names to value arrays for parallel execution
	MultiArgs map[api.Name][]any
)

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

func (e *ExecContext) UpdateHealth(s api.HealthStatus, msg string) error {
	return e.engine.UpdateStepHealth(e.stepID, s, msg)
}

func (tx *flowTx) executeStartedWork(
	st *api.Step, inputs api.Args, meta api.Metadata, items api.WorkItems,
) {
	execCtx := &ExecContext{
		engine: tx.Engine,
		flowID: tx.flowID,
		stepID: st.ID,
		step:   st,
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
		settleFailure(workRuntime{e}, tkn, err)
	}
}

func (e *ExecContext) flowStep() api.FlowStep {
	return api.FlowStep{FlowID: e.flowID, StepID: e.stepID}
}

func (e *ExecContext) performWork(inputs api.Args, tkn api.Token) error {
	handler, err := e.engine.steps.Lookup(e.step.Type)
	if err != nil {
		return errors.Join(ErrUnsupportedStepType, err)
	}
	return handler.Invoke(workRuntime{e}, e.step, inputs, tkn)
}

func (tx *flowTx) startPendingWork(st *api.Step) (api.WorkItems, error) {
	sid := st.ID
	ex := tx.Value().Executions[sid]
	if !policy.StepActive(ex.Status) {
		return nil, fmt.Errorf("%w: expected %s to be active, got %s",
			ErrInvariantViolated, sid, ex.Status)
	}

	limit := policy.StepParallelism(st)
	now := tx.Now()
	started := api.WorkItems{}
	canDispatch := tx.canDispatchLocally(st.ID)
	for _, tkn := range slices.Sorted(maps.Keys(ex.WorkItems)) {
		ex = tx.Value().Executions[sid]
		if !policy.StepActive(ex.Status) ||
			policy.CountActiveWorkItems(ex.WorkItems) >= limit {
			break
		}
		work := ex.WorkItems[tkn]
		shouldStart, err := tx.shouldStartPendingWorkItem(
			st, ex.Inputs, work, now,
		)
		if err != nil {
			return nil, err
		}
		if !shouldStart {
			continue
		}

		inputs := ex.Inputs.Apply(work.Inputs)
		if st.DefaultedHandling() == api.HandlingMemoized {
			if cached, ok := tx.memoCache.Get(st, inputs); ok {
				return started, tx.handleMemoCacheHit(sid, tkn, cached)
			}
		}
		if !canDispatch {
			continue
		}

		if err := tx.raiseWorkStarted(sid, tkn, inputs); err != nil {
			return nil, err
		}
		started[tkn] = tx.Value().Executions[sid].WorkItems[tkn]
	}

	ex = tx.Value().Executions[sid]
	if !canDispatch && policy.WorkReadyToDispatch(st, &ex, now) {
		if err := tx.raiseDispatchDeferred(sid); err != nil {
			return nil, err
		}
		err := tx.scheduleWorkDispatch(sid, now.Add(localDispatchBackoff))
		if err != nil {
			return nil, err
		}
	}
	return started, nil
}

func (tx *flowTx) startRetryWorkItem(
	st *api.Step, tkn api.Token,
) (api.WorkItems, time.Time, error) {
	sid := st.ID
	ex := tx.Value().Executions[sid]
	if !policy.StepActive(ex.Status) {
		return nil, time.Time{}, nil
	}

	work, ok := ex.WorkItems[tkn]
	if !ok {
		return nil, time.Time{}, nil
	}

	now := tx.Now()
	action, nextAt := policy.RetryStartDecision(&work, now)
	if action == policy.RetryStartWait {
		return nil, nextAt, nil
	}
	if action != policy.RetryStartCheckPending {
		return nil, time.Time{}, nil
	}

	shouldStart, err := tx.shouldStartRetryPending(
		shouldStartRetryPendingArgs{
			step:  st,
			base:  ex.Inputs,
			work:  work,
			items: ex.WorkItems,
			when:  now,
		},
	)
	if err != nil || !shouldStart {
		return nil, time.Time{}, err
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
	st *api.Step, base api.Args, work api.WorkState, when time.Time,
) (bool, error) {
	sid := st.ID
	if !policy.WorkPending(work.Status) {
		return false, nil
	}
	if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(when) {
		return false, nil
	}
	inputs := base.Apply(work.Inputs)
	shouldStart, err := tx.evaluateStepPredicate(st, inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(sid, base, err)
	}
	return shouldStart, nil
}

type shouldStartRetryPendingArgs struct {
	step  *api.Step
	base  api.Args
	work  api.WorkState
	items api.WorkItems
	when  time.Time
}

func (tx *flowTx) shouldStartRetryPending(
	args shouldStartRetryPendingArgs,
) (bool, error) {
	sid := args.step.ID
	if !args.work.NextRetryAt.IsZero() &&
		args.work.NextRetryAt.After(args.when) {
		return false, nil
	}
	limit := policy.StepParallelism(args.step)
	active := policy.CountActiveWorkItems(args.items)
	if active >= limit {
		return false, nil
	}
	inputs := args.base.Apply(args.work.Inputs)
	shouldStart, err := tx.evaluateStepPredicate(args.step, inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(sid, args.base, err)
	}
	return shouldStart, nil
}

func (tx *flowTx) raiseWorkStarted(
	sid api.StepID, tkn api.Token, inputs api.Args,
) error {
	if err := tx.checkWorkTransition(sid, tkn, api.WorkActive); err != nil {
		return err
	}
	if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkStarted,
		api.WorkStartedEvent{
			FlowID: tx.flowID,
			StepID: sid,
			Token:  tkn,
			Inputs: inputs,
		},
	); err != nil {
		return err
	}
	st := tx.Value().Plan.Steps[sid]
	if h, err := tx.steps.Lookup(st.Type); err == nil && h.Inline {
		return tx.invokeInline(h, st, tkn, inputs)
	}
	fl := tx.Value()
	if at, ok := tx.workDeadline(&fl, sid, tkn); ok {
		if err := tx.scheduleWorkDeadlineAt(sid, tkn, at); err != nil {
			return err
		}
	}
	if st.Type == api.StepTypeFlow {
		return tx.startChildFlow(sid, tkn, inputs)
	}
	return nil
}

func (tx *flowTx) invokeInline(
	h *step.Handler, st *api.Step, tkn api.Token, inputs api.Args,
) error {
	rt := &txRuntime{tx: tx, sid: st.ID}
	if err := h.Invoke(rt, st, inputs, tkn); err != nil && rt.err == nil {
		settleFailure(rt, tkn, err)
	}
	return rt.err
}
