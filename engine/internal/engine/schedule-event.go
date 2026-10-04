package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type (
	scheduledWork struct {
		StepID api.StepID
		Token  api.Token
	}

	scheduledTimeout struct {
		StepID api.StepID
		Name   api.Name
	}
)

const (
	workDeadlineElapsed    timebox.EventType = "work_deadline_elapsed"
	workRetryReady         timebox.EventType = "work_retry_ready"
	compensationRetryReady timebox.EventType = "compensation_retry_ready"
	workDispatchRequested  timebox.EventType = "work_dispatch_requested"
	stepTimeoutElapsed     timebox.EventType = "step_timeout_elapsed"
	flowReconcileRequested timebox.EventType = "flow_reconcile_requested"
)

var (
	ErrScheduledEventTarget = errors.New("invalid scheduled event target")
)

func (e *Engine) emitScheduled(
	_ context.Context, delivery *scheduler.Delivery,
) error {
	ev := delivery.Event()
	fid, ok := events.ParseFlowID(ev.AggregateID)
	if !ok {
		return fmt.Errorf("%w: %s", ErrScheduledEventTarget,
			ev.AggregateID.String())
	}
	err := e.flowTx(fid, func(tx *flowTx) error {
		if err := delivery.Consume(tx.storeTx.Transaction); err != nil {
			return err
		}
		return tx.handleScheduled(ev)
	})
	if errors.Is(err, ErrDispatchUnavailable) {
		return nil
	}
	return err
}

func (tx *flowTx) handleScheduled(ev *timebox.Event) error {
	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		workDeadlineElapsed: timebox.MakeHandler(
			func(_ *timebox.Event, work scheduledWork) error {
				return tx.handleWorkDeadline(work.StepID, work.Token)
			},
		),
		workRetryReady: timebox.MakeHandler(
			func(_ *timebox.Event, work scheduledWork) error {
				return tx.handleRetry(work.StepID, work.Token)
			},
		),
		compensationRetryReady: timebox.MakeHandler(
			func(_ *timebox.Event, work scheduledWork) error {
				return tx.handleCompensationRetry(work.StepID, work.Token)
			},
		),
		workDispatchRequested: timebox.MakeHandler(
			func(_ *timebox.Event, sid api.StepID) error {
				return tx.handleWorkDispatch(sid)
			},
		),
		stepTimeoutElapsed: timebox.MakeHandler(
			func(_ *timebox.Event, timeout scheduledTimeout) error {
				return tx.handleStepTimeout(
					timeout.StepID, timeout.Name, tx.Now(),
				)
			},
		),
		flowReconcileRequested: func(*timebox.Event) error {
			return tx.recoverFlow()
		},
	})(ev)
}

func (tx *flowTx) scheduleEvent(
	path []string, at time.Time, eventType timebox.EventType, value any,
) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = tx.scheduler.Schedule(tx.storeTx.Transaction, path, at,
		&timebox.Event{
			AggregateID: events.FlowKey(tx.flowID),
			Type:        eventType,
			Data:        b,
		})
	if err != nil {
		return err
	}
	tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
		tx.scheduler.Wake()
	})
	return nil
}

func (tx *flowTx) cancelEvent(path []string) error {
	if err := tx.scheduler.Cancel(
		tx.storeTx.Transaction, path,
	); err != nil {
		return err
	}
	tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
		tx.scheduler.Wake()
	})
	return nil
}

func (tx *flowTx) cancelEventPrefix(prefix []string) error {
	if err := tx.scheduler.CancelPrefix(
		tx.storeTx.Transaction, prefix,
	); err != nil {
		return err
	}
	tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
		tx.scheduler.Wake()
	})
	return nil
}
