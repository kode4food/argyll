package engine

import (
	"errors"
	"fmt"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

var (
	ErrFeedInUse = errors.New("feed in use")
)

// RegisterFeed persists a new Feed. Registering an equal definition again is a no-op
func (e *Engine) RegisterFeed(f *api.Feed) error {
	f, err := e.prepareFeed(f)
	if err != nil {
		return err
	}
	return e.catalogTx(func(tx *catalogTx) error {
		cat := tx.ag.Value()
		if err := checkFeedCatalog(&cat, f); err != nil {
			return err
		}
		old, ok := cat.Feeds[f.ID]
		if !ok {
			return events.Raise(tx.ag, api.EventTypeFeedRegistered,
				api.FeedRegisteredEvent{Feed: f},
			)
		}
		if old.Equal(f) {
			return nil
		}
		return fmt.Errorf("%w: %s", api.ErrFeedExists, f.ID)
	})
}

// ListFeeds returns every registered Feed
func (e *Engine) ListFeeds() ([]*api.Feed, error) {
	cat, err := e.GetCatalogState()
	if err != nil {
		return nil, err
	}
	res := make([]*api.Feed, 0, len(cat.Feeds))
	for _, f := range cat.Feeds {
		res = append(res, f)
	}
	return res, nil
}

// UpdateFeed replaces an existing Feed's definition
func (e *Engine) UpdateFeed(f *api.Feed) error {
	f, err := e.prepareFeed(f)
	if err != nil {
		return err
	}
	return e.catalogTx(func(tx *catalogTx) error {
		cat := tx.ag.Value()
		old, ok := cat.Feeds[f.ID]
		if !ok {
			return fmt.Errorf("%w: %s", api.ErrFeedNotFound, f.ID)
		}
		if err := checkFeedCatalog(&cat, f); err != nil {
			return err
		}
		if old.Equal(f) {
			return nil
		}
		return events.Raise(tx.ag, api.EventTypeFeedUpdated,
			api.FeedUpdatedEvent{Feed: f},
		)
	})
}

// UnregisterFeed removes a Feed that no registered Step reads
func (e *Engine) UnregisterFeed(id api.FeedID) error {
	return e.catalogTx(func(tx *catalogTx) error {
		cat := tx.ag.Value()
		if _, ok := cat.Feeds[id]; !ok {
			return fmt.Errorf("%w: %s", api.ErrFeedNotFound, id)
		}
		if sids := cat.FeedConsumers(id); len(sids) > 0 {
			return fmt.Errorf("%w: %s", ErrFeedInUse, sids[0])
		}
		return events.Raise(tx.ag, api.EventTypeFeedUnregistered,
			api.FeedUnregisteredEvent{FeedID: id},
		)
	})
}

// GetFeed returns a Feed definition and the value of its current version
func (e *Engine) GetFeed(id api.FeedID) (api.FeedResponse, error) {
	cat, err := e.GetCatalogState()
	if err != nil {
		return api.FeedResponse{}, err
	}
	f, ok := cat.Feeds[id]
	if !ok {
		return api.FeedResponse{}, fmt.Errorf("%w: %s", api.ErrFeedNotFound, id)
	}
	state, err := e.feedState(f)
	if err != nil {
		return api.FeedResponse{}, err
	}
	return api.FeedResponse{
		Feed:  f,
		State: state,
	}, nil
}

// GetFeedStateSeq returns the folded state of one Feed version and the next sequence of its
// aggregate. A version that has never folded has no Version
func (e *Engine) GetFeedStateSeq(key timebox.AggregateID) (api.FeedState, int64, error) {
	var nextSeq int64
	state, err := e.feedExec.Exec(key,
		func(_ api.FeedState, ag *FeedAggregator) error {
			nextSeq = ag.NextSequence()
			return nil
		},
	)
	return state, nextSeq, err
}

func (e *Engine) prepareFeed(f *api.Feed) (*api.Feed, error) {
	if f == nil {
		return nil, fmt.Errorf("%w: %w", api.ErrInvalidFeed, api.ErrFeedIDEmpty)
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if _, err := e.scripts.Compile(feedFoldStep(f), f.Fold); err != nil {
		return nil, fmt.Errorf("%w: fold: %w", api.ErrInvalidFeed, err)
	}
	return f, nil
}

// feedState reads the folded value of a Feed's current version, substituting the seed for a version
// that has never folded
func (e *Engine) feedState(f *api.Feed) (api.FeedState, error) {
	version := f.Version()
	state, err := e.feedExec.Get(events.FeedKey(f.ID, version))
	if err != nil {
		return api.FeedState{}, err
	}
	if state.Version != "" {
		return state, nil
	}
	seed, err := f.Seed()
	if err != nil {
		return api.FeedState{}, err
	}
	state.ID = f.ID
	state.Version = version
	state.Value = seed
	return state, nil
}

// validateStepFeeds requires every Feed a Step reads to be registered
func validateStepFeeds(cat *api.CatalogState, st *api.Step) error {
	for name, attr := range st.Attributes {
		if attr != nil && attr.IsFeed() && cat.Feeds[attr.FeedID()] == nil {
			return fmt.Errorf("%w: %s for attribute %q", api.ErrFeedNotFound, attr.FeedID(), name)
		}
	}
	return nil
}

// checkFeedCatalog requires the Feed's Space to exist and each of its Attributes to be the output
// of a registered Step
func checkFeedCatalog(cat *api.CatalogState, f *api.Feed) error {
	if _, ok := cat.Spaces.Defined[f.SpaceID]; !ok && f.SpaceID != api.NoSpace {
		return fmt.Errorf("%w: %w: %s", api.ErrInvalidFeed, api.ErrSpaceNotFound, f.SpaceID)
	}
	for _, name := range f.Attributes {
		edges := cat.Attributes[name]
		if edges == nil || len(edges.Providers) == 0 {
			return fmt.Errorf("%w: %w: %s", api.ErrInvalidFeed, api.ErrFeedAttributeUnknown, name)
		}
	}
	return nil
}
