package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/util"
)

// EventFilter is a predicate function for filtering events
type EventFilter func(*timebox.Event) bool

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

// FilterWorkflow creates a filter that matches events for a specific workflow
// ID
func FilterWorkflow(flowID timebox.ID) EventFilter {
	return func(ev *timebox.Event) bool {
		if !IsWorkflowEvent(ev) {
			return false
		}
		return ev.AggregateID[1] == flowID
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
