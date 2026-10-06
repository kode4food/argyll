package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/scheduler"

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
	ErrScheduledMessageTarget = errors.New("invalid scheduled message target")
)

func (e *Engine) processScheduled(
	t *timebox.Transaction, msg *timebox.Message,
) error {
	fid, ok := events.ParseFlowID(msg.AggregateID)
	if !ok {
		return fmt.Errorf("%w: %s", ErrScheduledMessageTarget,
			msg.AggregateID.String())
	}
	st := storeTx{Engine: e, Transaction: t}
	_, err := st.flowTx(fid, func(tx *flowTx) error {
		return tx.handleScheduled(msg)
	})
	if errors.Is(err, ErrDispatchUnavailable) {
		return scheduler.ErrRetry
	}
	return err
}

func (tx *flowTx) handleScheduled(msg *timebox.Message) error {
	switch msg.Type {
	case workDeadlineElapsed:
		return handleScheduledValue(msg, func(work scheduledWork) error {
			return tx.handleWorkDeadline(work.StepID, work.Token)
		})
	case workRetryReady:
		return handleScheduledValue(msg, func(work scheduledWork) error {
			return tx.handleRetry(work.StepID, work.Token)
		})
	case compensationRetryReady:
		return handleScheduledValue(msg, func(work scheduledWork) error {
			return tx.handleCompensationRetry(work.StepID, work.Token)
		})
	case workDispatchRequested:
		return handleScheduledValue(msg, tx.handleWorkDispatch)
	case stepTimeoutElapsed:
		return handleScheduledValue(msg, func(timeout scheduledTimeout) error {
			return tx.handleStepTimeout(
				timeout.StepID, timeout.Name, tx.Now(),
			)
		})
	case flowReconcileRequested:
		return tx.recoverFlow()
	}
	return nil
}

func (tx *flowTx) scheduleEvent(
	path []string, at time.Time, eventType timebox.EventType, value any,
) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return tx.scheduler.Schedule(tx.storeTx.Transaction, path, at,
		&timebox.Message{
			AggregateID: events.FlowKey(tx.flowID),
			Type:        eventType,
			Data:        b,
		},
	)
}

func (tx *flowTx) cancelEvent(path []string) error {
	return tx.scheduler.Cancel(tx.storeTx.Transaction, path)
}

func (tx *flowTx) cancelEventPrefix(prefix []string) error {
	return tx.scheduler.CancelPrefix(tx.storeTx.Transaction, prefix)
}

func handleScheduledValue[T any](msg *timebox.Message, fn func(T) error) error {
	value, err := msg.GetValue[T]()
	if err != nil {
		return err
	}
	return fn(value)
}
