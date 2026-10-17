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
	// workSettler records how dispatched work or its compensation ended
	workSettler interface {
		step.Runtime
		notCompleteWork(api.Token, string) error
		failWork(api.Token, string) error
	}

	// workRuntime is the Runtime an invocation receives, settling work items
	workRuntime struct {
		*ExecContext
	}

	// txRuntime settles an Inline handler's work in the transaction that
	// started it, keeping the first settlement error to abort that transaction
	txRuntime struct {
		tx  *flowTx
		sid api.StepID
		err error
	}
)

func (r workRuntime) CompleteWork(tkn api.Token, outputs api.Args) error {
	return r.engine.CompleteWork(r.flowStep(), tkn, outputs)
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
			slog.Error("Failed to update step health",
				log.StepID(r.sid),
				log.Error(err))
		}
	})
	return nil
}

func (r *txRuntime) CompleteWork(tkn api.Token, outputs api.Args) error {
	return r.settle(r.tx.completeWork(r.sid, tkn, outputs))
}

func (r workRuntime) notCompleteWork(tkn api.Token, errMsg string) error {
	return r.engine.NotCompleteWork(r.flowStep(), tkn, errMsg)
}

func (r workRuntime) failWork(tkn api.Token, errMsg string) error {
	return r.engine.FailWork(r.flowStep(), tkn, errMsg)
}

func (r *txRuntime) notCompleteWork(tkn api.Token, errMsg string) error {
	if err := r.tx.raiseWorkNotCompleted(r.sid, tkn, errMsg); err != nil {
		return r.settle(err)
	}
	return r.settle(r.tx.handleWorkNotCompleted(r.sid, tkn))
}

func (r *txRuntime) failWork(tkn api.Token, errMsg string) error {
	return r.settle(r.tx.failWork(r.sid, tkn, errMsg))
}

func (r *txRuntime) settle(err error) error {
	if r.err == nil {
		r.err = err
	}
	return err
}

// settleFailure records a handler error as a retry when the handler reports the
// work not completed, and as a permanent failure otherwise
func settleFailure(s workSettler, tkn api.Token, err error) {
	if errors.Is(err, api.ErrInvalidWorkTransition) {
		return
	}

	if errors.Is(err, api.ErrWorkNotCompleted) {
		if recErr := s.notCompleteWork(tkn, err.Error()); recErr != nil {
			slog.Error("Failed to record work not completed",
				log.FlowID(s.FlowID()),
				log.StepID(s.StepID()),
				log.Error(recErr))
		}
		return
	}

	if recErr := s.failWork(tkn, err.Error()); recErr != nil {
		slog.Error("Failed to record work failure",
			log.FlowID(s.FlowID()),
			log.StepID(s.StepID()),
			log.Error(recErr))
	}
}
