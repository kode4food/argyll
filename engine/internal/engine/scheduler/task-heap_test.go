package scheduler_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
)

func TestTaskHeapKeyedOrderAndCancelPrefix(t *testing.T) {
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	h := scheduler.NewTaskHeap()
	noop := func() error { return nil }
	insert := func(path []string, at time.Time) {
		h.Insert(&scheduler.Task{Path: path, At: at, Func: noop})
	}

	insert([]string{"a"}, now.Add(3*time.Second))
	insert([]string{"b"}, now.Add(2*time.Second))
	insert([]string{"a"}, now.Add(time.Second))

	peek := h.Peek()
	if assert.NotNil(t, peek) {
		assert.Equal(t, []string{"a"}, []string(peek.Path))
		assert.Equal(t, now.Add(time.Second).Unix(), peek.At.Unix())
	}

	h.Cancel([]string{"a"})
	peek = h.Peek()
	if assert.NotNil(t, peek) {
		assert.Equal(t, []string{"b"}, []string(peek.Path))
	}

	insert([]string{"retry", "f1", "s1", "t1"}, now)
	insert([]string{"retry", "f1", "s2", "t2"}, now)
	insert([]string{"retry", "f2", "s1", "t1"}, now)

	h.CancelPrefix([]string{"retry", "f1"})
	for {
		task := h.PopTask()
		if task == nil {
			break
		}
		assert.False(t, len(task.Path) >= 2 &&
			task.Path[0] == "retry" && task.Path[1] == "f1")
	}
}

func TestTaskHeapNoOps(t *testing.T) {
	h := scheduler.NewTaskHeap()
	assert.Nil(t, h.PopTask())
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

	h.Insert(nil)
	h.Insert(&scheduler.Task{At: now})
	h.Insert(&scheduler.Task{Func: func() error { return nil }})
	assert.Nil(t, h.Peek())

	h.Cancel(nil)
	h.Cancel([]string{"missing"})
	h.CancelPrefix(nil)
	h.CancelPrefix([]string{"missing"})
	assert.Nil(t, h.Peek())
}

func TestTaskHeapPopNonKeyed(t *testing.T) {
	h := scheduler.NewTaskHeap()
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	h.Insert(&scheduler.Task{
		At:   now,
		Func: func() error { return nil },
	})

	task := h.PopTask()
	if assert.NotNil(t, task) {
		assert.Nil(t, task.Path)
	}
	assert.Nil(t, h.PopTask())
}
