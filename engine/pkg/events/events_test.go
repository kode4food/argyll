package events_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestRaiseEnqueuesEvent(t *testing.T) {
	server, err := miniredis.Run()
	require.NoError(t, err)
	defer server.Close()

	tb, err := timebox.NewTimebox()
	require.NoError(t, err)
	defer func() { _ = tb.Close() }()

	store, err := tb.NewStore(timebox.Config{
		Redis: timebox.RedisConfig{Addr: server.Addr()},
	})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	exec := timebox.NewExecutor(store, func() int { return 0 }, nil)

	id := timebox.NewAggregateID("flow", "flow-1")
	called := false
	_, err = exec.Exec(context.Background(), id,
		func(_ int, ag *timebox.Aggregator[int]) error {
			ag.OnSuccess(func(_ int, evs []*timebox.Event) {
				called = true
				if !assert.Len(t, evs, 1) {
					return
				}

				ev := evs[0]
				assert.Equal(t, timebox.EventType(api.EventTypeFlowStarted), ev.Type)
				assert.Equal(t, int64(0), ev.Sequence)
				assert.Equal(t, id, ev.AggregateID)

				var data api.FlowStartedEvent
				err := json.Unmarshal(ev.Data, &data)
				assert.NoError(t, err)
				assert.Equal(t, api.FlowID("flow-1"), data.FlowID)
			})
			return events.Raise(ag, api.EventTypeFlowStarted,
				api.FlowStartedEvent{FlowID: "flow-1"},
			)
		},
	)
	assert.NoError(t, err)
	assert.True(t, called)
}
