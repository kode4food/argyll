package events

import "github.com/kode4food/timebox"

type EventFilter func(*timebox.Event) bool

func FilterEvents(eventTypes ...timebox.EventType) EventFilter {
	lookup := map[timebox.EventType]bool{}
	for _, et := range eventTypes {
		lookup[et] = true
	}
	return func(ev *timebox.Event) bool {
		return lookup[ev.Type]
	}
}

func FilterWorkflow(workflowID timebox.ID) EventFilter {
	return func(ev *timebox.Event) bool {
		if !IsWorkflowEvent(ev) {
			return false
		}
		return ev.AggregateID[1] == workflowID
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
