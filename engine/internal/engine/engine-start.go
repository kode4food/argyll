package engine

import (
	"errors"
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
)

var (
	ErrRecoverFlows = errors.New("failed to recover flows")
)

// Start begins processing flows and events
func (e *Engine) Start() error {
	slog.Info("Engine starting")

	go e.scheduler.Run(e.ctx)

	if err := e.RecoverFlows(); err != nil {
		return errors.Join(ErrRecoverFlows, err)
	}

	return nil
}

// ScheduleTask schedules a function to run at the given time
func (e *Engine) ScheduleTask(
	path []string, at time.Time, fn scheduler.TaskFunc,
) {
	e.scheduler.Schedule(path, at, fn)
}

// CancelTask removes a scheduled task for the exact path
func (e *Engine) CancelTask(path []string) {
	e.scheduler.Cancel(path)
}

// CancelPrefixedTasks removes all scheduled tasks under the given prefix
func (e *Engine) CancelPrefixedTasks(prefix []string) {
	e.scheduler.CancelPrefix(prefix)
}

// Now returns the current wall time from Engine's configured clock
func (e *Engine) Now() time.Time {
	return e.clock()
}
