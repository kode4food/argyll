package events_test

import (
	"testing"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestMemoProjection(t *testing.T) {
	backend := memory.Open()
	defer func() { assert.NoError(t, backend.Close()) }()
	store, err := timebox.NewStore(backend)
	if !assert.NoError(t, err) {
		return
	}
	key, err := events.MemoKey(&api.Step{}, api.Args{})
	assert.NoError(t, err)
	exec := store.Executor(events.NewMemoState, events.MemoAppliers.Reduce)
	state, err := exec.Get(key)
	assert.NoError(t, err)
	assert.Nil(t, state)
	state, err = exec.Exec(key,
		func(_ *api.WorkSucceededEvent,
			ag *timebox.Aggregator[*api.WorkSucceededEvent],
		) error {
			if err := events.Raise(ag, api.EventTypeWorkFailed,
				api.WorkFailedEvent{},
			); err != nil {
				return err
			}
			if err := events.Raise(ag, api.EventTypeWorkSucceeded,
				api.WorkSucceededEvent{Token: "first"},
			); err != nil {
				return err
			}
			return events.Raise(ag, api.EventTypeWorkSucceeded,
				api.WorkSucceededEvent{Token: "second"},
			)
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, api.Token("first"), state.Token)
}

func TestMemoKeyError(t *testing.T) {
	_, err := events.MemoKey(
		&api.Step{}, api.Args{"invalid": make(chan int)},
	)
	assert.Error(t, err)
}
