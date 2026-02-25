package engine_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const eventTimeout = 3 * time.Second

func TestEventsOrdered(t *testing.T) {
	var mu sync.Mutex
	var order []int
	done := make(chan struct{})

	runner := engine.NewEventQueue(
		func(batch []engine.QueueEvent) error {
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
	)
	runner.Start()
	t.Cleanup(runner.Flush)

	runner.Enqueue(api.EventTypeFlowActivated, 1)
	runner.Enqueue(api.EventTypeFlowActivated, 2)
	runner.Enqueue(api.EventTypeFlowActivated, 3)

	select {
	case <-done:
	case <-time.After(eventTimeout):
		assert.Fail(t, "timed out waiting for events")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int{1, 2, 3}, order)
}

func TestEventsHandlerError(t *testing.T) {
	done := make(chan struct{})
	var mu sync.Mutex
	calls := 0

	runner := engine.NewEventQueue(
		func(batch []engine.QueueEvent) error {
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
	)
	runner.Start()
	t.Cleanup(runner.Flush)

	runner.Enqueue(api.EventTypeFlowActivated, 1)

	select {
	case <-done:
	case <-time.After(eventTimeout):
		assert.Fail(t, "timed out waiting for events")
	}
}

func TestEventsHandlerPanic(t *testing.T) {
	done := make(chan struct{})
	var mu sync.Mutex
	calls := 0

	runner := engine.NewEventQueue(
		func(batch []engine.QueueEvent) error {
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
	)
	runner.Start()
	t.Cleanup(runner.Flush)

	runner.Enqueue(api.EventTypeFlowActivated, 1)

	select {
	case <-done:
	case <-time.After(eventTimeout):
		assert.Fail(t, "timed out waiting for events")
	}
}

func TestEventsCancel(t *testing.T) {
	handled := make(chan struct{}, 1)

	runner := engine.NewEventQueue(
		func(batch []engine.QueueEvent) error {
			handled <- struct{}{}
			return nil
		},
	)
	runner.Start()

	runner.Cancel()
	runner.Cancel()

	select {
	case <-handled:
		t.Fatal("unexpected event handled after cancel")
	default:
	}
}
