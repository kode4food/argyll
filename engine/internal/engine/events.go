package engine

import (
	"log/slog"
	"sync"

	"github.com/kode4food/caravan"
	"github.com/kode4food/caravan/message"
	"github.com/kode4food/caravan/topic"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// EventQueue executes queued engine events sequentially
	EventQueue struct {
		queue    topic.Topic[Event]
		prod     topic.Producer[Event]
		cons     topic.Consumer[Event]
		handler  EventHandler
		stop     chan struct{}
		stopOnce sync.Once
		started  sync.Once
		runWG    sync.WaitGroup
	}

	EventHandler func(api.EventType, any) error

	Event struct {
		eventType api.EventType
		data      any
	}
)

// NewEventQueue creates a new engine event queue
func NewEventQueue(handler EventHandler) *EventQueue {
	queue := caravan.NewTopic[Event]()
	tr := &EventQueue{
		queue:   queue,
		prod:    queue.NewProducer(),
		cons:    queue.NewConsumer(),
		handler: handler,
		stop:    make(chan struct{}),
	}
	return tr
}

// Start begins processing queued engine events
func (t *EventQueue) Start() {
	t.started.Do(func() {
		t.runWG.Go(func() {
			for {
				select {
				case <-t.stop:
					return
				case ev, ok := <-t.cons.Receive():
					if !ok {
						return
					}
					t.runTask(ev)
				}
			}
		})
	})
}

// Enqueue adds an engine event to the queue
func (t *EventQueue) Enqueue(typ api.EventType, data any) {
	if typ == "" {
		return
	}
	message.Send(t.prod, Event{
		eventType: typ,
		data:      data,
	})
}

// Flush waits for queued events to complete and stops the queue
func (t *EventQueue) Flush() {
	t.stopOnce.Do(func() {
		close(t.stop)
	})
	t.runWG.Wait()
	for {
		select {
		case ev, ok := <-t.cons.Receive():
			if !ok {
				t.prod.Close()
				t.cons.Close()
				return
			}
			t.runTask(ev)
		default:
			t.prod.Close()
			t.cons.Close()
			return
		}
	}
}

func (t *EventQueue) runTask(ev Event) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Engine event panic",
				slog.Any("panic", r))
		}
	}()
	if err := t.handler(ev.eventType, ev.data); err != nil {
		slog.Error("Failed to raise engine event",
			slog.String("event_type", string(ev.eventType)),
			slog.Any("error", err))
	}
}
