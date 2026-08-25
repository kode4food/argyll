package events

import (
	"slices"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const CatalogPrefix = "catalog"

var (
	CatalogKey = timebox.NewAggregateID(CatalogPrefix)

	CatalogAppliers = makeCatalogAppliers()
)

// NewCatalogState creates an empty catalog state with initialized maps
func NewCatalogState() api.CatalogState {
	return api.CatalogState{
		Steps:      api.Steps{},
		Spaces:     api.Spaces{},
		Selection:  api.SpaceSelection{},
		Attributes: api.AttributeGraph{},
	}
}

// IsCatalogEvent returns true if the event is for the catalog aggregate
func IsCatalogEvent(ev *timebox.Event) bool {
	return IsCatalogEventID(ev.AggregateID)
}

// IsCatalogEventID returns true if the ID is for the catalog aggregate
func IsCatalogEventID(id timebox.AggregateID) bool {
	return len(id) == 1 && id[0] == CatalogPrefix
}

func makeCatalogAppliers() timebox.Appliers[api.CatalogState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[api.CatalogState]{
		api.EventTypeStepRegistered:    timebox.MakeApplier(stepRegistered),
		api.EventTypeStepUnregistered:  timebox.MakeApplier(stepUnregistered),
		api.EventTypeStepUpdated:       timebox.MakeApplier(stepUpdated),
		api.EventTypeSpaceRegistered:   timebox.MakeApplier(spaceRegistered),
		api.EventTypeSpaceUpdated:      timebox.MakeApplier(spaceUpdated),
		api.EventTypeSpaceUnregistered: timebox.MakeApplier(spaceUnregistered),
	})
}

func stepRegistered(
	st api.CatalogState, ev *timebox.Event, data api.StepRegisteredEvent,
) api.CatalogState {
	return st.
		SetStep(data.Step.ID, data.Step).
		SetStepSpaces(data.Step.ID, data.Spaces).
		SetLastUpdated(ev.Timestamp)
}

func stepUnregistered(
	st api.CatalogState, ev *timebox.Event, data api.StepUnregisteredEvent,
) api.CatalogState {
	return st.
		DeleteStep(data.StepID).
		SetStepSpaces(data.StepID, nil).
		SetLastUpdated(ev.Timestamp)
}

func stepUpdated(
	st api.CatalogState, ev *timebox.Event, data api.StepUpdatedEvent,
) api.CatalogState {
	return st.
		SetStep(data.Step.ID, data.Step).
		SetStepSpaces(data.Step.ID, data.Spaces).
		SetLastUpdated(ev.Timestamp)
}

func spaceRegistered(
	st api.CatalogState, ev *timebox.Event, data api.SpaceRegisteredEvent,
) api.CatalogState {
	return st.
		SetSpace(data.Space.ID, data.Space).
		SetSpaceSelection(data.Space.ID, stepIDs(data.Steps)).
		SetLastUpdated(ev.Timestamp)
}

func spaceUpdated(
	st api.CatalogState, ev *timebox.Event, data api.SpaceUpdatedEvent,
) api.CatalogState {
	return st.
		SetSpace(data.Space.ID, data.Space).
		SetSpaceSelection(data.Space.ID, stepIDs(data.Steps)).
		SetLastUpdated(ev.Timestamp)
}

func spaceUnregistered(
	st api.CatalogState, ev *timebox.Event, data api.SpaceUnregisteredEvent,
) api.CatalogState {
	return st.
		DeleteSpace(data.SpaceID).
		DeleteSpaceSelection(data.SpaceID).
		SetLastUpdated(ev.Timestamp)
}

func stepIDs(steps api.Steps) []api.StepID {
	res := make([]api.StepID, 0, len(steps))
	for id := range steps {
		res = append(res, id)
	}
	slices.Sort(res)
	return res
}
