package engine

import (
	"container/heap"
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// TaskFunc is called when its run time arrives
	TaskFunc func() error

	Task struct {
		Func  TaskFunc
		At    time.Time
		Path  taskPath
		id    string
		index int
	}

	TaskHeap struct {
		items  []*Task
		byID   map[string]*Task
		byPath *util.PathTree[*Task]
	}

	taskPath []string

	taskReqOp uint8

	taskReq struct {
		op     taskReqOp
		task   *Task
		key    taskPath
		prefix taskPath
	}

	retryTimer struct {
		timer *time.Timer
	}
)

const (
	taskReqSchedule taskReqOp = iota
	taskReqCancel
	taskReqCancelPrefix
)

func NewTaskHeap() *TaskHeap {
	h := &TaskHeap{
		byID:   map[string]*Task{},
		byPath: util.NewPathTree[*Task](),
	}
	heap.Init(h)
	return h
}

// ScheduleTask schedules a function to run at the given time
func (e *Engine) ScheduleTask(at time.Time, fn TaskFunc) {
	e.scheduleTaskReq(taskReq{
		op:   taskReqSchedule,
		task: &Task{Func: fn, At: at},
	})
}

func (e *Engine) ScheduleTaskKeyed(path []string, at time.Time, fn TaskFunc) {
	e.scheduleTaskReq(taskReq{
		op:   taskReqSchedule,
		task: &Task{Func: fn, At: at, Path: clonePath(path)},
	})
}

func (e *Engine) CancelScheduledTask(path []string) {
	e.scheduleTaskReq(taskReq{op: taskReqCancel, key: clonePath(path)})
}

func (e *Engine) CancelScheduledTaskPrefix(prefix []string) {
	e.scheduleTaskReq(taskReq{
		op: taskReqCancelPrefix, prefix: clonePath(prefix),
	})
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
			if err := task.Func(); err != nil {
				slog.Error("Scheduled task failed", log.Error(err))
			}
			resetTimer()
		}
	}
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
	if len(t.Path) > 0 {
		t.id = taskPathID(t.Path)
		if old, ok := h.byID[t.id]; ok && old != nil {
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

func (h *TaskHeap) Cancel(path []string) {
	if len(path) == 0 {
		return
	}
	t, ok := h.byID[taskPathID(path)]
	if !ok || t == nil {
		return
	}
	heap.Remove(h, t.index)
}

func (h *TaskHeap) CancelPrefix(prefix []string) {
	if len(prefix) == 0 {
		return
	}
	h.detachPrefix(prefix)
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
	if len(t.Path) > 0 {
		if t.id == "" {
			t.id = taskPathID(t.Path)
		}
		h.byID[t.id] = t
		h.byPath.Insert(t.Path, t)
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
	h.removeIndexes(t)
	return t
}

func (h *TaskHeap) removeIndexes(t *Task) {
	if t == nil || len(t.Path) == 0 {
		return
	}
	delete(h.byID, t.id)
	h.byPath.Remove(t.Path)
}

func (h *TaskHeap) detachPrefix(prefix []string) {
	h.byPath.DetachWith(prefix, func(t *Task) {
		delete(h.byID, t.id)
		heap.Remove(h, t.index)
	})
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

func clonePath(path []string) taskPath {
	if len(path) == 0 {
		return nil
	}
	cp := make(taskPath, len(path))
	copy(cp, path)
	return cp
}

func taskPathID(path []string) string {
	if len(path) == 0 {
		return ""
	}
	n := 0
	for _, p := range path {
		n += len(p) + 1
	}
	b := make([]byte, 0, n)
	for _, p := range path {
		b = append(b, p...)
		b = append(b, 0)
	}
	return string(b)
}
