package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// Scheduler runs delayed tasks and supports replacement and prefix cancel
	Scheduler struct {
		now       Clock
		makeTimer TimerConstructor
		tasks     chan taskReq
	}

	// TaskFunc is called when its run time arrives
	TaskFunc func() error

	taskReqOp uint8

	taskReq struct {
		op     taskReqOp
		task   *Task
		key    taskPath
		prefix taskPath
	}
)

const (
	taskReqSchedule taskReqOp = iota
	taskReqCancel
	taskReqCancelPrefix
)

// New creates a scheduler using the provided clock and timer constructor
func New(now Clock, makeTimer TimerConstructor) *Scheduler {
	return &Scheduler{
		now:       now,
		makeTimer: makeTimer,
		tasks:     make(chan taskReq, 100),
	}
}

// Schedule enqueues a task to run at the requested time
func (s *Scheduler) Schedule(
	ctx context.Context, path []string, at time.Time, fn TaskFunc,
) {
	s.scheduleTaskReq(ctx, taskReq{
		op:   taskReqSchedule,
		task: &Task{Func: fn, At: at, Path: path},
	})
}

// Cancel removes the task registered for the exact path
func (s *Scheduler) Cancel(ctx context.Context, path []string) {
	s.scheduleTaskReq(ctx, taskReq{op: taskReqCancel, key: path})
}

// CancelPrefix removes all tasks under the provided path prefix
func (s *Scheduler) CancelPrefix(ctx context.Context, prefix []string) {
	s.scheduleTaskReq(ctx, taskReq{
		op: taskReqCancelPrefix, prefix: prefix,
	})
}

// Run processes scheduler requests until the context is cancelled
func (s *Scheduler) Run(ctx context.Context) {
	timer := s.makeTimer(0)
	var timerCh <-chan time.Time
	tasks := NewTaskHeap()

	resetTimer := func() {
		var next time.Time
		if t := tasks.Peek(); t != nil {
			next = t.At
		}
		if next.IsZero() {
			timer.Stop()
			timerCh = nil
			return
		}
		delay := next.Sub(s.now())
		timer.Reset(delay)
		timerCh = timer.Channel()
	}

	resetTimer()

	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case req := <-s.tasks:
			switch req.op {
			case taskReqSchedule:
				tasks.Insert(req.task)
			case taskReqCancel:
				tasks.Cancel(req.key)
			case taskReqCancelPrefix:
				tasks.CancelPrefix(req.prefix)
			}
			resetTimer()
		case <-timerCh:
			task := tasks.PopTask()
			if task == nil {
				resetTimer()
				continue
			}
			if err := task.Func(); err != nil {
				slog.Error("Scheduled task failed", log.Error(err))
			}
			resetTimer()
		}
	}
}

func (s *Scheduler) scheduleTaskReq(ctx context.Context, req taskReq) {
	select {
	case s.tasks <- req:
	case <-ctx.Done():
	}
}
