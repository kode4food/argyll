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

	task struct {
		fn       TaskFunc
		deadline time.Time
	}

	taskList []*task
)

func (e *Engine) scheduler() {
	var t retryTimer
	var timer <-chan time.Time
	var tasks taskList

	resetTimer := func() {
		var nextTime time.Time
		if peek := tasks.Peek(); peek != nil {
			nextTime = peek.deadline
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
			tasks.Insert(tsk.fn, tsk.deadline)
			resetTimer()

		case <-timer:
			tsk := tasks.Pop()
			if tsk != nil {
				// Execute task
				nextDeadline, err := tsk.fn()
				if err != nil {
					slog.Error("Scheduled task failed", log.Error(err))
				}

				// Reschedule if it returns a deadline
				if !nextDeadline.IsZero() {
					tasks.Insert(tsk.fn, nextDeadline)
				}
			}
			resetTimer()
		}
	}
}

// RegisterTask schedules a function to run at the given deadline
func (e *Engine) RegisterTask(fn TaskFunc, deadline time.Time) {
	select {
	case e.tasks <- task{fn, deadline}:
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

// timeoutTask executes expired timeouts and returns the next deadline
func (e *Engine) timeoutTask() (time.Time, error) {
	if err := e.fireExpiredTimeouts(); err != nil {
		return time.Time{}, err
	}
	partState, err := e.GetPartitionState()
	if err != nil {
		return time.Time{}, err
	}
	if len(partState.Timeouts) > 0 {
		return partState.Timeouts[0].FiresAt, nil
	}
	return time.Time{}, nil
}

func (l *taskList) Insert(fn TaskFunc, deadline time.Time) {
	*l = append(*l, &task{
		fn:       fn,
		deadline: deadline,
	})
	slices.SortStableFunc(*l, func(a, b *task) int {
		if a.deadline.Before(b.deadline) {
			return -1
		}
		if b.deadline.Before(a.deadline) {
			return 1
		}
		return 0
	})
}

func (l *taskList) Pop() *task {
	if len(*l) == 0 {
		return nil
	}
	t := (*l)[0]
	*l = (*l)[1:]
	return t
}

func (l *taskList) Peek() *task {
	if len(*l) == 0 {
		return nil
	}
	return (*l)[0]
}

func (l *taskList) Len() int {
	return len(*l)
}
