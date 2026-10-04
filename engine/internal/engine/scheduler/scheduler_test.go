package scheduler_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"

	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
)

func TestPaths(t *testing.T) {
	store := newStore(t)
	runner, err := scheduler.New(scheduler.Config{
		Store: store,
		Emitter: func(context.Context, *scheduler.Delivery) error {
			return nil
		},
		Clock: scheduler.Now,
	})
	assert.NoError(t, err)
	event := newEvent()
	at := scheduler.Now().Add(time.Hour)

	assert.NoError(t, store.Transact(func(tx *timebox.Transaction) error {
		for _, path := range [][]string{
			{"flow", "one"},
			{"flow", "two"},
			{"flow", "one/two"},
			{"flowish"},
			{"other"},
		} {
			if err := runner.Schedule(tx, path, at, event); err != nil {
				return err
			}
		}
		return tx.Schedule("foreign", at, event)
	}))

	assert.NoError(t, store.Transact(func(tx *timebox.Transaction) error {
		return runner.CancelPrefix(tx, []string{"flow"})
	}))
	assert.Equal(t, 3, activeScheduleCount(t, store))

	assert.NoError(t, store.Transact(func(tx *timebox.Transaction) error {
		return runner.Cancel(tx, []string{"other"})
	}))
	assert.Equal(t, 2, activeScheduleCount(t, store))
	foreign, err := store.GetEvents(
		timebox.NewAggregateID(timebox.ScheduleAggregateType, "foreign"), 0,
	)
	assert.NoError(t, err)
	if assert.NotEmpty(t, foreign) {
		assert.Equal(t, timebox.ScheduleChanged, foreign[len(foreign)-1].Type)
	}
}

func TestDelivery(t *testing.T) {
	store := newStore(t)
	delivered := make(chan *timebox.Event, 1)
	runner, err := scheduler.New(scheduler.Config{
		Store: store,
		Emitter: func(
			_ context.Context, delivery *scheduler.Delivery,
		) error {
			err := store.Transact(func(tx *timebox.Transaction) error {
				return delivery.Consume(tx)
			})
			if err == nil {
				delivered <- delivery.Event()
			}
			return err
		},
		Clock: scheduler.Now,
	})
	assert.NoError(t, err)
	event := newEvent()
	assert.NoError(t, store.Transact(func(tx *timebox.Transaction) error {
		return runner.Schedule(
			tx, []string{"due"}, scheduler.Now().Add(-time.Second), event,
		)
	}))

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	var got *timebox.Event
	assert.Eventually(t, func() bool {
		select {
		case got = <-delivered:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	assert.Equal(t, event, got)
	cancel()
	assert.ErrorIs(t, <-done, context.Canceled)
}

func newStore(t *testing.T) *timebox.Store {
	t.Helper()
	backend := memory.Open()
	t.Cleanup(func() {
		assert.NoError(t, backend.Close())
	})
	store, err := backend.NewStore(timebox.Config{})
	assert.NoError(t, err)
	return store
}

func newEvent() *timebox.Event {
	return &timebox.Event{
		AggregateID: timebox.AggregateID{Type: "flow", Key: "test"},
		Type:        "test",
	}
}

func activeScheduleCount(t *testing.T, store *timebox.Store) int {
	t.Helper()
	ids, err := store.ListAggregates(timebox.ScheduleAggregateType)
	if !assert.NoError(t, err) {
		return 0
	}
	res := 0
	for _, id := range ids {
		evs, err := store.GetEvents(id, 0)
		if !assert.NoError(t, err) {
			return 0
		}
		if len(evs) > 0 && evs[len(evs)-1].Type == timebox.ScheduleChanged {
			res++
		}
	}
	return res
}
