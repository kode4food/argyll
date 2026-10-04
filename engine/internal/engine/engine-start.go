package engine

import (
	"errors"
	"log/slog"
	"time"
)

var (
	ErrLoadLocalHealth = errors.New("failed to load local health")
	ErrRecoverFlows    = errors.New("failed to recover flows")
)

// Start begins processing flows and events
func (e *Engine) Start() error {
	slog.Info("Engine starting")

	if err := e.loadLocalHealth(); err != nil {
		return errors.Join(ErrLoadLocalHealth, err)
	}

	e.schedulerWG.Go(func() {
		err := e.scheduler.Run(e.ctx)
		if err != nil && !errors.Is(err, e.ctx.Err()) {
			slog.Error("Scheduler stopped", slog.Any("error", err))
		}
	})

	if err := e.RecoverFlows(); err != nil {
		return errors.Join(ErrRecoverFlows, err)
	}

	return nil
}

// Now returns the current wall time from Engine's configured clock
func (e *Engine) Now() time.Time {
	return e.clock()
}
