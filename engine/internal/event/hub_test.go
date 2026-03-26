package event_test

import (
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/event"
)

const (
	eventWait = time.Second
	idleWait  = 100 * time.Millisecond
)

func TestHubPublishOrder(t *testing.T) {
	h := event.NewHub()
	consumer := h.NewConsumer()
	defer consumer.Close()

	evs := []*timebox.Event{
		newEvent("incremented", timebox.NewAggregateID("flow", "a"), 0),
		newEvent("incremented", timebox.NewAggregateID("flow", "a"), 1),
		newEvent("incremented", timebox.NewAggregateID("flow", "a"), 2),
	}

	ch := consumer.Receive()
	h.Publish(evs...)

	for i, want := range evs {
		select {
		case got := <-ch:
			assert.Equal(t, want.Sequence, got.Sequence, "index %d", i)
			assert.Equal(t, want.Type, got.Type, "index %d", i)
		case <-time.After(eventWait):
			t.Fatalf("timeout waiting for event %d", i)
		}
	}
}

func TestHubAggregatePrefix(t *testing.T) {
	h := event.NewHub()
	consumer := h.NewAggregateConsumer(timebox.NewAggregateID("flow"))
	defer consumer.Close()

	ch := consumer.Receive()
	h.Publish(
		newEvent("incremented", timebox.NewAggregateID("flow", "a"), 0),
		newEvent("incremented", timebox.NewAggregateID("flow", "b"), 1),
		newEvent("incremented", timebox.NewAggregateID("engine", "x"), 2),
	)

	for range 2 {
		select {
		case ev := <-ch:
			assert.Equal(t, timebox.ID("flow"), ev.AggregateID[0])
		case <-time.After(eventWait):
			t.Fatal("timeout waiting for flow event")
		}
	}

	select {
	case ev := <-ch:
		t.Fatalf("unexpected event for %v", ev.AggregateID)
	case <-time.After(idleWait):
	}
}

func TestHubTypeFilterWithPrefix(t *testing.T) {
	h := event.NewHub()
	consumer := h.NewAggregateConsumer(
		timebox.NewAggregateID("flow"), "incremented",
	)
	defer consumer.Close()

	ch := consumer.Receive()
	h.Publish(
		newEvent("incremented", timebox.NewAggregateID("flow", "a"), 0),
		newEvent("decremented", timebox.NewAggregateID("flow", "a"), 1),
		newEvent("incremented", timebox.NewAggregateID("engine", "x"), 2),
	)

	select {
	case ev := <-ch:
		assert.Equal(t, timebox.EventType("incremented"), ev.Type)
		assert.Equal(t, timebox.ID("flow"), ev.AggregateID[0])
	case <-time.After(eventWait):
		t.Fatal("timeout waiting for filtered event")
	}

	select {
	case ev := <-ch:
		t.Fatalf("unexpected event for %v", ev.AggregateID)
	case <-time.After(idleWait):
	}
}

func TestHubTypeOnly(t *testing.T) {
	h := event.NewHub()
	consumer := h.NewTypeConsumer("incremented")
	defer consumer.Close()

	ch := consumer.Receive()
	h.Publish(
		newEvent("incremented", timebox.NewAggregateID("flow", "a"), 0),
		newEvent("decremented", timebox.NewAggregateID("engine", "x"), 1),
	)

	select {
	case ev := <-ch:
		assert.Equal(t, timebox.EventType("incremented"), ev.Type)
	case <-time.After(eventWait):
		t.Fatal("timeout waiting for increment event")
	}

	select {
	case ev := <-ch:
		t.Fatalf("unexpected event for %v", ev.AggregateID)
	case <-time.After(idleWait):
	}
}

func TestHubUnsubscribe(t *testing.T) {
	h := event.NewHub()
	consumer := h.NewAggregateConsumer(timebox.NewAggregateID("flow"))
	ch := consumer.Receive()

	h.Publish(newEvent("incremented", timebox.NewAggregateID("flow", "a"), 0))
	select {
	case <-ch:
	case <-time.After(eventWait):
		t.Fatal("timeout waiting for first event")
	}

	consumer.Close()
	h.Publish(newEvent("incremented", timebox.NewAggregateID("flow", "a"), 1))

	select {
	case _, ok := <-ch:
		assert.False(t, ok)
	case <-time.After(eventWait):
		t.Fatal("timeout waiting for consumer close")
	}
}

func TestHubMultiplePrefixes(t *testing.T) {
	h := event.NewHub()
	consumer := h.NewAggregatesConsumer([]timebox.AggregateID{
		timebox.NewAggregateID("flow"),
		timebox.NewAggregateID("engine"),
	})
	defer consumer.Close()

	ch := consumer.Receive()
	h.Publish(
		newEvent("incremented", timebox.NewAggregateID("flow", "a"), 0),
		newEvent("incremented", timebox.NewAggregateID("engine", "x"), 1),
	)

	got := make([]timebox.ID, 0, 2)
	for range 2 {
		select {
		case ev := <-ch:
			got = append(got, ev.AggregateID[0])
		case <-time.After(eventWait):
			t.Fatal("timeout waiting for event")
		}
	}
	assert.ElementsMatch(t, []timebox.ID{"flow", "engine"}, got)
}

func TestHubNoSubscribers(t *testing.T) {
	h := event.NewHub()
	h.Publish(newEvent("incremented", timebox.NewAggregateID("flow", "a"), 0))
}

func TestHubPublishNonBlocking(t *testing.T) {
	h := event.NewHub()
	consumer := h.NewConsumer()
	defer consumer.Close()

	done := make(chan struct{})
	go func() {
		h.Publish(
			newEvent("incremented", timebox.NewAggregateID("flow", "a"), 0),
		)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(eventWait):
		t.Fatal("publish blocked on inactive consumer")
	}
}

func newEvent(
	typ timebox.EventType, agg timebox.AggregateID, seq int64,
) *timebox.Event {
	return &timebox.Event{
		Type:        typ,
		AggregateID: agg,
		Sequence:    seq,
	}
}
