package engine

import (
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

const stepDispatchBackoff = 1 * time.Second

func (e *Engine) HandleCommitted(evs ...*timebox.Event) {
	for _, ev := range evs {
		if ev.Raised {
			continue
		}
		e.handleCommitted(ev)
	}
}

func (e *Engine) handleCommitted(ev *timebox.Event) {
	switch api.EventType(ev.Type) {
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
		if !e.canDispatchLocally(fs.StepID) {
			return
		}
		e.scheduleDispatch(fs, e.Now())
	case api.EventTypeRetryScheduled:
		data, err := timebox.GetEventValue[api.RetryScheduledEvent](ev)
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
	default:
		return
	}
}

func (e *Engine) recoverDispatch(flow api.FlowState) {
	steps := e.findDispatchSteps(flow)
	if steps.IsEmpty() {
		return
	}

	now := e.Now()
	for sid := range steps {
		e.scheduleDispatch(api.FlowStep{
			FlowID: flow.ID,
			StepID: sid,
		}, now)
	}
}

func (e *Engine) findDispatchSteps(state api.FlowState) util.Set[api.StepID] {
	steps := util.Set[api.StepID]{}
	now := e.Now()

	for sid, ex := range state.Executions {
		if ex.Status != api.StepActive {
			continue
		}
		step, ok := state.Plan.Steps[sid]
		if !ok {
			continue
		}
		if hasReadyPendingDispatch(step, ex, now) {
			steps.Add(sid)
		}
	}

	return steps
}

func hasReadyPendingDispatch(
	step *api.Step, ex api.ExecutionState, now time.Time,
) bool {
	limit := stepParallelism(step)
	if countActiveWorkItems(ex.WorkItems) >= limit {
		return false
	}

	for _, work := range ex.WorkItems {
		if work.Status != api.WorkPending {
			continue
		}
		if !work.NextRetryAt.IsZero() && work.NextRetryAt.After(now) {
			continue
		}
		return true
	}

	return false
}

func (e *Engine) scheduleDispatch(fs api.FlowStep, at time.Time) {
	e.ScheduleTask(dispatchKey(fs), at, func() error {
		err := e.dispatch(fs)
		if err != nil {
			e.scheduleDispatch(fs, e.Now().Add(stepDispatchBackoff))
		}
		return err
	})
}

func (e *Engine) dispatch(fs api.FlowStep) error {
	var inputs api.Args
	var step *api.Step
	var meta api.Metadata

	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		fl := tx.Value()
		if fl.ID == "" || flowTransitions.IsTerminal(fl.Status) {
			return nil
		}
		if !fl.DeactivatedAt.IsZero() {
			return nil
		}

		ex := fl.Executions[fs.StepID]
		if ex.Status != api.StepActive {
			return nil
		}

		step = fl.Plan.Steps[fs.StepID]

		inputs = ex.Inputs
		meta = fl.Metadata
		if hasReadyPendingDispatch(step, ex, tx.Now()) &&
			!tx.canDispatchLocally(step.ID) {
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

func dispatchKey(fs api.FlowStep) []string {
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
