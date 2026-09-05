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

type compensationWaveWalk struct {
	pending util.Set[api.StepID]
	seen    util.Set[api.StepID]
}

// CompleteCompensation marks a compensation as successfully completed
func (e *Engine) CompleteCompensation(fs api.FlowStep, tkn api.Token) error {
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
		walk := &compensationWaveWalk{pending: pending, seen: util.SetOf(sid)}
		if !walk.dependentPending(fl.Plan, sid) {
			wave = append(wave, sid)
		}
	}
	return wave, nil
}

func (tx *flowTx) startPendingCompensations(
	st *api.Step, ex api.ExecutionState,
) error {
	if !hasSucceededWork(ex) {
		return nil
	}

	comp, err := tx.Engine.steps.Compensator(st)
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
		if err := tx.raiseCompStarted(st.ID, tkn); err != nil {
			return err
		}
		toCompensate[tkn] = client.CompensateRequest{
			Step:     st,
			Inputs:   ex.Inputs.Apply(work.Inputs),
			Outputs:  work.Outputs,
			Metadata: tx.compensateMetadata(meta, st, tkn),
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

func (tx *flowTx) completeCompensation(sid api.StepID, tkn api.Token) error {
	ex := tx.Value().Executions[sid]
	if !policy.WorkCompUnsettled(ex.WorkItems[tkn].Status) {
		return nil
	}
	if err := tx.raiseCompSucceeded(sid, tkn); err != nil {
		return err
	}
	return tx.maybeDeactivate()
}

func (tx *flowTx) failCompensation(
	sid api.StepID, tkn api.Token, errMsg string,
) error {
	ex := tx.Value().Executions[sid]
	if !policy.WorkCompUnsettled(ex.WorkItems[tkn].Status) {
		return nil
	}
	if err := tx.raiseCompFailed(sid, tkn, errMsg); err != nil {
		return err
	}
	return tx.maybeDeactivate()
}

func (tx *flowTx) scheduleCompensationRetry(
	sid api.StepID, tkn api.Token, errMsg string,
) error {
	ex := tx.Value().Executions[sid]
	work, ok := ex.WorkItems[tkn]
	if !ok || !policy.WorkCompActive(work.Status) {
		return nil
	}

	st := tx.Value().Plan.Steps[sid]
	if tx.ShouldRetry(st, work) {
		nextRetryAt := tx.calculateNextRetryAt(
			tx.Now(), st.WorkConfig, work.RetryCount,
		)
		err := tx.raiseCompRetryScheduled(raiseCompRetryScheduledArgs{
			stepID:      sid,
			token:       tkn,
			work:        work,
			errMsg:      errMsg,
			nextRetryAt: nextRetryAt,
		})
		if err != nil {
			return err
		}
		return nil
	}

	return tx.failCompensation(sid, tkn, errMsg)
}

func (tx *flowTx) performCompensation(
	tkn api.Token, req client.CompensateRequest,
) {
	sid := req.Step.ID
	fs := api.FlowStep{FlowID: tx.flowID, StepID: sid}
	comp, err := tx.Engine.steps.Compensator(req.Step)
	if err != nil {
		slog.Error("Failed to resolve step compensator",
			log.StepID(sid),
			log.Error(err))
		return
	}
	if comp == nil {
		return
	}

	err = comp(req)
	if err == nil {
		if req.Step.HTTP.Compensate.Async() {
			return
		}
		if recErr := tx.Engine.CompleteCompensation(fs, tkn); recErr != nil {
			slog.Error("Failed to record compensation success",
				log.FlowID(tx.flowID),
				log.StepID(sid),
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
				log.StepID(sid),
				log.Error(recErr))
		}
		return
	}

	if recErr := tx.Engine.FailCompensation(
		fs, tkn, err.Error(),
	); recErr != nil {
		slog.Error("Failed to record compensation failure",
			log.FlowID(tx.flowID),
			log.StepID(sid),
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
		if !ok || !policy.WorkCompPending(work.Status) {
			return nil
		}
		if work.NextRetryAt.After(tx.Now()) {
			tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
				tx.scheduleCompensationTask(fs, tkn, work.NextRetryAt)
			})
			return nil
		}

		st := fl.Plan.Steps[fs.StepID]
		if !e.canDispatchLocally(st.ID) {
			return tx.raiseDispatchDeferred(fs.StepID)
		}

		if err := tx.raiseCompStarted(fs.StepID, tkn); err != nil {
			return err
		}
		req := client.CompensateRequest{
			Step:     st,
			Inputs:   ex.Inputs.Apply(work.Inputs),
			Outputs:  work.Outputs,
			Metadata: tx.compensateMetadata(fl.Metadata, st, tkn),
		}

		tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
			go tx.performCompensation(tkn, req)
		})
		return nil
	})
}

func (e *Engine) recoverCompensations(fl api.FlowState) {
	now := e.Now()
	for sid := range fl.Executions {
		st, ok := fl.Plan.Steps[sid]
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
		e.recoverStepCompensations(fl, sid, now)
	}
}

func (e *Engine) recoverStepCompensations(
	fl api.FlowState, sid api.StepID, now time.Time,
) {
	ex := fl.Executions[sid]
	for tkn, work := range ex.WorkItems {
		if retryAt, ok := policy.CompRetryAt(work, now); ok {
			e.scheduleCompensationTask(api.FlowStep{
				FlowID: fl.ID,
				StepID: sid,
			}, tkn, retryAt)
			continue
		}
		if policy.WorkSucceeded(work.Status) &&
			(policy.StepFailed(ex.Status) || flowCompensating(fl)) {
			// Compensation was never started (e.g., engine crashed after
			// step failed but before startPendingCompensations ran)
			e.scheduleCompensationStart(fl.ID, sid, now)
			return // one task per step covers all succeeded items
		}
	}
}

func (e *Engine) scheduleCompensationStart(
	fid api.FlowID, sid api.StepID, at time.Time,
) {
	key := compStartKey(fid, sid)
	e.ScheduleTask(key, at, func() error {
		return e.flowTx(fid, func(tx *flowTx) error {
			fl := tx.Value()
			if fl.ID == "" {
				return nil
			}
			if flowCompensating(fl) {
				// Covers the whole flow, so sibling tasks are redundant
				return tx.compensateFlow()
			}
			ex := fl.Executions[sid]
			if !policy.StepFailed(ex.Status) {
				return nil
			}
			st := fl.Plan.Steps[sid]
			return tx.startPendingCompensations(st, ex)
		})
	})
}

func (tx *flowTx) raiseCompStarted(sid api.StepID, tkn api.Token) error {
	if err := tx.checkWorkTransition(
		sid, tkn, api.WorkCompensating,
	); err != nil {
		return err
	}
	return events.Raise(tx.FlowAggregator, api.EventTypeCompStarted,
		api.CompStartedEvent{
			FlowID: tx.flowID,
			StepID: sid,
			Token:  tkn,
		},
	)
}

func (tx *flowTx) raiseCompSucceeded(sid api.StepID, tkn api.Token) error {
	if err := tx.checkWorkTransition(
		sid, tkn, api.WorkCompensated,
	); err != nil {
		return err
	}
	return events.Raise(tx.FlowAggregator, api.EventTypeCompSucceeded,
		api.CompSucceededEvent{
			FlowID: tx.flowID,
			StepID: sid,
			Token:  tkn,
		},
	)
}

func (tx *flowTx) raiseCompFailed(
	sid api.StepID, tkn api.Token, errMsg string,
) error {
	if err := tx.checkWorkTransition(
		sid, tkn, api.WorkCompFailed,
	); err != nil {
		return err
	}
	return events.Raise(tx.FlowAggregator, api.EventTypeCompFailed,
		api.CompFailedEvent{
			FlowID: tx.flowID,
			StepID: sid,
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
	if err := tx.checkWorkTransition(
		args.stepID, args.token, api.WorkCompPending,
	); err != nil {
		return err
	}
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

// compensationPending reports whether succeeded work still owes an unstarted
// compensation, which a terminal flow must not deactivate out from under
func (e *Engine) compensationPending(fl api.FlowState) bool {
	for sid, ex := range fl.Executions {
		st, ok := fl.Plan.Steps[sid]
		if !ok || !hasSucceededWork(ex) {
			continue
		}
		if !policy.StepFailed(ex.Status) && !flowCompensating(fl) {
			continue
		}
		if comp, err := e.steps.Compensator(st); err == nil && comp != nil {
			return true
		}
	}
	return false
}

// dependentPending walks consumers transitively, since a direct dependent
// with no compensator can still sit upstream of one that has
func (w *compensationWaveWalk) dependentPending(
	pl *api.ExecutionPlan, sid api.StepID,
) bool {
	for _, dep := range dependents(pl, sid) {
		if w.seen.Contains(dep) {
			continue
		}
		w.seen.Add(dep)
		if w.pending.Contains(dep) || w.dependentPending(pl, dep) {
			return true
		}
	}
	return false
}

func (tx *flowTx) compensateMetadata(
	meta api.Metadata, st *api.Step, tkn api.Token,
) api.Metadata {
	res := meta.Apply(api.Metadata{
		api.MetaFlowID:       tx.flowID,
		api.MetaStepID:       st.ID,
		api.MetaReceiptToken: tkn,
	})
	if st.HTTP != nil && st.HTTP.Compensate.Async() {
		res[api.MetaWebhookURL] = tx.Engine.compensateCallbackURL(
			tx.flowID, st.ID, tkn,
		)
	}
	return res
}

func flowCompensating(fl api.FlowState) bool {
	return fl.Status == api.FlowFailed && fl.Compensate
}

func compensationActive(fl api.FlowState) bool {
	for _, ex := range fl.Executions {
		for _, work := range ex.WorkItems {
			if policy.WorkCompUnsettled(work.Status) {
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

func compStartKey(fid api.FlowID, sid api.StepID) []string {
	return []string{string(fid), "comp-start", string(sid)}
}

func compensateKey(fs api.FlowStep, tkn api.Token) []string {
	return []string{
		string(fs.FlowID), "comp", string(fs.StepID), string(tkn),
	}
}
