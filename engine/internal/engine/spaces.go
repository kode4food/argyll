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
func (e *Engine) RegisterSpace(sp api.Space) error {
	sp, err := e.prepareSpace(sp)
	if err != nil {
		return err
	}
	return e.CatalogTx(func(tx *CatalogTx) error {
		cat := tx.ag.Value()
		old, ok := cat.Spaces[sp.ID]
		if !ok {
			ids, err := e.selectSpaceSteps(cat.Steps, sp)
			if err != nil {
				return err
			}
			return events.Raise(tx.ag, api.EventTypeSpaceRegistered,
				api.SpaceRegisteredEvent{
					Space:   sp,
					StepIDs: ids,
				},
			)
		}
		if old.Equal(sp) {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrSpaceExists, sp.ID)
	})
}

// PreviewSpace prepares an unsaved Space and returns it with selected Step IDs
func (e *Engine) PreviewSpace(sp api.Space) (api.SpacePreviewResponse, error) {
	sp, err := e.prepareSelector(sp)
	if err != nil {
		return api.SpacePreviewResponse{}, err
	}
	cat, err := e.GetCatalogState()
	if err != nil {
		return api.SpacePreviewResponse{}, err
	}
	ids, err := e.selectSpaceSteps(cat.Steps, sp)
	if err != nil {
		return api.SpacePreviewResponse{}, err
	}
	return api.SpacePreviewResponse{Space: sp, StepIDs: ids}, nil
}

// ListSpaces returns all planning spaces
func (e *Engine) ListSpaces() ([]api.Space, error) {
	cat, err := e.GetCatalogState()
	if err != nil {
		return nil, err
	}
	res := make([]api.Space, 0, len(cat.Spaces))
	for _, sp := range cat.Spaces {
		res = append(res, sp)
	}
	return res, nil
}

// UpdateSpace persists changes to an existing planning space
func (e *Engine) UpdateSpace(sp api.Space) error {
	sp, err := e.prepareSpace(sp)
	if err != nil {
		return err
	}
	return e.CatalogTx(func(tx *CatalogTx) error {
		cat := tx.ag.Value()
		old, ok := cat.Spaces[sp.ID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrSpaceNotFound, sp.ID)
		}
		if old.Equal(sp) {
			return nil
		}
		if err := e.validateSpaceGoals(cat, sp); err != nil {
			return err
		}
		ids, err := e.selectSpaceSteps(cat.Steps, sp)
		if err != nil {
			return err
		}
		return events.Raise(tx.ag, api.EventTypeSpaceUpdated,
			api.SpaceUpdatedEvent{
				Space:   sp,
				StepIDs: ids,
			},
		)
	})
}

// UnregisterSpace removes a planning space
func (e *Engine) UnregisterSpace(spaceID api.SpaceID) error {
	return e.CatalogTx(func(tx *CatalogTx) error {
		cat := tx.ag.Value()
		if _, ok := cat.Spaces[spaceID]; !ok {
			return fmt.Errorf("%w: %s", ErrSpaceNotFound, spaceID)
		}
		if ref, ok := spaceSubFlow(cat, spaceID); ok {
			return fmt.Errorf("%w: %s", ErrSpaceInUse, ref)
		}
		return events.Raise(tx.ag, api.EventTypeSpaceUnregistered,
			api.SpaceUnregisteredEvent{SpaceID: spaceID},
		)
	})
}

func (e *Engine) matchingSpaceIDs(
	cat api.CatalogState, st *api.Step,
) ([]api.SpaceID, error) {
	var res []api.SpaceID
	for id, sp := range cat.Spaces {
		matches, err := e.spaceMatches(sp, st)
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
		sp, ok := cat.Spaces[st.Flow.SpaceID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrSpaceNotFound, st.Flow.SpaceID)
		}
		if err := e.validateSubFlowGoals(steps, st, sp); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) validateSpaceGoals(cat api.CatalogState, sp api.Space) error {
	for _, st := range cat.Steps {
		if st.Flow == nil || st.Flow.SpaceID != sp.ID {
			continue
		}
		if err := e.validateSubFlowGoals(cat.Steps, st, sp); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) validateSubFlowGoals(
	steps api.Steps, st *api.Step, sp api.Space,
) error {
	for _, goalID := range st.Flow.Goals {
		goal, ok := steps[goalID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrStepNotFound, goalID)
		}
		matches, err := e.spaceMatches(sp, goal)
		if err != nil {
			return err
		}
		if !matches {
			return fmt.Errorf("%w: %s", ErrSpaceGoalExcluded, goalID)
		}
	}
	return nil
}

func (e *Engine) prepareSpace(sp api.Space) (api.Space, error) {
	sp, err := e.prepareSelector(sp)
	if err != nil {
		return api.Space{}, err
	}
	if err := sp.Validate(); err != nil {
		return api.Space{}, errors.Join(ErrInvalidSpace, err)
	}
	return sp, nil
}

func (e *Engine) prepareSelector(sp api.Space) (api.Space, error) {
	sp = sp.Normalize()
	if len(sp.QBE) > 0 {
		sp.Selector = script.QBESelector(sp.QBE)
	}
	if err := sp.ValidateSelector(); err != nil {
		return api.Space{}, errors.Join(ErrInvalidSpace, err)
	}
	_, err := e.scripts.Compile(script.MatchStep, sp.Selector)
	if err != nil {
		return api.Space{}, errors.Join(ErrInvalidSpace, err)
	}
	return sp, nil
}

func (e *Engine) selectSpaceSteps(
	steps api.Steps, sp api.Space,
) ([]api.StepID, error) {
	res := []api.StepID{}
	for id, st := range steps {
		matches, err := e.spaceMatches(sp, st)
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

func (e *Engine) spaceMatches(sp api.Space, st *api.Step) (bool, error) {
	labels := make(map[string]any, len(st.Labels))
	for key, value := range st.Labels {
		labels[key] = value
	}
	return e.Matcher(sp.Selector, labels)
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
