package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestSpaceSubFlowReferences(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goal := helpers.NewSimpleStep("goal")
		goal.Tags = api.Tags{"domain:payments"}
		assert.NoError(t, eng.RegisterStep(goal))

		subFlow := &api.Step{
			ID:         "sub-flow",
			Name:       "Sub-flow",
			Type:       api.StepTypeFlow,
			Attributes: api.AttributeSpecs{},
			Flow: &api.FlowConfig{
				Goals:   []api.StepID{goal.ID},
				SpaceID: "payments",
			},
		}
		err := eng.RegisterStep(subFlow)
		assert.ErrorIs(t, err, engine.ErrSpaceNotFound)

		sp := api.Space{
			ID:   "payments",
			Name: "Payments",
			QBE:  api.SpaceQuery{{"domain:payments"}},
		}
		assert.NoError(t, eng.RegisterSpace(sp))
		assert.NoError(t, eng.RegisterStep(subFlow))

		changedGoal := goal.Copy()
		changedGoal.Tags = api.Tags{"domain:inventory"}
		err = eng.UpdateStep(changedGoal)
		assert.ErrorIs(t, err, engine.ErrSpaceGoalExcluded)

		changedSpace := sp
		changedSpace.QBE = api.SpaceQuery{{"domain:inventory"}}
		err = eng.UpdateSpace(changedSpace)
		assert.ErrorIs(t, err, engine.ErrSpaceGoalExcluded)

		err = eng.UnregisterStep(goal.ID)
		assert.ErrorIs(t, err, engine.ErrSubFlowGoalInUse)
		err = eng.UnregisterSpace(sp.ID)
		assert.ErrorIs(t, err, engine.ErrSpaceInUse)

		updated := subFlow.Copy()
		flow := *updated.Flow
		flow.SpaceID = ""
		updated.Flow = &flow
		assert.NoError(t, eng.UpdateStep(updated))
		assert.NoError(t, eng.UnregisterStep(goal.ID))
		assert.NoError(t, eng.UnregisterSpace(sp.ID))
	})
}
