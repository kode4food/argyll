package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestUpdateStepCycles(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := helpers.NewSimpleStep("step-a")
		stepA.Attributes = api.AttributeSpecs{
			"foo": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(stepA))

		stepB := helpers.NewSimpleStep("step-b")
		stepB.Attributes = api.AttributeSpecs{
			"foo": {Role: api.RoleRequired, Type: api.TypeString},
			"bar": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(stepB))

		stepC := helpers.NewSimpleStep("step-c")
		stepC.Attributes = api.AttributeSpecs{
			"bar": {Role: api.RoleRequired, Type: api.TypeString},
			"baz": {Role: api.RoleOutput, Type: api.TypeString},
		}
		assert.NoError(t, eng.RegisterStep(stepC))

		updatedA := helpers.NewSimpleStep("step-a")
		updatedA.Attributes = api.AttributeSpecs{
			"baz": {Role: api.RoleRequired, Type: api.TypeString},
			"qux": {Role: api.RoleOutput, Type: api.TypeString},
		}

		assert.NoError(t, eng.UpdateStep(updatedA))
	})
}

func TestRegisterStepRejectsFlowGoalCycles(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "flow-a",
			Name: "Flow A",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-b"},
			},
			Attributes: api.AttributeSpecs{},
		}
		stepB := &api.Step{
			ID:   "flow-b",
			Name: "Flow B",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-a"},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, eng.RegisterStep(stepA))

		err := eng.RegisterStep(stepB)
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorIs(t, err, engine.ErrCircularDependency)
	})
}

func TestCatalogTxRejectsGoalCycle(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "flow-a",
			Name: "Flow A",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-b"},
			},
			Attributes: api.AttributeSpecs{},
		}
		stepB := &api.Step{
			ID:   "flow-b",
			Name: "Flow B",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-a"},
			},
			Attributes: api.AttributeSpecs{},
		}

		err := eng.CatalogTx(func(tx *engine.CatalogTx) error {
			if err := tx.Register(stepA); err != nil {
				return err
			}
			return tx.Register(stepB)
		})
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorIs(t, err, engine.ErrCircularDependency)

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)
		assert.Empty(t, cat.Steps)
	})
}

func TestUpdateStepRejectsGoalCycles(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "flow-a",
			Name: "Flow A",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-b"},
			},
			Attributes: api.AttributeSpecs{},
		}
		stepB := &api.Step{
			ID:   "flow-b",
			Name: "Flow B",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"leaf"},
			},
			Attributes: api.AttributeSpecs{},
		}
		leaf := helpers.NewSimpleStep("leaf")

		assert.NoError(t, eng.RegisterStep(leaf))
		assert.NoError(t, eng.RegisterStep(stepA))
		assert.NoError(t, eng.RegisterStep(stepB))

		updatedLeaf := &api.Step{
			ID:   "leaf",
			Name: "Leaf",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{"flow-a"},
			},
			Attributes: api.AttributeSpecs{},
		}

		err := eng.UpdateStep(updatedLeaf)
		assert.ErrorIs(t, err, engine.ErrInvalidStep)
		assert.ErrorIs(t, err, engine.ErrCircularDependency)
	})
}
