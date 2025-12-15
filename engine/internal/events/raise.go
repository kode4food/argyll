package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// Raise raises an event through the aggregator
func Raise[T, E any](
	ag *timebox.Aggregator[T], eventType api.EventType, event E,
) error {
	return ag.Raise(timebox.EventType(eventType), event)
}
