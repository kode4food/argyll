package engine_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const sharedGraphDepth = 24

func TestRegisterSharedGraph(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprintf("batch=%t", batch), func(t *testing.T) {
			helpers.WithEngine(t, func(eng *engine.Engine) {
				steps := sharedGraphSteps()
				if batch {
					assert.NoError(t, eng.RegisterSteps(steps...))
				} else {
					for _, st := range steps {
						assert.NoError(t, eng.RegisterStep(st))
					}
				}
				cat, err := eng.GetCatalogState()
				assert.NoError(t, err)
				assert.Len(t, cat.Steps, len(steps))
				first := helpers.NewStepWithOutputs(
					steps[0].ID, api.Name(steps[0].ID),
				)
				last := steps[len(steps)-1]
				first.Attributes[api.Name(last.ID)] = &api.AttributeSpec{
					Role: api.RoleRequired,
					Type: api.TypeString,
				}
				assert.ErrorIs(t,
					eng.UpdateStep(first), api.ErrCircularDependency)
			})
		})
	}
}

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
				Goals: api.Goals{Steps: []api.StepID{"flow-b"}},
			},
			Attributes: api.AttributeSpecs{},
		}
		stepB := &api.Step{
			ID:   "flow-b",
			Name: "Flow B",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{"flow-a"}},
			},
			Attributes: api.AttributeSpecs{},
		}

		assert.NoError(t, eng.RegisterStep(stepA))

		err := eng.RegisterStep(stepB)
		assert.ErrorIs(t, err, api.ErrInvalidStep)
		assert.ErrorIs(t, err, api.ErrCircularDependency)
	})
}

func TestRegisterStepsRejectsGoalCycle(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		stepA := &api.Step{
			ID:   "flow-a",
			Name: "Flow A",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{"flow-b"}},
			},
			Attributes: api.AttributeSpecs{},
		}
		stepB := &api.Step{
			ID:   "flow-b",
			Name: "Flow B",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{"flow-a"}},
			},
			Attributes: api.AttributeSpecs{},
		}

		err := eng.RegisterSteps(stepA, stepB)
		assert.ErrorIs(t, err, api.ErrInvalidStep)
		assert.ErrorIs(t, err, api.ErrCircularDependency)

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
				Goals: api.Goals{Steps: []api.StepID{"flow-b"}},
			},
			Attributes: api.AttributeSpecs{},
		}
		stepB := &api.Step{
			ID:   "flow-b",
			Name: "Flow B",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: api.Goals{Steps: []api.StepID{"leaf"}},
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
				Goals: api.Goals{Steps: []api.StepID{"flow-a"}},
			},
			Attributes: api.AttributeSpecs{},
		}

		err := eng.UpdateStep(updatedLeaf)
		assert.ErrorIs(t, err, api.ErrInvalidStep)
		assert.ErrorIs(t, err, api.ErrCircularDependency)
	})
}

func sharedGraphSteps() []*api.Step {
	var steps []*api.Step
	for layer := range sharedGraphDepth {
		for i := range 2 {
			sid := api.StepID(fmt.Sprintf("layer%d_%d", layer, i))
			st := helpers.NewStepWithOutputs(sid, api.Name(sid))
			if layer > 0 {
				for j := range 2 {
					name := api.Name(fmt.Sprintf("layer%d_%d", layer-1, j))
					st.Attributes[name] = &api.AttributeSpec{
						Role: api.RoleRequired,
						Type: api.TypeString,
					}
				}
			}
			steps = append(steps, st)
		}
	}
	return steps
}
