package engine

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// Start begins processing flows and events
func (e *Engine) Start() error {
	slog.Info("Engine starting")

	e.eventQueue.Start()
	go e.scheduler.Run(e.ctx)

	if err := e.RecoverFlows(); err != nil {
		e.eventQueue.Cancel()
		return fmt.Errorf("%w: %w", ErrRecoverFlows, err)
	}

	return nil
}

// ScheduleTask schedules a function to run at the given time
func (e *Engine) ScheduleTask(
	path []string, at time.Time, fn scheduler.TaskFunc,
) {
	e.scheduler.Schedule(e.ctx, path, at, fn)
}

// CancelTask removes a scheduled task for the exact path
func (e *Engine) CancelTask(path []string) {
	e.scheduler.Cancel(e.ctx, path)
}

// CancelPrefixedTasks removes all scheduled tasks under the given prefix
func (e *Engine) CancelPrefixedTasks(prefix []string) {
	e.scheduler.CancelPrefix(e.ctx, prefix)
}

// Now returns the current wall time from Engine's configured clock
func (e *Engine) Now() time.Time {
	return e.clock()
}

// EnqueueEvent schedules a partition aggregate event for sequential processing
func (e *Engine) EnqueueEvent(typ api.EventType, data any) {
	e.eventQueue.Enqueue(typ, data)
}
