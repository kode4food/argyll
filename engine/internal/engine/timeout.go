package engine

import (
	"errors"
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

func (e *Engine) scheduleTimeoutScan(
	flowID api.FlowID, deadline time.Time,
) {
	e.ScheduleTask(func() (time.Time, error) {
		nextDeadline, err := e.scanPendingTimeouts(flowID, time.Now())
		if err != nil && errors.Is(err, ErrFlowNotFound) {
			return time.Time{}, nil
		}
		return nextDeadline, err
	}, deadline)
}

func (e *Engine) scanPendingTimeouts(
	flowID api.FlowID, now time.Time,
) (time.Time, error) {
	var nextDeadline time.Time

	err := e.flowTx(flowID, func(tx *flowTx) error {
		for {
			flow := tx.Value()
			if flowTransitions.IsTerminal(flow.Status) {
				return nil
			}

			startedAny := false
			nextDeadline = time.Time{}

			for stepID, exec := range flow.Executions {
				if exec.Status != api.StepPending {
					continue
				}

				ready, d := tx.canStartStepAt(stepID, flow, now)
				if !d.IsZero() && (nextDeadline.IsZero() ||
					d.Before(nextDeadline)) {
					nextDeadline = d
				}
				if !ready {
					continue
				}

				if err := tx.prepareStep(stepID); err != nil {
					if errors.Is(err, ErrStepAlreadyPending) {
						continue
					}
					return err
				}
				startedAny = true
				break
			}

			if !startedAny {
				return nil
			}
		}
	})
	if err != nil {
		slog.Error("Optional timeout scan failed",
			log.FlowID(flowID),
			log.Error(err))
	}

	return nextDeadline, err
}
