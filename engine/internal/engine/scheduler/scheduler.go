package scheduler

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/scheduler"
)

type (
	// Scheduler stores and emits Argyll's deferred messages
	Scheduler struct {
		runner *scheduler.Scheduler
	}

	// Config configures Argyll's deferred message scheduler
	Config struct {
		Store            *timebox.Store
		Emitter          scheduler.Emitter
		Clock            Clock
		TimerConstructor TimerConstructor
	}

	// Clock provides the current scheduler time
	Clock = scheduler.Clock

	// Timer provides the scheduler wakeup
	Timer = scheduler.Timer

	// TimerConstructor creates a scheduler Timer
	TimerConstructor = scheduler.TimerConstructor
)

const keyPrefix = "argyll:"

// New creates an Argyll scheduler over a Timebox store
func New(cfg Config) (*Scheduler, error) {
	runnerCfg := scheduler.Config{
		Store:            cfg.Store,
		Emitter:          cfg.Emitter,
		Clock:            cfg.Clock,
		TimerConstructor: cfg.TimerConstructor,
	}
	runner, err := scheduler.New(runnerCfg)
	if err != nil {
		return nil, err
	}
	return &Scheduler{runner: runner}, nil
}

// Now returns the current scheduler time
func Now() time.Time {
	return time.Now()
}

// Run emits due messages until ctx ends
func (s *Scheduler) Run(ctx context.Context) error {
	return s.runner.Run(ctx)
}

// Schedule creates or replaces a durable deferred message
func (s *Scheduler) Schedule(
	tx *timebox.Transaction, path []string, at time.Time, msg *timebox.Message,
) error {
	return tx.Schedule(encodePath(path), at, msg)
}

// Cancel removes the durable message at path
func (s *Scheduler) Cancel(tx *timebox.Transaction, path []string) error {
	return tx.CancelSchedule(encodePath(path))
}

// CancelPrefix removes every Argyll message below prefix
func (s *Scheduler) CancelPrefix(
	tx *timebox.Transaction, prefix []string,
) error {
	return tx.CancelSchedulePrefix(encodePath(prefix))
}

func encodePath(path []string) timebox.ScheduleKey {
	var b strings.Builder
	b.WriteString(keyPrefix)
	for _, part := range path {
		b.WriteString(base64.RawURLEncoding.EncodeToString([]byte(part)))
		b.WriteByte('/')
	}
	return timebox.ScheduleKey(b.String())
}
