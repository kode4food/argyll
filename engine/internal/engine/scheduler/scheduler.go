package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// Scheduler runs delayed tasks and supports replacement and prefix cancel
	Scheduler struct {
		mu         sync.Mutex
		cond       *sync.Cond
		now        Clock
		timer      Timer
		timerFired bool
		tasks      *TaskHeap
	}

	// TaskFunc is called when its run time arrives
	TaskFunc func() error

	taskHead struct {
		id string
		at time.Time
		ok bool
	}
)

// New creates a scheduler using the provided clock and timer constructor
func New(now Clock, makeTimer TimerConstructor) *Scheduler {
	s := &Scheduler{
		now:   now,
		timer: makeTimer(0),
		tasks: NewTaskHeap(),
	}
	s.timer.Stop()
	s.cond = sync.NewCond(&s.mu)
	return s
}

// Schedule enqueues a task to run at the requested time
func (s *Scheduler) Schedule(
	path []string, at time.Time, fn TaskFunc,
) {
	s.mu.Lock()
	prev := s.currentHead()
	s.tasks.Insert(&Task{Func: fn, At: at, Path: path})
	s.notifyIfHeadChanged(prev)
	s.mu.Unlock()
}

// Cancel removes the task registered for the exact path
func (s *Scheduler) Cancel(path []string) {
	s.mu.Lock()
	prev := s.currentHead()
	s.tasks.Cancel(path)
	s.notifyIfHeadChanged(prev)
	s.mu.Unlock()
}

// CancelPrefix removes all tasks under the provided path prefix
func (s *Scheduler) CancelPrefix(prefix []string) {
	s.mu.Lock()
	prev := s.currentHead()
	s.tasks.CancelPrefix(prefix)
	s.notifyIfHeadChanged(prev)
	s.mu.Unlock()
}

// Run processes scheduler requests until the context is cancelled
func (s *Scheduler) Run(ctx context.Context) {
	s.mu.Lock()
	s.resetTimer()
	go s.signalTimer(ctx)
	s.mu.Unlock()

	for {
		s.mu.Lock()
		for {
			if ctx.Err() != nil {
				s.timer.Stop()
				s.mu.Unlock()
				return
			}

			if s.tasks.Peek() == nil {
				s.cond.Wait()
				continue
			}
			task := s.tasks.Peek()
			if !s.timerFired && task.At.After(s.now()) {
				s.cond.Wait()
				continue
			}

			task = s.tasks.PopTask()
			s.resetTimer()
			s.mu.Unlock()
			if err := task.Func(); err != nil {
				slog.Error("Scheduled task failed", log.Error(err))
			}
			break
		}
	}
}

func (s *Scheduler) resetTimer() {
	next := s.nextRunAt()
	if next.IsZero() {
		s.timerFired = false
		s.timer.Stop()
		return
	}

	delay := max(next.Sub(s.now()), 0)
	s.timerFired = delay == 0
	s.timer.Reset(delay)
}

func (s *Scheduler) notifyIfHeadChanged(prev taskHead) {
	if !headChanged(prev, s.currentHead()) {
		return
	}
	s.resetTimer()
	s.cond.Signal()
}

func (s *Scheduler) currentHead() taskHead {
	t := s.tasks.Peek()
	if t == nil {
		return taskHead{}
	}
	return taskHead{id: t.id, at: t.At, ok: true}
}

func (s *Scheduler) nextRunAt() time.Time {
	if t := s.tasks.Peek(); t != nil {
		return t.At
	}
	return time.Time{}
}

func headChanged(prev, next taskHead) bool {
	return prev.ok != next.ok || prev.id != next.id || !prev.at.Equal(next.at)
}

func (s *Scheduler) signalTimer(ctx context.Context) {
	ch := s.timer.Channel()
	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			s.cond.Broadcast()
			s.mu.Unlock()
			return
		case <-ch:
			s.mu.Lock()
			s.timerFired = true
			s.cond.Signal()
			s.mu.Unlock()
		}
	}
}
