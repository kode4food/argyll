package engine

import (
	"testing"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/stretchr/testify/assert"
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

func TestRetryQueueStopPreventsNewPushes(t *testing.T) {
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
