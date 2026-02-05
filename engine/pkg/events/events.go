package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func MakeAppliers[T any](
	app map[api.EventType]timebox.Applier[T],
) timebox.Appliers[T] {
	res := map[timebox.EventType]timebox.Applier[T]{}
	for et, fn := range app {
		res[timebox.EventType(et)] = fn
	}
	return res
}

func MakeDispatcher(
	handlers map[api.EventType]timebox.Handler,
) timebox.Handler {
	res := map[timebox.EventType]timebox.Handler{}
	for et, fn := range handlers {
		res[timebox.EventType(et)] = fn
	}
	return timebox.MakeDispatcher(res)
}
