package engine

import (
	"fmt"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
)

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
