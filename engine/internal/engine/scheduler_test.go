package engine_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
)

const schedulerWaitTimeout = time.Second

func TestScheduleTask(t *testing.T) {
	withFakeScheduler(t, func(
		eng *engine.Engine, timer *fakeTimer, now time.Time,
	) {
		done := make(chan struct{}, 1)

		eng.ScheduleTask(
			[]string{"sched", "run"},
			now.Add(40*time.Millisecond),
			func() error {
				done <- struct{}{}
				return nil
			},
		)
		delay := timer.WaitReset(t)
		assert.Equal(t, 40*time.Millisecond, delay)
		timer.Fire(now)

		select {
		case <-done:
		case <-time.After(schedulerWaitTimeout):
			t.Fatal("scheduled task did not run")
		}
	})
}

func TestScheduleTaskReplacesSamePath(t *testing.T) {
	withFakeScheduler(t, func(
		eng *engine.Engine, timer *fakeTimer, now time.Time,
	) {
		var firstRuns atomic.Int32
		var secondRuns atomic.Int32
		secondDone := make(chan struct{}, 1)
		path := []string{"sched", "replace"}

		eng.ScheduleTask(path, now.Add(300*time.Millisecond),
			func() error {
				firstRuns.Add(1)
				return nil
			},
		)
		assert.Equal(t, 300*time.Millisecond, timer.WaitReset(t))

		eng.ScheduleTask(path, now.Add(40*time.Millisecond),
			func() error {
				secondRuns.Add(1)
				secondDone <- struct{}{}
				return nil
			},
		)
		assert.Equal(t, 40*time.Millisecond, timer.WaitReset(t))
		timer.Fire(now)

		select {
		case <-secondDone:
		case <-time.After(schedulerWaitTimeout):
			t.Fatal("replacement task did not run")
		}
		assert.Equal(t, int32(0), firstRuns.Load())
		assert.Equal(t, int32(1), secondRuns.Load())
	})
}

func TestCancelTask(t *testing.T) {
	withFakeScheduler(t, func(
		eng *engine.Engine, timer *fakeTimer, now time.Time,
	) {
		var ran atomic.Bool
		done := make(chan struct{}, 1)

		path := []string{"sched", "cancel", "one"}
		eng.ScheduleTask(path, now.Add(100*time.Millisecond),
			func() error {
				ran.Store(true)
				done <- struct{}{}
				return nil
			},
		)
		assert.Equal(t, 100*time.Millisecond, timer.WaitReset(t))
		eng.CancelTask(path)
		timer.WaitStop(t)
		timer.Fire(now)

		select {
		case <-done:
			t.Fatal("cancelled task ran")
		case <-time.After(100 * time.Millisecond):
		}
		assert.False(t, ran.Load())
	})
}

func TestCancelPrefixedTasks(t *testing.T) {
	withFakeScheduler(t, func(
		eng *engine.Engine, timer *fakeTimer, now time.Time,
	) {
		var cancelledRuns atomic.Int32
		var activeRuns atomic.Int32
		activeDone := make(chan struct{}, 1)

		cancelledPrefix := []string{"sched", "prefix", "cancelled"}
		eng.ScheduleTask(
			[]string{"sched", "prefix", "cancelled", "a"},
			now.Add(100*time.Millisecond),
			func() error {
				cancelledRuns.Add(1)
				return nil
			},
		)
		eng.ScheduleTask(
			[]string{"sched", "prefix", "cancelled", "b"},
			now.Add(100*time.Millisecond),
			func() error {
				cancelledRuns.Add(1)
				return nil
			},
		)

		eng.ScheduleTask(
			[]string{"sched", "prefix", "active", "c"},
			now.Add(100*time.Millisecond),
			func() error {
				activeRuns.Add(1)
				activeDone <- struct{}{}
				return nil
			},
		)
		assert.Equal(t, 100*time.Millisecond, timer.WaitReset(t))
		assert.Equal(t, 100*time.Millisecond, timer.WaitReset(t))
		assert.Equal(t, 100*time.Millisecond, timer.WaitReset(t))
		timer.DrainResets()

		eng.CancelPrefixedTasks(cancelledPrefix)
		assert.Equal(t, 100*time.Millisecond, timer.WaitReset(t))
		timer.Fire(now)

		select {
		case <-activeDone:
		case <-time.After(schedulerWaitTimeout):
			t.Fatal("active task did not run")
		}
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

func withFakeScheduler(
	t *testing.T,
	fn func(*engine.Engine, *fakeTimer, time.Time),
) {
	t.Helper()
	helpers.WithEngine(t, func(eng *engine.Engine) {
		now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
		eng.SetClock(func() time.Time { return now })
		tf := newFakeTimerFactory()
		eng.SetTimerConstructor(tf.NewTimer)
		assert.NoError(t, eng.Start())
		timer := tf.WaitTimer(t)
		timer.DrainResets()
		timer.DrainStops()
		fn(eng, timer, now)
	})
}

type fakeTimerFactory struct {
	created chan *fakeTimer
}

func newFakeTimerFactory() *fakeTimerFactory {
	return &fakeTimerFactory{
		created: make(chan *fakeTimer, 1),
	}
}

func (f *fakeTimerFactory) NewTimer(delay time.Duration) engine.Timer {
	timer := newFakeTimer(delay)
	select {
	case f.created <- timer:
	default:
	}
	return timer
}

func (f *fakeTimerFactory) WaitTimer(t *testing.T) *fakeTimer {
	t.Helper()
	select {
	case timer := <-f.created:
		return timer
	case <-time.After(schedulerWaitTimeout):
		t.Fatal("scheduler timer was not created")
		return nil
	}
}

type fakeTimer struct {
	ch      chan time.Time
	resets  chan time.Duration
	stops   chan struct{}
	stopped atomic.Bool
}

func newFakeTimer(delay time.Duration) *fakeTimer {
	timer := &fakeTimer{
		ch:     make(chan time.Time, 1),
		resets: make(chan time.Duration, 16),
		stops:  make(chan struct{}, 16),
	}
	_ = delay
	return timer
}

func (t *fakeTimer) Channel() <-chan time.Time {
	return t.ch
}

func (t *fakeTimer) Reset(delay time.Duration) bool {
	t.stopped.Store(false)
	drainTimeChan(t.ch)
	t.resets <- delay
	return true
}

func (t *fakeTimer) Stop() bool {
	alreadyStopped := t.stopped.Load()
	t.stopped.Store(true)
	drainTimeChan(t.ch)
	t.stops <- struct{}{}
	return !alreadyStopped
}

func (t *fakeTimer) Fire(at time.Time) {
	if t.stopped.Load() {
		return
	}
	select {
	case t.ch <- at:
	default:
	}
}

func (t *fakeTimer) WaitReset(test *testing.T) time.Duration {
	test.Helper()
	select {
	case delay := <-t.resets:
		return delay
	case <-time.After(schedulerWaitTimeout):
		test.Fatal("scheduler timer reset not observed")
		return 0
	}
}

func (t *fakeTimer) WaitStop(test *testing.T) {
	test.Helper()
	select {
	case <-t.stops:
	case <-time.After(schedulerWaitTimeout):
		test.Fatal("scheduler timer stop not observed")
	}
}

func (t *fakeTimer) DrainResets() {
	for {
		select {
		case <-t.resets:
		default:
			return
		}
	}
}

func (t *fakeTimer) DrainStops() {
	for {
		select {
		case <-t.stops:
		default:
			return
		}
	}
}

func drainTimeChan(ch <-chan time.Time) {
	select {
	case <-ch:
	default:
	}
}
