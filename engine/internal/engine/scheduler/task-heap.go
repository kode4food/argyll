package scheduler

import (
	"container/heap"
	"time"

	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// Task describes a scheduled function and its execution metadata
	Task struct {
		Func  TaskFunc
		At    time.Time
		Path  taskPath
		id    string
		index int
	}

	// TaskHeap stores scheduled tasks ordered by execution time
	TaskHeap struct {
		items  []*Task
		byID   map[string]*Task
		byPath *util.PathTree[*Task]
	}

	taskPath []string
)

// NewTaskHeap creates an empty task heap with keyed lookup indexes
func NewTaskHeap() *TaskHeap {
	h := &TaskHeap{
		byID:   map[string]*Task{},
		byPath: util.NewPathTree[*Task](),
	}
	heap.Init(h)
	return h
}

// Insert adds a task to the heap or replaces an existing keyed task
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

// PopTask removes and returns the next scheduled task
func (h *TaskHeap) PopTask() *Task {
	if h.Len() == 0 {
		return nil
	}
	return heap.Pop(h).(*Task)
}

// Peek returns the next scheduled task without removing it
func (h *TaskHeap) Peek() *Task {
	if len(h.items) == 0 {
		return nil
	}
	return h.items[0]
}

// Cancel removes the keyed task for the exact path
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

// CancelPrefix removes all keyed tasks under the provided prefix
func (h *TaskHeap) CancelPrefix(prefix []string) {
	if len(prefix) == 0 {
		return
	}
	h.detachPrefix(prefix)
}

// Len returns the number of scheduled tasks in the heap
func (h *TaskHeap) Len() int {
	return len(h.items)
}

// Less reports whether the task at i should sort before the task at j
func (h *TaskHeap) Less(i, j int) bool {
	return h.items[i].At.Before(h.items[j].At)
}

// Swap exchanges the heap items at the provided indexes
func (h *TaskHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].index = i
	h.items[j].index = j
}

// Push adds a task to the underlying heap implementation
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

// Pop removes a task from the underlying heap implementation
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
