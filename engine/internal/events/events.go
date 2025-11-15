package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/util"
)

type EventFilter func(*timebox.Event) bool

func FilterEvents(eventTypes ...timebox.EventType) EventFilter {
	lookup := util.Set[timebox.EventType]{}
	for _, et := range eventTypes {
		lookup.Add(et)
	}
	return func(ev *timebox.Event) bool {
		return lookup.Contains(ev.Type)
	}
}

func FilterWorkflow(flowID timebox.ID) EventFilter {
	return func(ev *timebox.Event) bool {
		if !IsWorkflowEvent(ev) {
			return false
		}
		return ev.AggregateID[1] == flowID
	}
}

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
