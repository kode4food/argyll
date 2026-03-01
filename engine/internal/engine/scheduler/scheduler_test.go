package scheduler_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	testTimerConstructor struct {
		created chan *fakeTimer
	}

	fakeTimer struct {
		ch      chan time.Time
		resets  chan time.Duration
		stops   chan struct{}
		stopped atomic.Bool
	}
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
		assertEventuallyEqual(t, 40*time.Millisecond, timer.WaitReset)
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
		timer.DrainResets()

		eng.CancelPrefixedTasks(cancelledPrefix)
		assertEventuallyEqual(t, 100*time.Millisecond, timer.WaitReset)
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

func TestFlowWithoutTimeoutOptionalsSchedulesNoTimeoutTasks(t *testing.T) {
	withFakeScheduler(t, func(
		eng *engine.Engine, timer *fakeTimer, _ time.Time,
	) {
		step := &api.Step{
			ID:   "script-step",
			Name: "Script Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return { result = input }",
			},
			Attributes: api.AttributeSpecs{
				"input": {
					Role: api.RoleRequired,
					Type: api.TypeString,
				},
				"result": {
					Role: api.RoleOutput,
					Type: api.TypeString,
				},
			},
		}
		assert.NoError(t, eng.RegisterStep(step))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-no-timeouts")
		assert.NoError(t, eng.StartFlow(flowID, plan,
			flowopt.WithInit(api.Args{"input": "ok"}),
		))

		assertFlowEventuallyCompleted(t, eng, flowID)
		assertNoSchedulerActivity(t, timer)
	})
}

func (c *testTimerConstructor) NewTimer(
	delay time.Duration,
) scheduler.Timer {
	timer := newFakeTimer(delay)
	select {
	case c.created <- timer:
	default:
	}
	return timer
}

func (c *testTimerConstructor) WaitTimer(t *testing.T) *fakeTimer {
	t.Helper()
	select {
	case timer := <-c.created:
		return timer
	case <-time.After(schedulerWaitTimeout):
		t.Fatal("scheduler timer was not created")
		return nil
	}
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

func assertNoSchedulerActivity(t *testing.T, timer *fakeTimer) {
	t.Helper()

	select {
	case delay := <-timer.resets:
		t.Fatalf("unexpected scheduler reset: %s", delay)
	case <-timer.stops:
		t.Fatal("unexpected scheduler stop")
	case <-time.After(100 * time.Millisecond):
	}
}

func assertEventuallyEqual(
	t *testing.T, expected time.Duration,
	wait func(*testing.T) time.Duration,
) {
	t.Helper()
	assert.Equal(t, expected, wait(t))
}

func assertFlowEventuallyCompleted(
	t *testing.T, eng *engine.Engine, flowID api.FlowID,
) {
	t.Helper()

	deadline := time.Now().Add(schedulerWaitTimeout)
	for time.Now().Before(deadline) {
		flow, err := eng.GetFlowState(flowID)
		if err == nil && flow.Status == api.FlowCompleted {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("flow %s did not complete", flowID)
}

func withFakeScheduler(
	t *testing.T, fn func(*engine.Engine, *fakeTimer, time.Time),
) {
	t.Helper()
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	tc := newTestTimerConstructor()
	helpers.WithEngineDeps(t, engine.Dependencies{
		Clock:            func() time.Time { return now },
		TimerConstructor: tc.NewTimer,
	}, func(eng *engine.Engine) {
		assert.NoError(t, eng.Start())
		timer := tc.WaitTimer(t)
		timer.DrainResets()
		timer.DrainStops()
		fn(eng, timer, now)
	})
}

func newTestTimerConstructor() *testTimerConstructor {
	return &testTimerConstructor{
		created: make(chan *fakeTimer, 1),
	}
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

func drainTimeChan(ch <-chan time.Time) {
	select {
	case <-ch:
	default:
	}
}
