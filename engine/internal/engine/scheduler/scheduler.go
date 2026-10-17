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
	// Scheduler stores and processes Argyll's scheduled tasks
	Scheduler struct {
		runner *scheduler.Scheduler
	}

	// Config configures Argyll's task scheduler
	Config struct {
		Store            *timebox.Store
		Processor        scheduler.Processor
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

// New creates an Argyll scheduler over a Timebox store
func New(cfg Config) (*Scheduler, error) {
	runnerCfg := scheduler.Config{
		Store:            cfg.Store,
		Processor:        cfg.Processor,
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

// Run processes due scheduled tasks until ctx ends
func (s *Scheduler) Run(ctx context.Context) error {
	return s.runner.Run(ctx)
}

// Schedule creates or replaces a durable scheduled task
func (s *Scheduler) Schedule(
	tx *timebox.Transaction, path []string, at time.Time, msg *timebox.Message,
) error {
	return tx.Schedule(encodePath(path), at, msg)
}

func encodePath(path []string) timebox.ScheduleKey {
	var b strings.Builder
	for _, part := range path {
		b.WriteString(base64.RawURLEncoding.EncodeToString([]byte(part)))
		b.WriteByte('/')
	}
	return timebox.ScheduleKey(b.String())
}
