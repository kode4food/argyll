package events_test

import (
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestRaiseEnqueuesEvent(t *testing.T) {
	ag := &timebox.Aggregator[int]{}

	err := events.Raise(
		ag, api.EventTypeFlowStarted, api.FlowStartedEvent{FlowID: "flow-1"},
	)
	assert.NoError(t, err)
	assert.Len(t, ag.Enqueued(), 1)
}
