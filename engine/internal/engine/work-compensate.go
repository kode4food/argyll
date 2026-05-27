package engine

import (
	"errors"
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/internal/engine/step"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
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

func (e *Engine) compensator(st *api.Step) (step.CompensateFunc, error) {
	comp, err := e.steps.Compensator(st)
	if err != nil {
		return nil, err
	}
	if st.Memoizable {
		return nil, nil
	}
	return comp, nil
}

func (tx *flowTx) startPendingCompensations(
	step *api.Step, ex api.ExecutionState,
) error {
	hasSucceeded := false
	for _, work := range ex.WorkItems {
		if policy.WorkSucceeded(work.Status) {
			hasSucceeded = true
			break
		}
	}
	if !hasSucceeded {
		return nil
	}

	comp, err := tx.Engine.compensator(step)
	if err != nil {
		return err
	}
	if comp == nil {
		return nil
	}

	type pending struct {
		tkn     api.Token
		inputs  api.Args
		outputs api.Args
	}
	var toCompensate []pending

	for tkn, work := range ex.WorkItems {
		if !policy.WorkSucceeded(work.Status) {
			continue
		}
		if err := tx.raiseCompStarted(step.ID, tkn); err != nil {
			return err
		}
		toCompensate = append(toCompensate, pending{
			tkn:     tkn,
			inputs:  ex.Inputs.Apply(work.Inputs),
			outputs: work.Outputs,
		})
	}

	if len(toCompensate) == 0 {
		return nil
	}

	meta := tx.Value().Metadata
	tx.OnSuccess(func(_ api.FlowState, _ []*timebox.Event) {
		for _, p := range toCompensate {
			go tx.performCompensation(step, p.inputs, p.outputs, p.tkn, meta)
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
		err := tx.raiseCompRetryScheduled(
			stepID, tkn, work, errMsg, nextRetryAt,
		)
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
	step *api.Step, inputs api.Args, outputs api.Args,
	tkn api.Token, meta api.Metadata,
) {
	fs := api.FlowStep{FlowID: tx.flowID, StepID: step.ID}
	comp, err := tx.Engine.compensator(step)
	if err != nil {
		slog.Error("Failed to resolve step compensator",
			log.StepID(step.ID),
			log.Error(err))
		return
	}
	if comp == nil {
		return
	}
	err = comp(step, inputs, outputs, meta)
	if err == nil {
		if recErr := tx.Engine.CompleteCompensation(fs, tkn); recErr != nil {
			slog.Error("Failed to record compensation success",
				log.FlowID(tx.flowID),
				log.StepID(step.ID),
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
				log.StepID(step.ID),
				log.Error(recErr))
		}
		return
	}

	if recErr := tx.Engine.FailCompensation(
		fs, tkn, err.Error(),
	); recErr != nil {
		slog.Error("Failed to record compensation failure",
			log.FlowID(tx.flowID),
			log.StepID(step.ID),
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
	var st *api.Step
	var inputs api.Args
	var outputs api.Args
	var meta api.Metadata

	err := e.flowTx(fs.FlowID, func(tx *flowTx) error {
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

		st = fl.Plan.Steps[fs.StepID]
		if !e.canDispatchLocally(st.ID) {
			return tx.raiseDispatchDeferred(fs.StepID)
		}

		// Raise CompStarted to clear NextRetryAt (self-transition)
		if err := tx.raiseCompStarted(fs.StepID, tkn); err != nil {
			return err
		}
		inputs = ex.Inputs.Apply(work.Inputs)
		outputs = work.Outputs
		meta = fl.Metadata

		tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
			go tx.performCompensation(st, inputs, outputs, tkn, meta)
		})
		return nil
	})
	return err
}

func (e *Engine) recoverCompensations(flow api.FlowState) {
	now := e.Now()
	for sid, ex := range flow.Executions {
		st, ok := flow.Plan.Steps[sid]
		if !ok {
			continue
		}
		comp, err := e.compensator(st)
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
			} else if policy.StepFailed(ex.Status) &&
				policy.WorkSucceeded(work.Status) {
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

func (tx *flowTx) raiseCompRetryScheduled(
	stepID api.StepID, tkn api.Token, work api.WorkState,
	errMsg string, nextRetryAt time.Time,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeCompRetryScheduled,
		api.CompRetryScheduledEvent{
			FlowID:      tx.flowID,
			StepID:      stepID,
			Token:       tkn,
			RetryCount:  work.RetryCount + 1,
			NextRetryAt: nextRetryAt,
			Error:       errMsg,
		},
	)
}

func compensateKey(fs api.FlowStep, tkn api.Token) []string {
	return []string{
		"comp", string(fs.FlowID), string(fs.StepID), string(tkn),
	}
}
