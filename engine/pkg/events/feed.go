package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// FeedPrefix is the aggregate type of every Feed version
const FeedPrefix = "feed"

// FeedAppliers contains the event applier functions for Feed events
var FeedAppliers = makeFeedAppliers()

// FeedKey returns the aggregate ID for one Feed definition version
func FeedKey(id api.FeedID, version string) timebox.AggregateID {
	return timebox.NewAggregateID(FeedPrefix, timebox.ID(string(id)+":"+version))
}

// IsFeedEventID returns whether an aggregate ID names one Feed version
func IsFeedEventID(id timebox.AggregateID) bool {
	return id.Type == FeedPrefix && id.Key != ""
}

// NewFeedState creates a Feed state with no contributions
func NewFeedState() api.FeedState {
	return api.FeedState{}
}

func makeFeedAppliers() timebox.Appliers[api.FeedState] {
	return MakeAppliers(map[api.EventType]timebox.Applier[api.FeedState]{
		api.EventTypeFeedFolded:     timebox.MakeApplier(feedFolded),
		api.EventTypeFeedFoldFailed: timebox.MakeApplier(feedFoldFailed),
	})
}

func feedFolded(state api.FeedState, ev *timebox.Event, data api.FeedFoldedEvent) api.FeedState {
	return state.
		SetID(data.FeedID).
		SetVersion(data.Version).
		SetValue(data.Value).
		SetLastUpdated(ev.Timestamp)
}

func feedFoldFailed(
	state api.FeedState, ev *timebox.Event, data api.FeedFoldFailedEvent,
) api.FeedState {
	// Version marks a state that holds a value, which a failed fold never sets
	return state.
		SetID(data.FeedID).
		SetLastUpdated(ev.Timestamp)
}
