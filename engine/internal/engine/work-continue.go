package engine

import (
	"math"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type backoffCalculator func(baseDelay int64, retryCount int) int64

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

// ShouldRetry determines if a failed work item should be retried based on
// configured retry limits
func (e *Engine) ShouldRetry(st *api.Step, work api.WorkState) bool {
	workConfig := e.resolveRetryConfig(st.WorkConfig)

	if workConfig.MaxRetries == 0 {
		return false
	}
	if workConfig.MaxRetries < 0 {
		return true
	}

	return work.RetryCount < workConfig.MaxRetries
}

// CalculateNextRetry calculates the next retry time using the configured
// backoff strategy
func (e *Engine) CalculateNextRetry(
	config *api.WorkConfig, retryCount int,
) time.Time {
	return e.calculateNextRetryAt(e.Now(), config, retryCount)
}

func (e *Engine) calculateNextRetryAt(
	when time.Time, config *api.WorkConfig, retryCount int,
) time.Time {
	config = e.resolveRetryConfig(config)

	calculator, ok := backoffCalculators[config.BackoffType]
	if !ok {
		calculator = backoffCalculators[api.BackoffTypeFixed]
	}

	delay := min(
		calculator(config.InitBackoff, retryCount), config.MaxBackoff,
	)

	return when.Add(time.Duration(delay) * time.Millisecond)
}

func (tx *flowTx) scheduleRetry(sid api.StepID, tkn api.Token) error {
	ex := tx.Value().Executions[sid]
	if !policy.StepActive(ex.Status) {
		return nil
	}

	work, ok := ex.WorkItems[tkn]
	if !ok || !policy.WorkNotCompleted(work.Status) {
		return nil
	}

	st := tx.Value().Plan.Steps[sid]
	if tx.ShouldRetry(st, work) {
		nextRetryAt := tx.calculateNextRetryAt(
			tx.Now(), st.WorkConfig, work.RetryCount,
		)
		err := tx.raiseRetryScheduled(sid, tkn, work, nextRetryAt)
		if err != nil {
			return err
		}
		return nil
	}

	return tx.raiseWorkFailed(sid, tkn, work.Error)
}

func (tx *flowTx) continueStepWork(sid api.StepID, clearRetry bool) error {
	st := tx.Value().Plan.Steps[sid]
	started, err := tx.startPendingWork(st)
	if err != nil {
		return err
	}
	ex := tx.Value().Executions[sid]
	if policy.WorkReadyToDispatch(st, ex, tx.Now()) &&
		!tx.canDispatchLocally(st.ID) {
		if err := tx.raiseDispatchDeferred(sid); err != nil {
			return err
		}
	}
	if len(started) == 0 {
		return nil
	}
	if clearRetry {
		for tkn := range started {
			tx.clearRetryTask(sid, tkn)
		}
	}
	return tx.startContinuedWork(sid, st, started)
}

func (tx *flowTx) handleWorkContinuation(sid api.StepID) error {
	return tx.continueStepWork(sid, true)
}

func (tx *flowTx) startContinuedWork(
	sid api.StepID, st *api.Step, started api.WorkItems,
) error {
	tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
		ex := fl.Executions[sid]
		tx.executeStartedWork(st, ex.Inputs, fl.Metadata, started)
	})
	return nil
}

func (e *Engine) scheduleRetryTask(
	fs api.FlowStep, tkn api.Token, retryAt time.Time,
) {
	e.ScheduleTask(retryKey(fs, tkn), retryAt, func() error {
		err := e.runRetryTask(fs, tkn)
		if err != nil {
			e.scheduleRetryTask(fs, tkn,
				e.Now().Add(localDispatchBackoff),
			)
		}
		return err
	})
}

func (e *Engine) runRetryTask(fs api.FlowStep, tkn api.Token) error {
	var inputs api.Args
	var st *api.Step
	var meta api.Metadata

	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		fl := tx.Value()
		if fl.ID == "" || policy.FlowTerminal(fl.Status) {
			return nil
		}

		ex := fl.Executions[fs.StepID]
		if _, ok := ex.WorkItems[tkn]; !ok {
			return nil
		}

		st = fl.Plan.Steps[fs.StepID]

		work := ex.WorkItems[tkn]
		if policy.WorkClaimableForRetry(work.Status) &&
			!tx.canDispatchLocally(st.ID) {
			return tx.raiseDispatchDeferred(fs.StepID)
		}

		inputs = ex.Inputs
		meta = fl.Metadata

		started, retryAt, err := tx.startRetryWorkItem(st, tkn)
		if err != nil {
			return err
		}
		if retryAt.IsZero() && len(started) == 0 {
			return nil
		}

		tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
			if !retryAt.IsZero() {
				tx.scheduleRetryTask(fs, tkn, retryAt)
				return
			}
			tx.executeStartedWork(st, inputs, meta, started)
		})
		return nil
	})
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

func retryKey(fs api.FlowStep, tkn api.Token) []string {
	return []string{
		string(fs.FlowID), "retry", string(fs.StepID), string(tkn),
	}
}

func retryPrefix(fid api.FlowID) []string {
	return []string{string(fid), "retry"}
}
