package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// Raise raises an event through the aggregator
func Raise[T, V any](
	ag *timebox.Aggregator[T], typ api.EventType, value V,
) error {
	return timebox.Raise(ag, timebox.EventType(typ), value)
}
