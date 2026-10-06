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
		Processor: func(*timebox.Transaction, *timebox.Message) error {
			return nil
		},
		Clock: scheduler.Now,
	})
	assert.NoError(t, err)
	message := newMessage()
	at := scheduler.Now().Add(time.Hour)

	assert.NoError(t, store.Transact(func(tx *timebox.Transaction) error {
		for _, path := range [][]string{
			{"flow", "one"},
			{"flow", "two"},
			{"flow", "one/two"},
			{"flowish"},
			{"other"},
		} {
			if err := runner.Schedule(tx, path, at, message); err != nil {
				return err
			}
		}
		return tx.Schedule("foreign", at, message)
	}))

	assert.NoError(t, store.Transact(func(tx *timebox.Transaction) error {
		return runner.CancelPrefix(tx, []string{"flow"})
	}))
	assert.Equal(t, 3, activeScheduleCount(t, store))

	assert.NoError(t, store.Transact(func(tx *timebox.Transaction) error {
		return runner.Cancel(tx, []string{"other"})
	}))
	assert.Equal(t, 2, activeScheduleCount(t, store))
	foreign, err := store.LoadSchedule("foreign")
	assert.NoError(t, err)
	assert.NotNil(t, foreign)
}

func TestDelivery(t *testing.T) {
	store := newStore(t)
	delivered := make(chan *timebox.Message, 1)
	runner, err := scheduler.New(scheduler.Config{
		Store: store,
		Processor: func(_ *timebox.Transaction, msg *timebox.Message) error {
			delivered <- msg
			return nil
		},
		Clock: scheduler.Now,
	})
	assert.NoError(t, err)
	message := newMessage()
	assert.NoError(t, store.Transact(func(tx *timebox.Transaction) error {
		return runner.Schedule(
			tx, []string{"due"}, scheduler.Now().Add(-time.Second), message,
		)
	}))

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	var got *timebox.Message
	assert.Eventually(t, func() bool {
		select {
		case got = <-delivered:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	assert.Equal(t, message, got)
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

func newMessage() *timebox.Message {
	return &timebox.Message{
		AggregateID: timebox.AggregateID{Type: "flow", Key: "test"},
		Type:        "test",
	}
}

func activeScheduleCount(t *testing.T, store *timebox.Store) int {
	t.Helper()
	schedules, err := store.ListSchedules(time.Time{})
	if !assert.NoError(t, err) {
		return 0
	}
	return len(schedules)
}
