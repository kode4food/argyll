package engine

import (
	"errors"
	"log/slog"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/step"
)

type (
	// workRuntime is the Runtime an invocation receives, settling work items
	workRuntime struct {
		*ExecContext
	}

	// txRuntime settles an Inline handler's work in the transaction that started it, keeping the
	// first settlement error to abort that transaction
	txRuntime struct {
		tx  *flowTx
		sid api.StepID
		err error
	}
)

func (r workRuntime) CompleteWork(tkn api.Token, outputs api.Args) error {
	return r.engine.CompleteWork(r.flowStep(), tkn, outputs, nil)
}

func (r workRuntime) RetryWork(tkn api.Token, cause error) error {
	return r.engine.RetryWork(r.flowStep(), tkn, cause, nil)
}

func (r workRuntime) FailWork(tkn api.Token, cause error) error {
	return r.engine.FailWork(r.flowStep(), tkn, cause, nil)
}

func (r *txRuntime) FlowID() api.FlowID {
	return r.tx.flowID
}

func (r *txRuntime) StepID() api.StepID {
	return r.sid
}

func (r *txRuntime) Metadata() api.Metadata {
	return r.tx.Value().Metadata
}

func (r *txRuntime) UpdateHealth(s api.HealthStatus, msg string) error {
	r.tx.OnSuccess(func(api.FlowState, []*timebox.Event) {
		if err := r.tx.UpdateStepHealth(r.sid, s, msg); err != nil {
			slog.Error("Failed to update step health", log.StepID(r.sid), log.Error(err))
		}
	})
	return nil
}

func (r *txRuntime) CompleteWork(tkn api.Token, outputs api.Args) error {
	return r.settle(r.tx.completeWork(r.sid, tkn, outputs, nil))
}

func (r *txRuntime) RetryWork(tkn api.Token, cause error) error {
	return r.settle(r.tx.retryWork(r.sid, tkn, errorText(cause), nil))
}

func (r *txRuntime) FailWork(tkn api.Token, cause error) error {
	return r.settle(r.tx.failWork(r.sid, tkn, errorText(cause), nil))
}

func (r *txRuntime) settle(err error) error {
	if r.err == nil {
		r.err = err
	}
	return err
}

// settleFailure records a handler error as a retry when the handler reports the work not completed,
// and as a permanent failure otherwise
func settleFailure(rt step.Runtime, tkn api.Token, err error) {
	if errors.Is(err, api.ErrInvalidWorkTransition) {
		return
	}

	if errors.Is(err, api.ErrWorkNotCompleted) {
		if recErr := rt.RetryWork(tkn, err); recErr != nil {
			slog.Error("Failed to record work not completed",
				log.FlowID(rt.FlowID()),
				log.StepID(rt.StepID()),
				log.Error(recErr))
		}
		return
	}

	if recErr := rt.FailWork(tkn, err); recErr != nil {
		slog.Error("Failed to record work failure",
			log.FlowID(rt.FlowID()),
			log.StepID(rt.StepID()),
			log.Error(recErr))
	}
}

// errorText is the message an error records on its work item
func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
