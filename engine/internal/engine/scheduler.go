package engine

import (
	"container/heap"
	"log/slog"
	"strings"
	"time"

	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// TaskFunc is called when its run time arrives
	TaskFunc func() error

	Task struct {
		Func  TaskFunc
		At    time.Time
		Key   string
		index int
	}

	taskReqOp uint8

	taskReq struct {
		op     taskReqOp
		task   *Task
		key    string
		prefix string
	}

	retryTimer struct {
		timer *time.Timer
	}

	TaskHeap struct {
		items []*Task
		byKey map[string]*Task
	}
)

const (
	taskReqSchedule taskReqOp = iota
	taskReqCancel
	taskReqCancelPrefix
)

func NewTaskHeap() *TaskHeap {
	h := &TaskHeap{byKey: map[string]*Task{}}
	heap.Init(h)
	return h
}

func (e *Engine) scheduler() {
	var t retryTimer
	var timer <-chan time.Time
	tasks := NewTaskHeap()

	resetTimer := func() {
		var next time.Time
		if t := tasks.Peek(); t != nil {
			next = t.At
		}

		if next.IsZero() {
			t.Stop()
			timer = nil
			return
		}
		timer = t.Reset(next)
	}

	resetTimer()

	for {
		select {
		case <-e.ctx.Done():
			t.Stop()
			return
		case req := <-e.tasks:
			switch req.op {
			case taskReqSchedule:
				tasks.Insert(req.task)
			case taskReqCancel:
				tasks.Cancel(req.key)
			case taskReqCancelPrefix:
				tasks.CancelPrefix(req.prefix)
			}
			resetTimer()
		case <-timer:
			task := tasks.PopTask()
			if task == nil {
				resetTimer()
				continue
			}
			err := task.Func()
			if err != nil {
				slog.Error("Scheduled task failed", log.Error(err))
			}
			resetTimer()
		}
	}
}

// ScheduleTask schedules a function to run at the given time
func (e *Engine) ScheduleTask(fn TaskFunc, at time.Time) {
	e.scheduleTaskReq(taskReq{
		op:   taskReqSchedule,
		task: &Task{Func: fn, At: at},
	})
}

func (e *Engine) ScheduleTaskKeyed(key string, fn TaskFunc, at time.Time) {
	e.scheduleTaskReq(taskReq{
		op:   taskReqSchedule,
		task: &Task{Func: fn, At: at, Key: key},
	})
}

func (e *Engine) CancelScheduledTask(key string) {
	e.scheduleTaskReq(taskReq{op: taskReqCancel, key: key})
}

func (e *Engine) CancelScheduledTaskPrefix(prefix string) {
	e.scheduleTaskReq(taskReq{op: taskReqCancelPrefix, prefix: prefix})
}

func (e *Engine) scheduleTaskReq(req taskReq) {
	select {
	case e.tasks <- req:
	case <-e.ctx.Done():
	}
}

func (h *TaskHeap) Insert(t *Task) {
	if t == nil || t.Func == nil || t.At.IsZero() {
		return
	}
	if t.Key != "" {
		if old, ok := h.byKey[t.Key]; ok && old != nil {
			old.Func = t.Func
			old.At = t.At
			heap.Fix(h, old.index)
			return
		}
	}
	heap.Push(h, t)
}

func (h *TaskHeap) PopTask() *Task {
	if h.Len() == 0 {
		return nil
	}
	return heap.Pop(h).(*Task)
}

func (h *TaskHeap) Peek() *Task {
	if len(h.items) == 0 {
		return nil
	}
	return h.items[0]
}

func (h *TaskHeap) Cancel(key string) {
	if key == "" {
		return
	}
	t, ok := h.byKey[key]
	if !ok || t == nil {
		return
	}
	heap.Remove(h, t.index)
}

func (h *TaskHeap) CancelPrefix(prefix string) {
	if prefix == "" {
		return
	}
	for key, t := range h.byKey {
		if t == nil || !strings.HasPrefix(key, prefix) {
			continue
		}
		heap.Remove(h, t.index)
	}
}

func (h *TaskHeap) Len() int {
	return len(h.items)
}

func (h *TaskHeap) Less(i, j int) bool {
	return h.items[i].At.Before(h.items[j].At)
}

func (h *TaskHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].index = i
	h.items[j].index = j
}

func (h *TaskHeap) Push(x any) {
	t := x.(*Task)
	t.index = len(h.items)
	h.items = append(h.items, t)
	if t.Key != "" {
		h.byKey[t.Key] = t
	}
}

func (h *TaskHeap) Pop() any {
	old := h.items
	n := len(old)
	if n == 0 {
		return nil
	}
	t := old[n-1]
	old[n-1] = nil
	h.items = old[:n-1]
	t.index = -1
	if t.Key != "" {
		delete(h.byKey, t.Key)
	}
	return t
}

func (t *retryTimer) Reset(nextTime time.Time) <-chan time.Time {
	delay := max(time.Until(nextTime), 0)
	if t.timer == nil {
		t.timer = time.NewTimer(delay)
		return t.timer.C
	}
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
	t.timer.Reset(delay)
	return t.timer.C
}

func (t *retryTimer) Stop() {
	if t.timer != nil {
		t.timer.Stop()
	}
}
