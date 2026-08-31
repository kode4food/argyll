package engine

import (
	"errors"
	"fmt"
	"slices"

	"github.com/kode4food/argyll/engine/internal/engine/script"
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
	space, err := e.prepareSpace(space)
	if err != nil {
		return err
	}
	return e.CatalogTx(func(tx *CatalogTx) error {
		st := tx.ag.Value()
		old, ok := st.Spaces[space.ID]
		if !ok {
			ids, err := e.selectSpaceSteps(st.Steps, space)
			if err != nil {
				return err
			}
			return events.Raise(tx.ag, api.EventTypeSpaceRegistered,
				api.SpaceRegisteredEvent{
					Space:   space,
					StepIDs: ids,
				},
			)
		}
		if old.Equal(space) {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrSpaceExists, space.ID)
	})
}

// PreviewSpace returns the registered Step IDs selected by an unsaved Space.
// Only its selector matters, so an incomplete Space can be previewed
func (e *Engine) PreviewSpace(space api.Space) ([]api.StepID, error) {
	space, err := e.prepareSelector(space)
	if err != nil {
		return nil, err
	}
	cat, err := e.GetCatalogState()
	if err != nil {
		return nil, err
	}
	return e.selectSpaceSteps(cat.Steps, space)
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
	space, err := e.prepareSpace(space)
	if err != nil {
		return err
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
		if err := e.validateSpaceGoals(st, space); err != nil {
			return err
		}
		ids, err := e.selectSpaceSteps(st.Steps, space)
		if err != nil {
			return err
		}
		return events.Raise(tx.ag, api.EventTypeSpaceUpdated,
			api.SpaceUpdatedEvent{
				Space:   space,
				StepIDs: ids,
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

func (e *Engine) matchingSpaceIDs(
	cat api.CatalogState, step *api.Step,
) ([]api.SpaceID, error) {
	var res []api.SpaceID
	for id, space := range cat.Spaces {
		matches, err := e.spaceMatches(space, step)
		if err != nil {
			return nil, errors.Join(ErrInvalidSpace, err)
		}
		if matches {
			res = append(res, id)
		}
	}
	slices.Sort(res)
	return res, nil
}

func (e *Engine) validateSpaceSubFlows(
	cat api.CatalogState, newStep *api.Step,
) error {
	steps := stepsIncluding(cat, newStep)
	for _, st := range steps {
		if st.Flow == nil || st.Flow.SpaceID == "" {
			continue
		}
		space, ok := cat.Spaces[st.Flow.SpaceID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrSpaceNotFound, st.Flow.SpaceID)
		}
		if err := e.validateSubFlowGoals(steps, st, space); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) validateSpaceGoals(
	cat api.CatalogState, space api.Space,
) error {
	for _, st := range cat.Steps {
		if st.Flow == nil || st.Flow.SpaceID != space.ID {
			continue
		}
		if err := e.validateSubFlowGoals(cat.Steps, st, space); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) validateSubFlowGoals(
	steps api.Steps, st *api.Step, space api.Space,
) error {
	for _, goalID := range st.Flow.Goals {
		goal, ok := steps[goalID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrStepNotFound, goalID)
		}
		matches, err := e.spaceMatches(space, goal)
		if err != nil {
			return err
		}
		if !matches {
			return fmt.Errorf("%w: %s", ErrSpaceGoalExcluded, goalID)
		}
	}
	return nil
}

// prepareSpace prepares the selector and validates the Space identity
func (e *Engine) prepareSpace(space api.Space) (api.Space, error) {
	space, err := e.prepareSelector(space)
	if err != nil {
		return api.Space{}, err
	}
	if err := space.Validate(); err != nil {
		return api.Space{}, errors.Join(ErrInvalidSpace, err)
	}
	return space, nil
}

// prepareSelector normalizes the Space, generating its selector from the QBE
// when one is provided, and verifies that the selector compiles
func (e *Engine) prepareSelector(space api.Space) (api.Space, error) {
	space = space.Normalize()
	if len(space.QBE) > 0 {
		space.Selector = script.QBESelector(space.QBE)
	}
	if err := space.ValidateSelector(); err != nil {
		return api.Space{}, errors.Join(ErrInvalidSpace, err)
	}
	_, err := e.scripts.Compile(script.MatchStep, space.Selector)
	if err != nil {
		return api.Space{}, errors.Join(ErrInvalidSpace, err)
	}
	return space, nil
}

func (e *Engine) selectSpaceSteps(
	steps api.Steps, space api.Space,
) ([]api.StepID, error) {
	res := []api.StepID{}
	for id, step := range steps {
		matches, err := e.spaceMatches(space, step)
		if err != nil {
			return nil, err
		}
		if matches {
			res = append(res, id)
		}
	}
	slices.Sort(res)
	return res, nil
}

// spaceMatches evaluates the Space selector against the Step's labels, which
// are the only value the script is given
func (e *Engine) spaceMatches(space api.Space, step *api.Step) (bool, error) {
	labels := make(map[string]any, len(step.Labels))
	for key, value := range step.Labels {
		labels[key] = value
	}
	return e.Matcher(space.Selector, labels)
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
