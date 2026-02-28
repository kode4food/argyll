package event_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/event"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const eventTimeout = 3 * time.Second

func TestQueueOrdered(t *testing.T) {
	var mu sync.Mutex
	var order []int
	done := make(chan struct{})

	q := event.NewQueue(
		func(batch []event.Event) error {
			for _, ev := range batch {
				if ev.Type == "" {
					return errors.New("missing event type")
				}
				value, ok := ev.Data.(int)
				if !ok {
					return errors.New("invalid event data")
				}
				mu.Lock()
				order = append(order, value)
				if value == 3 {
					close(done)
				}
				mu.Unlock()
			}
			return nil
		},
		128,
	)
	q.Start()
	t.Cleanup(q.Flush)

	q.Enqueue(api.EventTypeFlowActivated, 1)
	q.Enqueue(api.EventTypeFlowActivated, 2)
	q.Enqueue(api.EventTypeFlowActivated, 3)

	select {
	case <-done:
	case <-time.After(eventTimeout):
		assert.Fail(t, "timed out waiting for events")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int{1, 2, 3}, order)
}

func TestQueueHandlerError(t *testing.T) {
	done := make(chan struct{})
	var mu sync.Mutex
	calls := 0

	q := event.NewQueue(
		func(batch []event.Event) error {
			mu.Lock()
			calls++
			n := calls
			mu.Unlock()
			if n == 1 {
				return errors.New("handler error")
			}
			close(done)
			return nil
		},
		128,
	)
	q.Start()
	t.Cleanup(q.Flush)

	q.Enqueue(api.EventTypeFlowActivated, 1)

	select {
	case <-done:
	case <-time.After(eventTimeout):
		assert.Fail(t, "timed out waiting for events")
	}
}

func TestQueueHandlerPanic(t *testing.T) {
	done := make(chan struct{})
	var mu sync.Mutex
	calls := 0

	q := event.NewQueue(
		func(batch []event.Event) error {
			mu.Lock()
			calls++
			n := calls
			mu.Unlock()
			if n == 1 {
				panic("test panic")
			}
			close(done)
			return nil
		},
		128,
	)
	q.Start()
	t.Cleanup(q.Flush)

	q.Enqueue(api.EventTypeFlowActivated, 1)

	select {
	case <-done:
	case <-time.After(eventTimeout):
		assert.Fail(t, "timed out waiting for events")
	}
}

func TestQueueCancel(t *testing.T) {
	handled := make(chan struct{}, 1)

	q := event.NewQueue(
		func(batch []event.Event) error {
			handled <- struct{}{}
			return nil
		},
		128,
	)
	q.Start()

	q.Cancel()
	q.Cancel()

	select {
	case <-handled:
		t.Fatal("unexpected event handled after cancel")
	default:
	}
}
