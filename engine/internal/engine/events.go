package engine

import (
	"log/slog"
	"sync"

	"github.com/kode4food/caravan"
	"github.com/kode4food/caravan/topic"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// EventQueue executes queued engine events sequentially
	EventQueue struct {
		prod      topic.Producer[event]
		cons      topic.Consumer[event]
		handler   EventHandler
		stop      chan struct{}
		wg        sync.WaitGroup
		startOnce sync.Once
		flushOnce sync.Once
		stopOnce  sync.Once
	}

	EventHandler func(api.EventType, any) error

	event struct {
		typ  api.EventType
		data any
	}
)

// NewEventQueue creates a new engine event queue
func NewEventQueue(handler EventHandler) *EventQueue {
	queue := caravan.NewTopic[event]()
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
					t.handleEvent(ev)
				}
			}
		})
	})
}

// Enqueue adds an engine event to the queue
func (t *EventQueue) Enqueue(typ api.EventType, data any) {
	t.prod.Send() <- event{
		typ:  typ,
		data: data,
	}
}

// Flush waits for queued events to complete and stops the queue
func (t *EventQueue) Flush() {
	t.stopOnce.Do(func() {
		close(t.stop)
	})
	t.wg.Wait()
	t.flushOnce.Do(t.flush)
}

func (t *EventQueue) flush() {
	for {
		select {
		case ev, ok := <-t.cons.Receive():
			if !ok {
				t.close()
				return
			}
			t.handleEvent(ev)
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

func (t *EventQueue) handleEvent(ev event) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Engine event panic",
				slog.Any("panic", r))
		}
	}()
	if err := t.handler(ev.typ, ev.data); err != nil {
		slog.Error("Failed to raise engine event",
			slog.String("event_type", string(ev.typ)),
			slog.Any("error", err))
	}
}
