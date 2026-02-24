package engine

import (
	"log/slog"
	"slices"
	"time"

	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// TaskFunc is called when its deadline arrives. Returns next deadline
	// for rescheduling or zero to stop
	TaskFunc func() (nextDeadline time.Time, err error)

	Task struct {
		Func     TaskFunc
		Deadline time.Time
	}

	TaskList []*Task
)

func (e *Engine) scheduler() {
	var t retryTimer
	var timer <-chan time.Time
	var tasks TaskList

	resetTimer := func() {
		var nextTime time.Time
		if peek := tasks.Peek(); peek != nil {
			nextTime = peek.Deadline
		}

		if nextTime.IsZero() {
			t.Stop()
			timer = nil
		} else {
			timer = t.Reset(nextTime)
		}
	}

	resetTimer()

	for {
		select {
		case <-e.ctx.Done():
			t.Stop()
			return

		case tsk := <-e.tasks:
			tasks.Insert(tsk.Func, tsk.Deadline)
			resetTimer()

		case <-timer:
			tsk := tasks.Pop()
			if tsk != nil {
				// Execute task
				nextDeadline, err := tsk.Func()
				if err != nil {
					slog.Error("Scheduled task failed", log.Error(err))
				}

				// Reschedule if it returns a deadline
				if !nextDeadline.IsZero() {
					tasks.Insert(tsk.Func, nextDeadline)
				}
			}
			resetTimer()
		}
	}
}

// ScheduleTask schedules a function to run at the given deadline
func (e *Engine) ScheduleTask(fn TaskFunc, deadline time.Time) {
	select {
	case e.tasks <- Task{fn, deadline}:
	case <-e.ctx.Done():
	}
}

// retryTask executes ready retries and returns the next deadline
func (e *Engine) retryTask() (time.Time, error) {
	e.executeReadyRetries()
	nextTime, ok := e.retryQueue.Peek()
	if !ok {
		return time.Time{}, nil
	}
	return nextTime, nil
}

func (l *TaskList) Insert(fn TaskFunc, deadline time.Time) {
	*l = append(*l, &Task{
		Func:     fn,
		Deadline: deadline,
	})
	slices.SortStableFunc(*l, func(a, b *Task) int {
		if a.Deadline.Before(b.Deadline) {
			return -1
		}
		if b.Deadline.Before(a.Deadline) {
			return 1
		}
		return 0
	})
}

func (l *TaskList) Pop() *Task {
	if len(*l) == 0 {
		return nil
	}
	t := (*l)[0]
	*l = (*l)[1:]
	return t
}

func (l *TaskList) Peek() *Task {
	if len(*l) == 0 {
		return nil
	}
	return (*l)[0]
}

func (l *TaskList) Len() int {
	return len(*l)
}
