package engine

import (
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

const localDispatchBackoff = 1 * time.Second

func (e *Engine) HandleCommitted(evs ...*timebox.Event) {
	for _, ev := range evs {
		e.handleCommitted(ev)
	}
}

func (e *Engine) handleCommitted(ev *timebox.Event) {
	switch api.EventType(ev.Type) {
	case api.EventTypeStepStarted:
		data, err := ev.GetValue[api.StepStartedEvent]()
		if err != nil {
			slog.Error("Failed to decode step started event",
				log.Error(err))
			return
		}
		fs := api.FlowStep{
			FlowID: data.FlowID,
			StepID: data.StepID,
		}
		e.scheduleWorkDispatch(fs, e.Now())
	case api.EventTypeDispatchDeferred:
		data, err := ev.GetValue[api.DispatchDeferredEvent]()
		if err != nil {
			slog.Error("Failed to decode dispatch deferred event",
				log.Error(err))
			return
		}
		//lint:ignore S1016 Keep the explicit literal to avoid coupling structs
		fs := api.FlowStep{
			FlowID: data.FlowID,
			StepID: data.StepID,
		}
		now := e.Now()
		e.scheduleWorkDispatch(fs, now)
		e.scheduleDispatchRecovery(fs, now)
	case api.EventTypeWorkRetryScheduled:
		data, err := ev.GetValue[api.WorkRetryScheduledEvent]()
		if err != nil {
			slog.Error("Failed to decode retry scheduled event",
				log.Error(err))
			return
		}
		fs := api.FlowStep{
			FlowID: data.FlowID,
			StepID: data.StepID,
		}
		if !e.canDispatchLocally(fs.StepID) {
			return
		}
		e.scheduleRetryTask(fs, data.Token, data.NextRetryAt)
	case api.EventTypeCompRetryScheduled:
		data, err := ev.GetValue[api.CompRetryScheduledEvent]()
		if err != nil {
			slog.Error("Failed to decode comp retry scheduled event",
				log.Error(err))
			return
		}
		fs := api.FlowStep{
			FlowID: data.FlowID,
			StepID: data.StepID,
		}
		if !e.canDispatchLocally(fs.StepID) {
			return
		}
		e.scheduleCompensationTask(fs, data.Token, data.NextRetryAt)
	default:
		return
	}
}

func (e *Engine) recoverWorkDispatch(fl api.FlowState) {
	steps := e.findWorkDispatchSteps(fl)
	if steps.IsEmpty() {
		return
	}

	now := e.Now()
	for sid := range steps {
		e.scheduleWorkDispatch(api.FlowStep{
			FlowID: fl.ID,
			StepID: sid,
		}, now)
	}
}

func (e *Engine) findWorkDispatchSteps(fl api.FlowState) util.Set[api.StepID] {
	steps := util.Set[api.StepID]{}
	now := e.Now()

	for sid, ex := range fl.Executions {
		if !policy.StepActive(ex.Status) {
			continue
		}
		st, ok := fl.Plan.Steps[sid]
		if !ok {
			continue
		}
		if hasReadyPendingWork(st, ex, now) {
			steps.Add(sid)
		}
	}

	return steps
}

func (e *Engine) scheduleWorkDispatch(fs api.FlowStep, at time.Time) {
	e.ScheduleTask(workDispatchKey(fs), at, func() error {
		err := e.dispatchWork(fs)
		if err != nil {
			e.scheduleWorkDispatch(fs, e.Now().Add(localDispatchBackoff))
		}
		return err
	})
}

func (e *Engine) dispatchWork(fs api.FlowStep) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		fl := tx.Value()
		if fl.ID == "" || policy.FlowTerminal(fl.Status) {
			return nil
		}
		if !fl.DeactivatedAt.IsZero() {
			return nil
		}

		ex := fl.Executions[fs.StepID]
		if !policy.StepActive(ex.Status) {
			return nil
		}

		st := fl.Plan.Steps[fs.StepID]
		inputs := ex.Inputs
		meta := fl.Metadata

		if hasReadyPendingWork(st, ex, tx.Now()) &&
			!tx.canDispatchLocally(st.ID) {
			tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
				tx.scheduleWorkDispatch(fs, tx.Now().Add(localDispatchBackoff))
			})
			return nil
		}

		started, err := tx.startPendingWork(st)
		if err != nil {
			return err
		}
		if len(started) == 0 {
			return nil
		}

		tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
			tx.executeStartedWork(st, inputs, meta, started)
		})
		return nil
	})
}

func (e *Engine) scheduleDispatchRecovery(fs api.FlowStep, at time.Time) {
	key := []string{"dispatch-recovery", string(fs.FlowID), string(fs.StepID)}
	e.ScheduleTask(key, at, func() error {
		return e.runDispatchRecovery(fs)
	})
}

func (e *Engine) runDispatchRecovery(fs api.FlowStep) error {
	fl, err := e.GetFlowState(fs.FlowID)
	if err != nil || fl.ID == "" {
		return err
	}

	now := e.Now()
	ex := fl.Executions[fs.StepID]
	st, ok := fl.Plan.Steps[fs.StepID]
	if !ok {
		return nil
	}

	comp, err := e.steps.Compensator(st)
	if err != nil {
		return err
	}
	for tkn, work := range ex.WorkItems {
		if comp != nil && policy.WorkCompActive(work.Status) {
			retryAt := work.NextRetryAt
			if retryAt.IsZero() || retryAt.Before(now) {
				retryAt = now
			}
			e.scheduleCompensationTask(fs, tkn, retryAt)
		}
		if retryAt, ok := policy.RecoverableDeadline(ex, work, now); ok {
			e.scheduleRetryTask(fs, tkn, retryAt)
		}
	}

	return nil
}

func (tx *flowTx) raiseDispatchDeferred(sid api.StepID) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeDispatchDeferred,
		api.DispatchDeferredEvent{
			FlowID: tx.flowID,
			StepID: sid,
		},
	)
}

func hasReadyPendingWork(
	st *api.Step, ex api.ExecutionState, when time.Time,
) bool {
	limit := policy.StepParallelism(st)
	if policy.CountActiveWorkItems(ex.WorkItems) >= limit {
		return false
	}

	for _, work := range ex.WorkItems {
		if !policy.WorkPending(work.Status) {
			continue
		}
		if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(when) {
			continue
		}
		return true
	}

	return false
}

func workDispatchKey(fs api.FlowStep) []string {
	return []string{"dispatch", string(fs.FlowID), string(fs.StepID)}
}
