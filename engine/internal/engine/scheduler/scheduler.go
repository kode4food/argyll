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
	// Scheduler stores and emits Argyll's deferred events
	Scheduler struct {
		runner *scheduler.Scheduler
	}

	// Delivery is one due deferred event
	Delivery struct {
		schedule *timebox.Schedule
	}

	// Config configures Argyll's deferred event scheduler
	Config struct {
		Store            *timebox.Store
		Emitter          func(context.Context, *Delivery) error
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
		Store: cfg.Store,
		Emitter: func(
			ctx context.Context, item *timebox.Schedule,
		) error {
			return cfg.Emitter(ctx, &Delivery{schedule: item})
		},
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

// Run emits due events until ctx ends or durable recovery fails
func (s *Scheduler) Run(ctx context.Context) error {
	return s.runner.Run(ctx)
}

// Wake requests prompt recovery of durable schedules
func (s *Scheduler) Wake() {
	s.runner.Wake()
}

// Schedule creates or replaces a durable deferred event
func (s *Scheduler) Schedule(
	tx *timebox.Transaction, path []string, at time.Time, event *timebox.Event,
) error {
	return tx.Schedule(encodePath(path), at, event)
}

// Cancel removes the durable event at path
func (s *Scheduler) Cancel(tx *timebox.Transaction, path []string) error {
	return tx.CancelSchedule(encodePath(path))
}

// CancelPrefix removes every Argyll event below prefix
func (s *Scheduler) CancelPrefix(
	tx *timebox.Transaction, prefix []string,
) error {
	return tx.CancelSchedulePrefix(encodePath(prefix))
}

// Event returns the ordinary Timebox event due for delivery
func (d *Delivery) Event() *timebox.Event {
	return d.schedule.Event
}

// Consume conditionally consumes this delivery in tx
func (d *Delivery) Consume(tx *timebox.Transaction) error {
	return tx.ConsumeSchedule(d.schedule.Key, d.schedule.Version)
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
