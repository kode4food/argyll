package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// MakeAppliers converts an api.EventType applier map to timebox types
func MakeAppliers[T any](
	app map[api.EventType]timebox.Applier[T],
) timebox.Appliers[T] {
	res := map[timebox.EventType]timebox.Applier[T]{}
	for et, fn := range app {
		res[timebox.EventType(et)] = fn
	}
	return res
}

// Raise raises an event through the aggregator
func Raise[T, V any](
	ag *timebox.Aggregator[T], typ api.EventType, value V,
) error {
	return timebox.Raise(ag, timebox.EventType(typ), value)
}
