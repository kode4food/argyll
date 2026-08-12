package engine

import (
	"errors"
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// CompleteCompensation marks a compensation as successfully completed
func (e *Engine) CompleteCompensation(
	fs api.FlowStep, tkn api.Token,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		return tx.completeCompensation(fs.StepID, tkn)
	})
}

// FailCompensation marks a compensation as permanently failed
func (e *Engine) FailCompensation(
	fs api.FlowStep, tkn api.Token, errMsg string,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		return tx.failCompensation(fs.StepID, tkn, errMsg)
	})
}

// NotCompleteCompensation records a transient compensation failure and
// schedules a retry
func (e *Engine) NotCompleteCompensation(
	fs api.FlowStep, tkn api.Token, errMsg string,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		return tx.scheduleCompensationRetry(fs.StepID, tkn, errMsg)
	})
}

// compensateFlow unwinds the flow one wave at a time, in reverse dependency
// order. maybeDeactivate re-enters here after every compensation outcome,
// which drives the next wave
func (tx *flowTx) compensateFlow() error {
	fl := tx.Value()
	if !flowCompensating(fl) || compensationActive(fl) {
		return nil
	}
	wave, err := tx.nextCompensationWave(fl)
	if err != nil {
		return err
	}
	for _, sid := range wave {
		st := fl.Plan.Steps[sid]
		if err := tx.startPendingCompensations(
			st, fl.Executions[sid],
		); err != nil {
			return err
		}
	}
	return nil
}

// nextCompensationWave returns the steps with succeeded work that no step
// still awaiting compensation depends on
func (tx *flowTx) nextCompensationWave(fl api.FlowState) ([]api.StepID, error) {
	pending := util.Set[api.StepID]{}
	for sid, ex := range fl.Executions {
		st, ok := fl.Plan.Steps[sid]
		if !ok || !hasSucceededWork(ex) {
			continue
		}
		comp, err := tx.Engine.steps.Compensator(st)
		if err != nil {
			return nil, err
		}
		if comp == nil {
			continue
		}
		pending.Add(sid)
	}

	var wave []api.StepID
	for sid := range pending {
		seen := util.SetOf(sid)
		if !dependentPending(fl.Plan, sid, pending, seen) {
			wave = append(wave, sid)
		}
	}
	return wave, nil
}

func (tx *flowTx) startPendingCompensations(
	step *api.Step, ex api.ExecutionState,
) error {
	if !hasSucceededWork(ex) {
		return nil
	}

	comp, err := tx.Engine.steps.Compensator(step)
	if err != nil {
		return err
	}
	if comp == nil {
		return nil
	}

	meta := tx.Value().Metadata
	toCompensate := map[api.Token]client.CompensateRequest{}

	for tkn, work := range ex.WorkItems {
		if !policy.WorkSucceeded(work.Status) {
			continue
		}
		if err := tx.raiseCompStarted(step.ID, tkn); err != nil {
			return err
		}
		toCompensate[tkn] = client.CompensateRequest{
			Step:     step,
			Inputs:   ex.Inputs.Apply(work.Inputs),
			Outputs:  work.Outputs,
			Metadata: meta,
		}
	}

	if len(toCompensate) == 0 {
		return nil
	}

	tx.OnSuccess(func(_ api.FlowState, _ []*timebox.Event) {
		for tkn, req := range toCompensate {
			go tx.performCompensation(tkn, req)
		}
	})
	return nil
}

func (tx *flowTx) completeCompensation(
	stepID api.StepID, tkn api.Token,
) error {
	ex := tx.Value().Executions[stepID]
	if !policy.WorkCompActive(ex.WorkItems[tkn].Status) {
		return nil
	}
	if err := tx.raiseCompSucceeded(stepID, tkn); err != nil {
		return err
	}
	return tx.maybeDeactivate()
}

func (tx *flowTx) failCompensation(
	stepID api.StepID, tkn api.Token, errMsg string,
) error {
	ex := tx.Value().Executions[stepID]
	if !policy.WorkCompActive(ex.WorkItems[tkn].Status) {
		return nil
	}
	if err := tx.raiseCompFailed(stepID, tkn, errMsg); err != nil {
		return err
	}
	return tx.maybeDeactivate()
}

func (tx *flowTx) scheduleCompensationRetry(
	stepID api.StepID, tkn api.Token, errMsg string,
) error {
	ex := tx.Value().Executions[stepID]
	work, ok := ex.WorkItems[tkn]
	if !ok || !policy.WorkCompActive(work.Status) {
		return nil
	}

	st := tx.Value().Plan.Steps[stepID]
	if tx.ShouldRetry(st, work) {
		nextRetryAt := tx.calculateNextRetryAt(
			tx.Now(), st.WorkConfig, work.RetryCount,
		)
		err := tx.raiseCompRetryScheduled(raiseCompRetryScheduledArgs{
			stepID:      stepID,
			token:       tkn,
			work:        work,
			errMsg:      errMsg,
			nextRetryAt: nextRetryAt,
		})
		if err != nil {
			return err
		}
		fs := api.FlowStep{FlowID: tx.flowID, StepID: stepID}
		tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
			tx.scheduleCompensationTask(fs, tkn, nextRetryAt)
		})
		return nil
	}

	return tx.failCompensation(stepID, tkn, errMsg)
}

func (tx *flowTx) performCompensation(
	tkn api.Token, req client.CompensateRequest,
) {
	stepID := req.Step.ID
	fs := api.FlowStep{FlowID: tx.flowID, StepID: stepID}
	comp, err := tx.Engine.steps.Compensator(req.Step)
	if err != nil {
		slog.Error("Failed to resolve step compensator",
			log.StepID(stepID),
			log.Error(err))
		return
	}
	if comp == nil {
		return
	}

	err = comp(req)
	if err == nil {
		if recErr := tx.Engine.CompleteCompensation(fs, tkn); recErr != nil {
			slog.Error("Failed to record compensation success",
				log.FlowID(tx.flowID),
				log.StepID(stepID),
				log.Error(recErr))
		}
		return
	}

	if errors.Is(err, api.ErrWorkNotCompleted) {
		if recErr := tx.Engine.NotCompleteCompensation(
			fs, tkn, err.Error(),
		); recErr != nil {
			slog.Error("Failed to record compensation not completed",
				log.FlowID(tx.flowID),
				log.StepID(stepID),
				log.Error(recErr))
		}
		return
	}

	if recErr := tx.Engine.FailCompensation(
		fs, tkn, err.Error(),
	); recErr != nil {
		slog.Error("Failed to record compensation failure",
			log.FlowID(tx.flowID),
			log.StepID(stepID),
			log.Error(recErr))
	}
}

func (e *Engine) scheduleCompensationTask(
	fs api.FlowStep, tkn api.Token, retryAt time.Time,
) {
	e.ScheduleTask(compensateKey(fs, tkn), retryAt, func() error {
		err := e.runCompensationTask(fs, tkn)
		if err != nil {
			e.scheduleCompensationTask(fs, tkn,
				e.Now().Add(localDispatchBackoff))
		}
		return err
	})
}

func (e *Engine) runCompensationTask(fs api.FlowStep, tkn api.Token) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		fl := tx.Value()
		if fl.ID == "" {
			return nil
		}

		ex := fl.Executions[fs.StepID]
		work, ok := ex.WorkItems[tkn]
		if !ok || !policy.WorkCompActive(work.Status) {
			return nil
		}
		if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(tx.Now()) {
			tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
				tx.scheduleCompensationTask(fs, tkn, work.NextRetryAt)
			})
			return nil
		}

		st := fl.Plan.Steps[fs.StepID]
		if !e.canDispatchLocally(st.ID) {
			return tx.raiseDispatchDeferred(fs.StepID)
		}

		// Raise CompStarted to clear NextRetryAt (self-transition)
		if err := tx.raiseCompStarted(fs.StepID, tkn); err != nil {
			return err
		}
		req := client.CompensateRequest{
			Step:     st,
			Inputs:   ex.Inputs.Apply(work.Inputs),
			Outputs:  work.Outputs,
			Metadata: fl.Metadata,
		}

		tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
			go tx.performCompensation(tkn, req)
		})
		return nil
	})
}

func (e *Engine) recoverCompensations(flow api.FlowState) {
	now := e.Now()
	for sid, ex := range flow.Executions {
		st, ok := flow.Plan.Steps[sid]
		if !ok {
			continue
		}
		comp, err := e.steps.Compensator(st)
		if err != nil {
			slog.Error("Failed to resolve step compensator",
				log.StepID(sid),
				log.Error(err))
			continue
		}
		if comp == nil {
			continue
		}
		for tkn, work := range ex.WorkItems {
			if policy.WorkCompActive(work.Status) {
				retryAt := work.NextRetryAt
				if retryAt.IsZero() || retryAt.Before(now) {
					retryAt = now
				}
				e.scheduleCompensationTask(api.FlowStep{
					FlowID: flow.ID,
					StepID: sid,
				}, tkn, retryAt)
			} else if policy.WorkSucceeded(work.Status) &&
				(policy.StepFailed(ex.Status) || flowCompensating(flow)) {
				// Compensation was never started (e.g., engine crashed after
				// step failed but before startPendingCompensations ran)
				e.scheduleCompensationStart(flow.ID, sid, now)
				break // one task per step covers all succeeded items
			}
		}
	}
}

func (e *Engine) scheduleCompensationStart(
	flowID api.FlowID, stepID api.StepID, at time.Time,
) {
	key := []string{"comp-start", string(flowID), string(stepID)}
	e.ScheduleTask(key, at, func() error {
		return e.flowTx(flowID, func(tx *flowTx) error {
			fl := tx.Value()
			if fl.ID == "" {
				return nil
			}
			if flowCompensating(fl) {
				// Covers the whole flow, so sibling tasks are redundant
				return tx.compensateFlow()
			}
			ex := fl.Executions[stepID]
			if !policy.StepFailed(ex.Status) {
				return nil
			}
			st := fl.Plan.Steps[stepID]
			return tx.startPendingCompensations(st, ex)
		})
	})
}

func (tx *flowTx) raiseCompStarted(stepID api.StepID, tkn api.Token) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeCompStarted,
		api.CompStartedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  tkn,
		},
	)
}

func (tx *flowTx) raiseCompSucceeded(stepID api.StepID, tkn api.Token) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeCompSucceeded,
		api.CompSucceededEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  tkn,
		},
	)
}

func (tx *flowTx) raiseCompFailed(
	stepID api.StepID, tkn api.Token, errMsg string,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeCompFailed,
		api.CompFailedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  tkn,
			Error:  errMsg,
		},
	)
}

type raiseCompRetryScheduledArgs struct {
	stepID      api.StepID
	token       api.Token
	work        api.WorkState
	errMsg      string
	nextRetryAt time.Time
}

func (tx *flowTx) raiseCompRetryScheduled(
	args raiseCompRetryScheduledArgs,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeCompRetryScheduled,
		api.CompRetryScheduledEvent{
			FlowID:      tx.flowID,
			StepID:      args.stepID,
			Token:       args.token,
			RetryCount:  args.work.RetryCount + 1,
			NextRetryAt: args.nextRetryAt,
			Error:       args.errMsg,
		},
	)
}

func flowCompensating(flow api.FlowState) bool {
	return flow.Status == api.FlowFailed && flow.Compensate
}

func compensationActive(flow api.FlowState) bool {
	for _, ex := range flow.Executions {
		for _, work := range ex.WorkItems {
			if policy.WorkCompActive(work.Status) {
				return true
			}
		}
	}
	return false
}

func hasSucceededWork(ex api.ExecutionState) bool {
	for _, work := range ex.WorkItems {
		if policy.WorkSucceeded(work.Status) {
			return true
		}
	}
	return false
}

// dependentPending walks consumers transitively, since a direct dependent
// with no compensator can still sit upstream of one that has
func dependentPending(
	pl *api.ExecutionPlan, sid api.StepID,
	pending, seen util.Set[api.StepID],
) bool {
	for _, dep := range dependents(pl, sid) {
		if seen.Contains(dep) {
			continue
		}
		seen.Add(dep)
		if pending.Contains(dep) ||
			dependentPending(pl, dep, pending, seen) {
			return true
		}
	}
	return false
}

func dependents(pl *api.ExecutionPlan, sid api.StepID) []api.StepID {
	st, ok := pl.Steps[sid]
	if !ok {
		return nil
	}
	var res []api.StepID
	for name, attr := range st.Attributes {
		if !attr.IsOutput() {
			continue
		}
		if deps, ok := pl.Attributes[name]; ok {
			res = append(res, deps.Consumers...)
		}
	}
	return res
}

func compensateKey(fs api.FlowStep, tkn api.Token) []string {
	return []string{
		"comp", string(fs.FlowID), string(fs.StepID), string(tkn),
	}
}
