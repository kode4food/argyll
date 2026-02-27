package engine_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
)

func TestScheduleTask(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		done := make(chan struct{}, 1)

		eng.ScheduleTask(
			[]string{"sched", "run"},
			time.Now().Add(40*time.Millisecond),
			func() error {
				done <- struct{}{}
				return nil
			},
		)

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("scheduled task did not run")
		}
	})
}

func TestScheduleTaskReplacesSamePath(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		var firstRuns atomic.Int32
		var secondRuns atomic.Int32
		path := []string{"sched", "replace"}

		eng.ScheduleTask(path, time.Now().Add(300*time.Millisecond),
			func() error {
				firstRuns.Add(1)
				return nil
			},
		)
		eng.ScheduleTask(path, time.Now().Add(40*time.Millisecond),
			func() error {
				secondRuns.Add(1)
				return nil
			},
		)

		time.Sleep(450 * time.Millisecond)
		assert.Equal(t, int32(0), firstRuns.Load())
		assert.Equal(t, int32(1), secondRuns.Load())
	})
}

func TestCancelTask(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		var ran atomic.Bool

		path := []string{"sched", "cancel", "one"}
		eng.ScheduleTask(path, time.Now().Add(100*time.Millisecond),
			func() error {
				ran.Store(true)
				return nil
			},
		)
		eng.CancelTask(path)

		time.Sleep(250 * time.Millisecond)
		assert.False(t, ran.Load())
	})
}

func TestCancelPrefixedTasks(t *testing.T) {
	helpers.WithStartedEngine(t, func(eng *engine.Engine) {
		var cancelledRuns atomic.Int32
		var activeRuns atomic.Int32

		cancelledPrefix := []string{"sched", "prefix", "cancelled"}
		eng.ScheduleTask(
			[]string{"sched", "prefix", "cancelled", "a"},
			time.Now().Add(100*time.Millisecond),
			func() error {
				cancelledRuns.Add(1)
				return nil
			},
		)
		eng.ScheduleTask(
			[]string{"sched", "prefix", "cancelled", "b"},
			time.Now().Add(100*time.Millisecond),
			func() error {
				cancelledRuns.Add(1)
				return nil
			},
		)

		eng.ScheduleTask(
			[]string{"sched", "prefix", "active", "c"},
			time.Now().Add(100*time.Millisecond),
			func() error {
				activeRuns.Add(1)
				return nil
			},
		)

		eng.CancelPrefixedTasks(cancelledPrefix)

		time.Sleep(300 * time.Millisecond)
		assert.Equal(t, int32(0), cancelledRuns.Load())
		assert.Equal(t, int32(1), activeRuns.Load())
	})
}

func TestTaskHeapKeyedOrderAndCancelPrefix(t *testing.T) {
	now := time.Now()
	h := engine.NewTaskHeap()
	noop := func() error { return nil }
	insert := func(path []string, at time.Time) {
		h.Insert(&engine.Task{Path: path, At: at, Func: noop})
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
	h := engine.NewTaskHeap()
	assert.Nil(t, h.PopTask())

	h.Insert(nil)
	h.Insert(&engine.Task{At: time.Now()})
	h.Insert(&engine.Task{Func: func() error { return nil }})
	assert.Nil(t, h.Peek())

	h.Cancel(nil)
	h.Cancel([]string{"missing"})
	h.CancelPrefix(nil)
	h.CancelPrefix([]string{"missing"})
	assert.Nil(t, h.Peek())
}

func TestTaskHeapPopNonKeyed(t *testing.T) {
	h := engine.NewTaskHeap()
	h.Insert(&engine.Task{
		At:   time.Now(),
		Func: func() error { return nil },
	})

	task := h.PopTask()
	if assert.NotNil(t, task) {
		assert.Nil(t, task.Path)
	}
	assert.Nil(t, h.PopTask())
}
