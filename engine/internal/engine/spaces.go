package engine

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

var (
	ErrInvalidSpace      = errors.New("invalid space")
	ErrSpaceExists       = errors.New("space exists")
	ErrSpaceNotFound     = errors.New("space not found")
	ErrSpaceInUse        = errors.New("space in use")
	ErrSpaceGoalExcluded = errors.New("goal not in space")
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
		if err := validateSpaceGoals(st, space); err != nil {
			return err
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
		st := tx.ag.Value()
		if _, ok := st.Spaces[spaceID]; !ok {
			return fmt.Errorf("%w: %s", ErrSpaceNotFound, spaceID)
		}
		if ref, ok := spaceSubFlow(st, spaceID); ok {
			return fmt.Errorf("%w: %s", ErrSpaceInUse, ref)
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

func validateSpaceSubFlows(cat api.CatalogState, newStep *api.Step) error {
	steps := stepsIncluding(cat, newStep)
	for _, st := range steps {
		if st.Flow == nil || st.Flow.SpaceID == "" {
			continue
		}
		space, ok := cat.Spaces[st.Flow.SpaceID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrSpaceNotFound, st.Flow.SpaceID)
		}
		if err := validateSubFlowGoals(steps, st, space); err != nil {
			return err
		}
	}
	return nil
}

func validateSpaceGoals(cat api.CatalogState, space api.Space) error {
	for _, st := range cat.Steps {
		if st.Flow == nil || st.Flow.SpaceID != space.ID {
			continue
		}
		if err := validateSubFlowGoals(cat.Steps, st, space); err != nil {
			return err
		}
	}
	return nil
}

func validateSubFlowGoals(
	steps api.Steps, st *api.Step, space api.Space,
) error {
	for _, goalID := range st.Flow.Goals {
		goal, ok := steps[goalID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrStepNotFound, goalID)
		}
		if !space.Matches(goal) {
			return fmt.Errorf("%w: %s", ErrSpaceGoalExcluded, goalID)
		}
	}
	return nil
}

func spaceSubFlow(
	cat api.CatalogState, spaceID api.SpaceID,
) (api.StepID, bool) {
	for id, st := range cat.Steps {
		if st.Flow != nil && st.Flow.SpaceID == spaceID {
			return id, true
		}
	}
	return "", false
}

func spaceSubFlowGoal(
	cat api.CatalogState, goalID api.StepID,
) (api.StepID, bool) {
	for id, st := range cat.Steps {
		if st.Flow != nil && st.Flow.SpaceID != "" &&
			slices.Contains(st.Flow.Goals, goalID) {
			return id, true
		}
	}
	return "", false
}
