package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

// EventFilter is a predicate function for filtering events
type EventFilter func(*timebox.Event) bool

func MakeAppliers[T any](
	app map[api.EventType]timebox.Applier[T],
) timebox.Appliers[T] {
	res := map[timebox.EventType]timebox.Applier[T]{}
	for et, fn := range app {
		res[timebox.EventType(et)] = fn
	}
	return res
}

// FilterEvents creates a filter that matches events with any of the specified
// event types
func FilterEvents(eventTypes ...timebox.EventType) EventFilter {
	lookup := util.Set[timebox.EventType]{}
	for _, et := range eventTypes {
		lookup.Add(et)
	}
	return func(ev *timebox.Event) bool {
		return lookup.Contains(ev.Type)
	}
}

// FilterFlow creates a filter that matches events for a specific flow ID
func FilterFlow(flowID api.FlowID) EventFilter {
	return func(ev *timebox.Event) bool {
		if !IsFlowEvent(ev) {
			return false
		}
		return ev.AggregateID[1] == timebox.ID(flowID)
	}
}

// OrFilters combines multiple filters using OR logic, matching if any filter
// matches the event
func OrFilters(filters ...EventFilter) EventFilter {
	return func(ev *timebox.Event) bool {
		for _, filter := range filters {
			if filter(ev) {
				return true
			}
		}
		return false
	}
}
