package events_test

import (
	"testing"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/events"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRaiseEnqueuesEvent(t *testing.T) {
	ag := &timebox.Aggregator[int]{}

	err := events.Raise(
		ag, api.EventTypeFlowStarted, api.FlowStartedEvent{FlowID: "flow-1"},
	)
	if err != nil {
		t.Fatalf("raise returned error: %v", err)
	}

	if len(ag.Enqueued()) != 1 {
		t.Fatalf("expected 1 event enqueued, got %d", len(ag.Enqueued()))
	}
}
