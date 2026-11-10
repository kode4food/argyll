package events_test

import (
	"encoding/json"
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func testFeedDef() *api.Feed {
	return &api.Feed{
		ID:         "order-total",
		Name:       "Order Total",
		Value:      "0",
		Fold:       &api.ScriptConfig{Language: "lua", Source: "return feed"},
		Attributes: api.Slice[api.Name]{"amount"},
	}
}

func feedEvent(t *testing.T, id timebox.AggregateID, typ api.EventType, data any) *timebox.Event {
	t.Helper()
	b, err := json.Marshal(data)
	assert.NoError(t, err)
	return &timebox.Event{
		Timestamp:   scheduler.Now(),
		AggregateID: id,
		Type:        timebox.EventType(typ),
		Data:        b,
	}
}

func TestCatalogFeedEvents(t *testing.T) {
	f := testFeedDef()
	cat := events.NewCatalogState()
	assert.NotNil(t, cat.Feeds)

	ev := feedEvent(t, events.CatalogKey, api.EventTypeFeedRegistered,
		api.FeedRegisteredEvent{Feed: f})
	cat = events.CatalogAppliers[ev.Type](cat, ev)
	assert.Equal(t, f, cat.Feeds[f.ID])
	assert.True(t, cat.LastUpdated.Equal(ev.Timestamp))

	updated := testFeedDef()
	updated.Name = "Renamed"
	ev = feedEvent(t, events.CatalogKey, api.EventTypeFeedUpdated,
		api.FeedUpdatedEvent{Feed: updated})
	cat = events.CatalogAppliers[ev.Type](cat, ev)
	assert.Equal(t, api.Name("Renamed"), cat.Feeds[f.ID].Name)

	ev = feedEvent(t, events.CatalogKey, api.EventTypeFeedUnregistered,
		api.FeedUnregisteredEvent{FeedID: f.ID})
	cat = events.CatalogAppliers[ev.Type](cat, ev)
	assert.Empty(t, cat.Feeds)
}

func TestFeedAggregate(t *testing.T) {
	key := events.FeedKey("order-total", "v1")
	assert.Equal(t, events.FeedPrefix, string(key.Type))
	assert.NotEqual(t, key, events.FeedKey("order-total", "v2"))

	state := events.NewFeedState()
	assert.Empty(t, state.Version)

	failure := api.FeedFoldFailedEvent{FeedID: "order-total", Version: "v1"}
	failed := feedEvent(t, key, api.EventTypeFeedFoldFailed, failure)
	state = events.FeedAppliers.Reduce(state, []*timebox.Event{failed})
	assert.Empty(t, state.Version)
	assert.Nil(t, state.Value)
	assert.Equal(t, api.FeedID("order-total"), state.ID)

	fold := api.FeedFoldedEvent{FeedID: "order-total", Version: "v1", Value: 7}
	folded := feedEvent(t, key, api.EventTypeFeedFolded, fold)
	state = events.FeedAppliers.Reduce(state, []*timebox.Event{folded})
	assert.Equal(t, "v1", state.Version)
	assert.EqualValues(t, 7, state.Value)
}
