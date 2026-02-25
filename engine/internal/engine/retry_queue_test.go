package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRetryQueueBasicOperations(t *testing.T) {
	rq := NewRetryQueue()
	defer rq.Stop()

	_, ok := rq.Peek()
	assert.False(t, ok)
	assert.Equal(t, 0, rq.Len())

	now := time.Now()
	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(time.Second),
	})

	assert.Equal(t, 1, rq.Len())
	peekTime, ok := rq.Peek()
	assert.True(t, ok)
	assert.Equal(t, now.Add(time.Second).Unix(), peekTime.Unix())
}

func TestRetryQueueOrdering(t *testing.T) {
	rq := NewRetryQueue()
	defer rq.Stop()

	now := time.Now()

	rq.Push(&RetryItem{
		FlowID:      "flow-3",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(3 * time.Second),
	})
	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(time.Second),
	})
	rq.Push(&RetryItem{
		FlowID:      "flow-2",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(2 * time.Second),
	})

	assert.Equal(t, 3, rq.Len())

	peekTime, ok := rq.Peek()
	assert.True(t, ok)
	assert.Equal(t, now.Add(time.Second).Unix(), peekTime.Unix())

	ready := rq.PopReady(now.Add(1500 * time.Millisecond))
	assert.Len(t, ready, 1)
	assert.Equal(t, api.FlowID("flow-1"), ready[0].FlowID)
	assert.Equal(t, 2, rq.Len())

	ready = rq.PopReady(now.Add(2500 * time.Millisecond))
	assert.Len(t, ready, 1)
	assert.Equal(t, api.FlowID("flow-2"), ready[0].FlowID)
	assert.Equal(t, 1, rq.Len())
}

func TestRetryQueuePopReadyMultiple(t *testing.T) {
	rq := NewRetryQueue()
	defer rq.Stop()

	now := time.Now()

	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now,
	})
	rq.Push(&RetryItem{
		FlowID:      "flow-2",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now,
	})
	rq.Push(&RetryItem{
		FlowID:      "flow-3",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(time.Hour),
	})

	ready := rq.PopReady(now)
	assert.Len(t, ready, 2)
	assert.Equal(t, 1, rq.Len())
}

func TestRetryQueueRemove(t *testing.T) {
	rq := NewRetryQueue()
	defer rq.Stop()

	now := time.Now()

	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now,
	})
	rq.Push(&RetryItem{
		FlowID:      "flow-2",
		StepID:      "step-2",
		Token:       "token-2",
		NextRetryAt: now.Add(time.Second),
	})

	assert.Equal(t, 2, rq.Len())

	rq.Remove("flow-1", "step-1", "token-1")
	assert.Equal(t, 1, rq.Len())

	peekTime, ok := rq.Peek()
	assert.True(t, ok)
	assert.Equal(t, now.Add(time.Second).Unix(), peekTime.Unix())

	rq.Remove("flow-99", "step-99", "token-99")
	assert.Equal(t, 1, rq.Len())
}

func TestRetryQueueRemoveFlow(t *testing.T) {
	rq := NewRetryQueue()
	defer rq.Stop()

	now := time.Now()

	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now,
	})
	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-2",
		Token:       "token-2",
		NextRetryAt: now.Add(time.Second),
	})
	rq.Push(&RetryItem{
		FlowID:      "flow-2",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(2 * time.Second),
	})

	assert.Equal(t, 3, rq.Len())

	rq.RemoveFlow("flow-1")
	assert.Equal(t, 1, rq.Len())

	ready := rq.PopReady(now.Add(3 * time.Second))
	assert.Len(t, ready, 1)
	assert.Equal(t, api.FlowID("flow-2"), ready[0].FlowID)
}

func TestRetryQueueUpdate(t *testing.T) {
	rq := NewRetryQueue()
	defer rq.Stop()

	now := time.Now()

	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(time.Hour),
	})

	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now,
	})

	assert.Equal(t, 1, rq.Len())

	peekTime, ok := rq.Peek()
	assert.True(t, ok)
	assert.Equal(t, now.Unix(), peekTime.Unix())
}

func TestRetryQueueUpdateHeadLater(t *testing.T) {
	rq := NewRetryQueue()
	defer rq.Stop()

	now := time.Now()
	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now,
	})
	rq.Push(&RetryItem{
		FlowID:      "flow-2",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(time.Second),
	})

	changed := rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(2 * time.Second),
	})

	assert.True(t, changed)
	peekTime, ok := rq.Peek()
	assert.True(t, ok)
	assert.Equal(t, now.Add(time.Second).Unix(), peekTime.Unix())
}

func TestRetryQueueStopPreventsPush(t *testing.T) {
	rq := NewRetryQueue()

	now := time.Now()
	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now,
	})

	assert.Equal(t, 1, rq.Len())

	rq.Stop()

	rq.Push(&RetryItem{
		FlowID:      "flow-2",
		StepID:      "step-2",
		Token:       "token-2",
		NextRetryAt: now,
	})

	assert.Equal(t, 1, rq.Len())
}

func TestRetryQueuePushChanged(t *testing.T) {
	rq := NewRetryQueue()
	defer rq.Stop()

	now := time.Now()
	changed := rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now,
	})
	assert.True(t, changed)

	changed = rq.Push(&RetryItem{
		FlowID:      "flow-2",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(time.Second),
	})
	assert.False(t, changed)

	changed = rq.Push(&RetryItem{
		FlowID:      "flow-2",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(2 * time.Second),
	})
	assert.False(t, changed)

	changed = rq.Push(&RetryItem{
		FlowID:      "flow-2",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now.Add(-time.Second),
	})
	assert.True(t, changed)
}

func TestRetryQueuePushStopped(t *testing.T) {
	rq := NewRetryQueue()
	rq.Stop()

	changed := rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: time.Now(),
	})

	assert.False(t, changed)
	assert.Equal(t, 0, rq.Len())
}

func TestRetryQueueStopIdempotent(t *testing.T) {
	rq := NewRetryQueue()
	rq.Stop()
	rq.Stop()
	rq.Stop()
}

func TestRetryQueueNotify(t *testing.T) {
	rq := NewRetryQueue()
	defer rq.Stop()

	now := time.Now()

	done := make(chan struct{})
	go func() {
		<-rq.Notify()
		close(done)
	}()

	rq.Push(&RetryItem{
		FlowID:      "flow-1",
		StepID:      "step-1",
		Token:       "token-1",
		NextRetryAt: now,
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("notification not received")
	}
}

func TestRequeueRetryItem(t *testing.T) {
	t.Run("schedules retry task when queue head changes", func(t *testing.T) {
		e := &Engine{
			ctx:        context.Background(),
			retryQueue: NewRetryQueue(),
			tasks:      make(chan Task, 4),
		}
		defer e.retryQueue.Stop()

		e.requeueRetryItem(&RetryItem{
			FlowID: "flow-1",
			StepID: "step-1",
			Token:  "token-1",
		})

		assert.Equal(t, 1, e.retryQueue.Len())
		select {
		case tsk := <-e.tasks:
			assert.NotNil(t, tsk.Func)
			assert.False(t, tsk.Deadline.IsZero())
		default:
			t.Fatal("expected retry task to be scheduled")
		}
	})

	t.Run("does not schedule when earlier retry exists", func(t *testing.T) {
		e := &Engine{
			ctx:        context.Background(),
			retryQueue: NewRetryQueue(),
			tasks:      make(chan Task, 4),
		}
		defer e.retryQueue.Stop()

		earlier := time.Now().Add(-10 * time.Second)
		e.retryQueue.Push(&RetryItem{
			FlowID:      "flow-0",
			StepID:      "step-0",
			Token:       "token-0",
			NextRetryAt: earlier,
		})

		e.requeueRetryItem(&RetryItem{
			FlowID: "flow-1",
			StepID: "step-1",
			Token:  "token-1",
		})

		assert.Equal(t, 2, e.retryQueue.Len())
		select {
		case <-e.tasks:
			t.Fatal("expected no retry task to be scheduled")
		default:
		}
	})
}
