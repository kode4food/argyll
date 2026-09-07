package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestCreatePlanEmbedsChildPlans(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		child := &api.Step{
			ID:   "child",
			Name: "Child",
			Type: api.StepTypeService,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput, Type: api.TypeString},
			},
			HTTP: &api.HTTPConfig{
				Invoke: api.HTTPAction{
					Endpoint: "http://test",
					Timeout:  30 * api.Second,
				},
			},
		}
		assert.NoError(t, eng.RegisterStep(child))

		parent := &api.Step{
			ID:   "parent",
			Name: "Parent",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{child.ID},
			},
			Attributes: api.AttributeSpecs{
				"wrapped": {
					Role: api.RoleOutput,
					Type: api.TypeString,
					Output: &api.OutputConfig{
						Mapping: &api.MappingConfig{Name: "result"},
					},
				},
			},
		}
		assert.NoError(t, eng.RegisterStep(parent))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)

		pl, err := plan.Create(&plan.Request{
			Match:    eng.Matcher,
			Children: eng.Children,
			Steps:    cat.Steps,
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		assert.NoError(t, err)

		if assert.Contains(t, pl.Children, parent.ID) {
			childPlan := pl.Children[parent.ID]
			assert.NotNil(t, childPlan)
			assert.Contains(t, childPlan.Steps, child.ID)
		}
	})
}

func TestCreatePlanRestrictsChildFlowToSpace(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		provider := helpers.NewSimpleStep("provider")
		provider.Attributes = api.AttributeSpecs{
			"input": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(provider))

		goal := helpers.NewSimpleStep("goal")
		goal.Tags = api.Tags{"domain:payments"}
		goal.Attributes = api.AttributeSpecs{
			"input": {Role: api.RoleRequired, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(goal))

		sp := api.Space{
			ID:   "payments",
			Name: "Payments",
			QBE:  api.SpaceQuery{{"domain:payments"}},
		}
		assert.NoError(t, eng.RegisterSpace(sp))

		parent := &api.Step{
			ID:         "parent",
			Name:       "Parent",
			Type:       api.StepTypeFlow,
			Attributes: api.AttributeSpecs{},
			Flow: &api.FlowConfig{
				Goals:   []api.StepID{goal.ID},
				SpaceID: sp.ID,
			},
		}
		assert.NoError(t, eng.RegisterStep(parent))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		pl, err := plan.Create(&plan.Request{
			Match:    eng.Matcher,
			Children: eng.Children,
			Catalog:  cat,
			Steps:    cat.Steps,
			Goals:    []api.StepID{parent.ID},
			Init:     api.InitArgs{},
		})
		assert.NoError(t, err)

		child := pl.Children[parent.ID]
		assert.NotContains(t, child.Steps, provider.ID)
		assert.Contains(t, child.Required, api.Name("input"))
	})
}
