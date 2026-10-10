package engine

import (
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// compRuntime is the Runtime a compensator receives, settling compensation
	compRuntime struct {
		*ExecContext
	}

	compensationWaveWalk struct {
		pending util.Set[api.StepID]
		seen    util.Set[api.StepID]
	}
)

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

// compensateFlow unwinds one wave in reverse dependency order, re-entered by
// maybeDeactivate after every outcome to drive the next
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
		if !ok || !hasSucceededWork(ex) || !flowRollsBack(fl, sid) {
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

	toCompensate := map[api.Token]api.Args{}

	for tkn, work := range ex.WorkItems {
		if !policy.WorkSucceeded(work.Status) {
			continue
		}
		if err := tx.raiseCompStarted(st.ID, tkn); err != nil {
			return err
		}
		toCompensate[tkn] = compensationArgs(st, ex, work)
	}

	if len(toCompensate) == 0 {
		return nil
	}

	exec := tx.compensationContext(st)
	tx.OnSuccess(func(_ api.FlowState, _ []*timebox.Event) {
		for tkn, args := range toCompensate {
			go exec.performCompensation(args, tkn)
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
	if err := tx.clearWorkDeadline(sid, tkn); err != nil {
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
	if err := tx.clearWorkDeadline(sid, tkn); err != nil {
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
		return tx.scheduleCompensationTask(sid, tkn, nextRetryAt)
	}

	return tx.failCompensation(sid, tkn, errMsg)
}

// CompleteWork records compensation success, so a compensator settles the same
// way an invocation does
func (r compRuntime) CompleteWork(tkn api.Token, _ api.Args) error {
	return r.engine.CompleteCompensation(r.flowStep(), tkn)
}

func (r compRuntime) notCompleteWork(tkn api.Token, errMsg string) error {
	return r.engine.NotCompleteCompensation(r.flowStep(), tkn, errMsg)
}

func (r compRuntime) failWork(tkn api.Token, errMsg string) error {
	return r.engine.FailCompensation(r.flowStep(), tkn, errMsg)
}

func (e *ExecContext) performCompensation(args api.Args, tkn api.Token) {
	comp, err := e.engine.steps.Compensator(e.step)
	if err != nil {
		slog.Error("Failed to resolve step compensator",
			log.StepID(e.stepID),
			log.Error(err))
		return
	}
	if comp == nil {
		return
	}
	rt := compRuntime{e}
	if err := comp(rt, e.step, args, tkn); err != nil {
		settleFailure(rt, tkn, err)
	}
}

func (tx *flowTx) compensationContext(st *api.Step) *ExecContext {
	return &ExecContext{
		engine: tx.Engine,
		step:   st,
		meta:   tx.Value().Metadata,
		flowID: tx.flowID,
		stepID: st.ID,
	}
}

func (tx *flowTx) scheduleCompensationTask(
	sid api.StepID, tkn api.Token, retryAt time.Time,
) error {
	fs := api.FlowStep{FlowID: tx.flowID, StepID: sid}
	return tx.scheduleEvent(compensateKey(fs, tkn), retryAt,
		compensationRetryReady,
		scheduledWork{StepID: sid, Token: tkn})
}

func (tx *flowTx) handleCompensationRetry(sid api.StepID, tkn api.Token) error {
	fl := tx.Value()
	if fl.ID == "" {
		return nil
	}

	ex := fl.Executions[sid]
	work, ok := ex.WorkItems[tkn]
	if !ok || !policy.WorkCompPending(work.Status) {
		return nil
	}
	if work.NextRetryAt.After(tx.Now()) {
		return tx.scheduleCompensationTask(sid, tkn, work.NextRetryAt)
	}

	st := fl.Plan.Steps[sid]
	if !tx.canDispatchLocally(st.ID) {
		return ErrDispatchUnavailable
	}

	if err := tx.raiseCompStarted(sid, tkn); err != nil {
		return err
	}
	args := compensationArgs(st, ex, work)

	exec := tx.compensationContext(st)
	tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
		go exec.performCompensation(args, tkn)
	})
	return nil
}

func (tx *flowTx) recoverCompensations() error {
	for sid := range tx.Value().Executions {
		st, ok := tx.Value().Plan.Steps[sid]
		if !ok {
			continue
		}
		comp, err := tx.Engine.steps.Compensator(st)
		if err != nil {
			slog.Error("Failed to resolve step compensator",
				log.StepID(sid),
				log.Error(err))
			continue
		}
		if comp == nil {
			continue
		}
		if err := tx.recoverStepCompensations(sid); err != nil {
			return err
		}
	}
	return nil
}

func (tx *flowTx) recoverStepCompensations(sid api.StepID) error {
	fl := tx.Value()
	ex := fl.Executions[sid]
	now := tx.Now()
	for tkn, work := range ex.WorkItems {
		if retryAt, ok := policy.CompRetryAt(work, now); ok {
			if err := tx.scheduleCompensationTask(
				sid, tkn, retryAt,
			); err != nil {
				return err
			}
			continue
		}
		if policy.WorkSucceeded(work.Status) &&
			(policy.StepFailed(ex.Status) || flowRollsBack(fl, sid)) {
			// Compensation never started, so it joins this transaction rather
			// than a task rereading the state
			if flowRollsBack(fl, sid) {
				// Covers the whole flow, so sibling steps are redundant
				return tx.compensateFlow()
			}
			// One pass per step covers all succeeded items
			return tx.startPendingCompensations(fl.Plan.Steps[sid], ex)
		}
	}
	return nil
}

func (tx *flowTx) raiseCompStarted(sid api.StepID, tkn api.Token) error {
	if err := tx.checkWorkTransition(
		sid, tkn, api.WorkCompensating,
	); err != nil {
		return err
	}
	if err := events.Raise(tx.FlowAggregator, api.EventTypeCompStarted,
		api.CompStartedEvent{
			FlowID: tx.flowID,
			StepID: sid,
			Token:  tkn,
		},
	); err != nil {
		return err
	}
	if at, ok := tx.workDeadline(tx.Value(), sid, tkn); ok {
		return tx.scheduleWorkDeadlineAt(sid, tkn, at)
	}
	return nil
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
		if !policy.StepFailed(ex.Status) && !flowRollsBack(fl, sid) {
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

func flowCompensating(fl api.FlowState) bool {
	return fl.Compensate && policy.FlowTerminal(fl.Status)
}

// flowRollsBack reports whether a terminal flow undoes the step's work: every
// step when the flow failed, but only steps the winning goal set does not use
// when it completed
func flowRollsBack(fl api.FlowState, sid api.StepID) bool {
	if !flowCompensating(fl) {
		return false
	}
	if fl.Status == api.FlowFailed {
		return true
	}
	// A completed flow always has its winning goal set active
	return !goalScope(fl.Plan, activeGoals(fl).Steps).Contains(sid)
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

// compensationArgs selects the compensated attributes, from the inputs or the
// outputs they belong to, so a compensator never receives the rest
func compensationArgs(
	st *api.Step, ex api.ExecutionState, work api.WorkState,
) api.Args {
	inputs := ex.Inputs.Apply(work.Inputs)
	res := api.Args{}
	for name, attr := range st.Attributes {
		if attr == nil || !attr.Compensated {
			continue
		}
		mapped, _ := st.MappedName(name)
		src := inputs
		if attr.IsOutput() {
			src = work.Outputs
		}
		if v, ok := src[mapped]; ok {
			res[mapped] = v
		}
	}
	return res
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

func compensateKey(fs api.FlowStep, tkn api.Token) []string {
	return []string{
		string(fs.FlowID), "comp", string(fs.StepID), string(tkn),
	}
}
