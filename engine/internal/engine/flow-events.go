package engine

import (
	"log/slog"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

func (a *flowActor) createEventHandler() timebox.Handler {
	const (
		flowCompleted = timebox.EventType(api.EventTypeFlowCompleted)
		flowFailed    = timebox.EventType(api.EventTypeFlowFailed)
	)

	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		flowCompleted: timebox.MakeHandler(a.processFlowCompleted),
		flowFailed:    timebox.MakeHandler(a.processFlowFailed),
	})
}

func (a *flowActor) processFlowCompleted(
	_ *timebox.Event, _ api.FlowCompletedEvent,
) error {
	return a.execTransaction(func(ag *FlowAggregator) error {
		a.maybeDeactivate(ag)
		return nil
	})
}

func (a *flowActor) processFlowFailed(
	_ *timebox.Event, _ api.FlowFailedEvent,
) error {
	return a.execTransaction(func(ag *FlowAggregator) error {
		a.maybeDeactivate(ag)
		return nil
	})
}

func (a *flowActor) handleEvent(ev *timebox.Event, handler timebox.Handler) {
	if err := handler(ev); err != nil {
		slog.Error("Failed to handle event",
			log.FlowID(a.flowID),
			slog.String("event_type", string(ev.Type)),
			log.Error(err))
	}
}
