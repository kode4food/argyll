package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kode4food/caravan"
	"github.com/kode4food/caravan/topic"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// EventQueue executes queued engine events sequentially, batching up to
	// maxBatchSize events per handler call
	EventQueue struct {
		prod        topic.Producer[QueueEvent]
		cons        topic.Consumer[QueueEvent]
		handler     EventHandler
		stop        chan struct{}
		wg          sync.WaitGroup
		startOnce   sync.Once
		stopOnce    sync.Once
		cleanupOnce sync.Once
	}

	// EventHandler processes a batch of engine events in a single execution
	EventHandler func([]QueueEvent) error

	// QueueEvent is an engine event envelope
	QueueEvent struct {
		Type api.EventType
		Data any
	}
)

var ErrEventHandlerPanicked = errors.New("event handler panicked")

const (
	maxBatchSize    = 128
	maxEventRetries = 3
	eventRetryDelay = 100 * time.Millisecond
)

// NewEventQueue creates a new engine event queue
func NewEventQueue(handler EventHandler) *EventQueue {
	queue := caravan.NewTopic[QueueEvent]()
	return &EventQueue{
		prod:    queue.NewProducer(),
		cons:    queue.NewConsumer(),
		handler: handler,
		stop:    make(chan struct{}),
	}
}

// Start begins processing queued engine events
func (t *EventQueue) Start() {
	t.startOnce.Do(func() {
		t.wg.Go(func() {
			for {
				select {
				case <-t.stop:
					return
				case ev, ok := <-t.cons.Receive():
					if !ok {
						return
					}
					t.handleBatch(t.collectBatch(ev))
				}
			}
		})
	})
}

// Enqueue adds an engine event to the queue
func (t *EventQueue) Enqueue(typ api.EventType, data any) {
	t.prod.Send() <- QueueEvent{
		Type: typ,
		Data: data,
	}
}

// Flush waits for queued events to complete and stops the queue
func (t *EventQueue) Flush() {
	t.stopOnce.Do(func() {
		close(t.stop)
	})
	t.wg.Wait()
	t.cleanupOnce.Do(t.flush)
}

// Cancel immediately stops the queue without processing remaining events
func (t *EventQueue) Cancel() {
	t.stopOnce.Do(func() {
		close(t.stop)
	})
	t.wg.Wait()
	t.cleanupOnce.Do(t.close)
}

func (t *EventQueue) collectBatch(first QueueEvent) []QueueEvent {
	batch := []QueueEvent{first}
	for len(batch) < maxBatchSize {
		select {
		case ev, ok := <-t.cons.Receive():
			if !ok {
				return batch
			}
			batch = append(batch, ev)
		default:
			return batch
		}
	}
	return batch
}

func (t *EventQueue) flush() {
	for {
		select {
		case ev, ok := <-t.cons.Receive():
			if !ok {
				t.close()
				return
			}
			t.handleBatch(t.collectBatch(ev))
		default:
			t.close()
			return
		}
	}
}

func (t *EventQueue) close() {
	t.prod.Close()
	t.cons.Close()
}

func (t *EventQueue) handleBatch(batch []QueueEvent) {
	for attempt := range maxEventRetries {
		err := t.tryHandleBatch(batch)
		if err == nil {
			return
		}
		slog.Error("Engine event batch failed",
			slog.Int("batch_size", len(batch)),
			slog.Int("attempt", attempt+1),
			slog.Int("max_attempts", maxEventRetries),
			slog.Any("error", err))
		if attempt < maxEventRetries-1 {
			time.Sleep(eventRetryDelay)
		}
	}
	slog.Error(
		"Engine event batch permanently failed; partition state may diverge",
		slog.Int("batch_size", len(batch)))
}

func (t *EventQueue) tryHandleBatch(batch []QueueEvent) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrEventHandlerPanicked, r)
		}
	}()
	return t.handler(batch)
}
