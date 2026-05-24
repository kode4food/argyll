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
		data, err := timebox.GetEventValue[api.StepStartedEvent](ev)
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
		data, err := timebox.GetEventValue[api.DispatchDeferredEvent](ev)
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
		data, err := timebox.GetEventValue[api.WorkRetryScheduledEvent](ev)
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
		data, err := timebox.GetEventValue[api.CompRetryScheduledEvent](ev)
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

func (e *Engine) recoverWorkDispatch(flow api.FlowState) {
	steps := e.findWorkDispatchSteps(flow)
	if steps.IsEmpty() {
		return
	}

	now := e.Now()
	for sid := range steps {
		e.scheduleWorkDispatch(api.FlowStep{
			FlowID: flow.ID,
			StepID: sid,
		}, now)
	}
}

func (e *Engine) findWorkDispatchSteps(state api.FlowState) util.Set[api.StepID] {
	steps := util.Set[api.StepID]{}
	now := e.Now()

	for sid, ex := range state.Executions {
		if !policy.StepActive(ex.Status) {
			continue
		}
		step, ok := state.Plan.Steps[sid]
		if !ok {
			continue
		}
		if hasReadyPendingWork(step, ex, now) {
			steps.Add(sid)
		}
	}

	return steps
}

func hasReadyPendingWork(
	step *api.Step, ex api.ExecutionState, when time.Time,
) bool {
	limit := policy.StepParallelism(step)
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

		step := fl.Plan.Steps[fs.StepID]
		inputs := ex.Inputs
		meta := fl.Metadata

		if hasReadyPendingWork(step, ex, tx.Now()) &&
			!tx.canDispatchLocally(step.ID) {
			tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
				tx.scheduleWorkDispatch(fs, tx.Now().Add(localDispatchBackoff))
			})
			return nil
		}

		started, err := tx.startPendingWork(step)
		if err != nil {
			return err
		}
		if len(started) == 0 {
			return nil
		}

		tx.OnSuccess(func(flow api.FlowState, _ []*timebox.Event) {
			tx.executeStartedWork(step, inputs, meta, started)
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
	step, ok := fl.Plan.Steps[fs.StepID]
	if !ok {
		return nil
	}

	canCompensate := policy.StepCanCompensate(step)
	for tkn, work := range ex.WorkItems {
		if canCompensate && policy.WorkCompActive(work.Status) {
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

func workDispatchKey(fs api.FlowStep) []string {
	return []string{"dispatch", string(fs.FlowID), string(fs.StepID)}
}

func (tx *flowTx) raiseDispatchDeferred(stepID api.StepID) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeDispatchDeferred,
		api.DispatchDeferredEvent{
			FlowID: tx.flowID,
			StepID: stepID,
		},
	)
}
