package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

const CatalogPrefix = "catalog"

var (
	CatalogKey = timebox.NewAggregateID(CatalogPrefix)

	CatalogAppliers = makeCatalogAppliers()
)

// NewCatalogState creates an empty catalog state with initialized maps
func NewCatalogState() *api.CatalogState {
	return &api.CatalogState{
		Steps:      api.Steps{},
		Attributes: api.AttributeGraph{},
	}
}

// IsCatalogEvent returns true if the event is for the catalog aggregate
func IsCatalogEvent(ev *timebox.Event) bool {
	return len(ev.AggregateID) >= 1 && ev.AggregateID[0] == CatalogPrefix
}

func makeCatalogAppliers() timebox.Appliers[*api.CatalogState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[*api.CatalogState]{
		api.EventTypeStepRegistered:   timebox.MakeApplier(stepRegistered),
		api.EventTypeStepUnregistered: timebox.MakeApplier(stepUnregistered),
		api.EventTypeStepUpdated:      timebox.MakeApplier(stepUpdated),
	})
}

func stepRegistered(
	st *api.CatalogState, ev *timebox.Event, data api.StepRegisteredEvent,
) *api.CatalogState {
	return st.
		SetStep(data.Step.ID, data.Step).
		SetLastUpdated(ev.Timestamp)
}

func stepUnregistered(
	st *api.CatalogState, ev *timebox.Event, data api.StepUnregisteredEvent,
) *api.CatalogState {
	return st.
		DeleteStep(data.StepID).
		SetLastUpdated(ev.Timestamp)
}

func stepUpdated(
	st *api.CatalogState, ev *timebox.Event, data api.StepUpdatedEvent,
) *api.CatalogState {
	return st.
		SetStep(data.Step.ID, data.Step).
		SetLastUpdated(ev.Timestamp)
}
