package event

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
	// Queue executes queued engine events sequentially in bounded batches
	Queue struct {
		prod        topic.Producer[Event]
		cons        topic.Consumer[Event]
		handler     Handler
		stop        chan struct{}
		batchSize   int
		wg          sync.WaitGroup
		startOnce   sync.Once
		stopOnce    sync.Once
		cleanupOnce sync.Once
	}

	// Handler processes a batch of engine events in a single execution
	Handler func([]Event) error

	// Event is an engine event envelope
	Event struct {
		Type api.EventType
		Data any
	}
)

var ErrHandlerPanicked = errors.New("event handler panicked")

const (
	maxRetries = 3
	retryDelay = 100 * time.Millisecond
)

// NewQueue creates a new engine event queue with the provided batch size
func NewQueue(handler Handler, batchSize int) *Queue {
	queue := caravan.NewTopic[Event]()
	return &Queue{
		prod:      queue.NewProducer(),
		cons:      queue.NewConsumer(),
		handler:   handler,
		stop:      make(chan struct{}),
		batchSize: batchSize,
	}
}

// Start begins processing queued engine events
func (q *Queue) Start() {
	q.startOnce.Do(func() {
		q.wg.Go(func() {
			for {
				select {
				case <-q.stop:
					return
				case ev, ok := <-q.cons.Receive():
					if !ok {
						return
					}
					q.handleBatch(q.collectBatch(ev))
				}
			}
		})
	})
}

// Enqueue adds an engine event to the queue
func (q *Queue) Enqueue(typ api.EventType, data any) {
	q.prod.Send() <- Event{
		Type: typ,
		Data: data,
	}
}

// Flush waits for queued events to complete and stops the queue
func (q *Queue) Flush() {
	q.stopOnce.Do(func() {
		close(q.stop)
	})
	q.wg.Wait()
	q.cleanupOnce.Do(q.flush)
}

// Cancel immediately stops the queue without processing remaining events
func (q *Queue) Cancel() {
	q.stopOnce.Do(func() {
		close(q.stop)
	})
	q.wg.Wait()
	q.cleanupOnce.Do(q.close)
}

func (q *Queue) collectBatch(first Event) []Event {
	batch := []Event{first}
	for len(batch) < q.batchSize {
		select {
		case ev, ok := <-q.cons.Receive():
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

func (q *Queue) flush() {
	for {
		select {
		case ev, ok := <-q.cons.Receive():
			if !ok {
				q.close()
				return
			}
			q.handleBatch(q.collectBatch(ev))
		default:
			q.close()
			return
		}
	}
}

func (q *Queue) close() {
	q.prod.Close()
	q.cons.Close()
}

func (q *Queue) handleBatch(batch []Event) {
	for attempt := range maxRetries {
		err := q.tryHandleBatch(batch)
		if err == nil {
			return
		}
		slog.Error("Engine event batch failed",
			slog.Int("batch_size", len(batch)),
			slog.Int("attempt", attempt+1),
			slog.Int("max_attempts", maxRetries),
			slog.Any("error", err))
		if attempt < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	slog.Error("Engine event batch permanently failed",
		slog.Int("batch_size", len(batch)))
}

func (q *Queue) tryHandleBatch(batch []Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrHandlerPanicked, r)
		}
	}()
	return q.handler(batch)
}
