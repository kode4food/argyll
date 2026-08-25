package engine

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

var (
	ErrInvalidSpace  = errors.New("invalid space")
	ErrSpaceExists   = errors.New("space exists")
	ErrSpaceNotFound = errors.New("space not found")
)

// RegisterSpace persists a new planning space
func (e *Engine) RegisterSpace(space api.Space) error {
	if err := space.Validate(); err != nil {
		return errors.Join(ErrInvalidSpace, err)
	}
	return e.CatalogTx(func(tx *CatalogTx) error {
		st := tx.ag.Value()
		old, ok := st.Spaces[space.ID]
		if !ok {
			return events.Raise(tx.ag, api.EventTypeSpaceRegistered,
				api.SpaceRegisteredEvent{
					Space: space,
					Steps: st.Query(space.Matches),
				},
			)
		}
		if old.Equal(space) {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrSpaceExists, space.ID)
	})
}

// ListSpaces returns all planning spaces
func (e *Engine) ListSpaces() ([]api.Space, error) {
	cat, err := e.GetCatalogState()
	if err != nil {
		return nil, err
	}
	res := make([]api.Space, 0, len(cat.Spaces))
	for _, space := range cat.Spaces {
		res = append(res, space)
	}
	return res, nil
}

// UpdateSpace persists changes to an existing planning space
func (e *Engine) UpdateSpace(space api.Space) error {
	if err := space.Validate(); err != nil {
		return errors.Join(ErrInvalidSpace, err)
	}
	return e.CatalogTx(func(tx *CatalogTx) error {
		st := tx.ag.Value()
		old, ok := st.Spaces[space.ID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrSpaceNotFound, space.ID)
		}
		if old.Equal(space) {
			return nil
		}
		return events.Raise(tx.ag, api.EventTypeSpaceUpdated,
			api.SpaceUpdatedEvent{
				Space: space,
				Steps: st.Query(space.Matches),
			},
		)
	})
}

// UnregisterSpace removes a planning space
func (e *Engine) UnregisterSpace(spaceID api.SpaceID) error {
	return e.CatalogTx(func(tx *CatalogTx) error {
		if _, ok := tx.ag.Value().Spaces[spaceID]; !ok {
			return fmt.Errorf("%w: %s", ErrSpaceNotFound, spaceID)
		}
		return events.Raise(tx.ag, api.EventTypeSpaceUnregistered,
			api.SpaceUnregisteredEvent{SpaceID: spaceID},
		)
	})
}

func matchingSpaceIDs(cat api.CatalogState, step *api.Step) []api.SpaceID {
	var res []api.SpaceID
	for id, space := range cat.Spaces {
		if space.Matches(step) {
			res = append(res, id)
		}
	}
	slices.Sort(res)
	return res
}
