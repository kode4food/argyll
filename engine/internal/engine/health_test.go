package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestUpdateStepHealth(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("health-step")

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		err = eng.UpdateStepHealth("health-step", api.HealthHealthy, "")
		assert.NoError(t, err)

		state, err := eng.GetNodeState()
		assert.NoError(t, err)

		health, ok := state.Health["health-step"]
		assert.True(t, ok)
		assert.Equal(t, api.HealthHealthy, health.Status)
	})
}

func TestUpdateUnhealthy(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("unhealthy-step")

		err := eng.RegisterStep(st)
		assert.NoError(t, err)

		err = eng.UpdateStepHealth(
			"unhealthy-step", api.HealthUnhealthy, "connection refused",
		)
		assert.NoError(t, err)

		state, err := eng.GetNodeState()
		assert.NoError(t, err)

		health, ok := state.Health["unhealthy-step"]
		assert.True(t, ok)
		assert.Equal(t, api.HealthUnhealthy, health.Status)
		assert.Equal(t, "connection refused", health.Error)
	})
}

func TestFlowHealth(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goalA := helpers.NewSimpleStep("goal-a")
		goalB := helpers.NewSimpleStep("goal-b")
		fl := &api.Step{
			ID:   "flow-step",
			Name: "Flow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{goalA.ID, goalB.ID},
			},
			Attributes: api.AttributeSpecs{
				"out": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		assert.NoError(t, eng.RegisterStep(goalA))
		assert.NoError(t, eng.RegisterStep(goalB))
		assert.NoError(t, eng.RegisterStep(fl))

		assert.NoError(t,
			eng.UpdateStepHealth(goalA.ID, api.HealthHealthy, ""),
		)
		assert.NoError(t,
			eng.UpdateStepHealth(goalB.ID, api.HealthUnknown, ""),
		)

		healthByStepID := resolveHealth(t, eng)
		health, ok := healthByStepID[fl.ID]
		assert.True(t, ok)
		assert.Equal(t, api.HealthHealthy, health.Status)

		assert.NoError(t,
			eng.UpdateStepHealth(goalB.ID, api.HealthUnhealthy, "boom"),
		)

		healthByStepID = resolveHealth(t, eng)
		health, ok = healthByStepID[fl.ID]
		assert.True(t, ok)
		assert.Equal(t, api.HealthUnhealthy, health.Status)
		assert.Contains(t, health.Error, "goal-b")
	})
}

func TestFlowHealthUnknownGoalError(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goal := helpers.NewSimpleStep("goal-unknown")
		fl := &api.Step{
			ID:   "flow-unknown",
			Name: "Flow Unknown",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{goal.ID},
			},
			Attributes: api.AttributeSpecs{
				"out": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		assert.NoError(t, eng.RegisterStep(goal))
		assert.NoError(t, eng.RegisterStep(fl))
		assert.NoError(t,
			eng.UpdateStepHealth(
				goal.ID, api.HealthUnknown, "goal check failed",
			),
		)

		health := resolveHealth(t, eng)
		assert.Equal(t, api.HealthUnknown, health[fl.ID].Status)
		assert.Contains(t, health[fl.ID].Error, "goal-unknown")
	})
}

func TestGetHealthFlowWorstGoal(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goalA := helpers.NewSimpleStep("goal-health-a")
		goalB := helpers.NewSimpleStep("goal-health-b")
		fl := &api.Step{
			ID:   "flow-health-step",
			Name: "Flow Health Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{goalA.ID, goalB.ID},
			},
			Attributes: api.AttributeSpecs{
				"out": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		assert.NoError(t, eng.RegisterStep(goalA))
		assert.NoError(t, eng.RegisterStep(goalB))
		assert.NoError(t, eng.RegisterStep(fl))
		assert.NoError(t,
			eng.UpdateStepHealth(goalA.ID, api.HealthHealthy, ""),
		)
		assert.NoError(t,
			eng.UpdateStepHealth(goalB.ID, api.HealthUnhealthy, "goal down"),
		)

		health := resolveHealth(t, eng)
		assert.Equal(t, api.HealthUnhealthy, health[fl.ID].Status)
		assert.Contains(t, health[fl.ID].Error, "goal-health-b")
	})
}

func TestFlowHealthIncludesPreviewSteps(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		provider := helpers.NewSimpleStep("provider")
		provider.Attributes = api.AttributeSpecs{
			"mid": {Role: api.RoleOutput, Type: api.TypeString},
		}
		goal := helpers.NewSimpleStep("goal")
		goal.Attributes = api.AttributeSpecs{
			"mid": {Role: api.RoleRequired, Type: api.TypeString},
			"out": {Role: api.RoleOutput, Type: api.TypeString},
		}
		fl := &api.Step{
			ID:   "flow-step",
			Name: "Flow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{goal.ID},
			},
			Attributes: api.AttributeSpecs{
				"out": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}

		assert.NoError(t, eng.RegisterStep(provider))
		assert.NoError(t, eng.RegisterStep(goal))
		assert.NoError(t, eng.RegisterStep(fl))

		assert.NoError(t, eng.UpdateStepHealth(goal.ID, api.HealthHealthy, ""))
		assert.NoError(t,
			eng.UpdateStepHealth(
				provider.ID, api.HealthUnhealthy, "provider down",
			),
		)

		health := resolveHealth(t, eng)
		assert.Equal(t, api.HealthUnhealthy, health[fl.ID].Status)
		assert.Contains(t, health[fl.ID].Error, "provider")
	})
}

func TestGetStepHealthNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		health := resolveHealth(t, eng)
		_, ok := health["missing-step"]
		assert.False(t, ok)
	})
}

func TestMergeNodeHealth(t *testing.T) {
	health := engine.MergeNodeHealth(
		map[api.NodeID]map[api.StepID]*api.HealthState{
			"node-b": {
				"step-a": {Status: api.HealthHealthy},
			},
			"node-a": {
				"step-a": {
					Status: api.HealthUnhealthy,
					Error:  "connection refused",
				},
			},
		},
	)

	if assert.Contains(t, health, api.StepID("step-a")) {
		assert.Equal(t, api.HealthUnhealthy, health["step-a"].Status)
		assert.Equal(
			t,
			"node node-a: connection refused",
			health["step-a"].Error,
		)
	}
}

func TestScriptHealthDefaults(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := &api.Step{
			ID:   "script-step",
			Name: "Script Step",
			Type: api.StepTypeScript,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
			Script: &api.ScriptConfig{
				Language: api.ScriptLangAle,
				Script:   "{:result 42}",
			},
		}

		assert.NoError(t, eng.RegisterStep(st))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)

		health := engine.ResolveHealth(cat, map[api.StepID]*api.HealthState{})
		if assert.Contains(t, health, st.ID) {
			assert.Equal(t, api.HealthHealthy, health[st.ID].Status)
		}
	})
}

func resolveHealth(
	t *testing.T, eng *engine.Engine,
) map[api.StepID]*api.HealthState {
	t.Helper()

	cat, err := eng.GetCatalogState()
	assert.NoError(t, err)

	part, err := eng.GetNodeState()
	assert.NoError(t, err)

	return engine.ResolveHealth(cat, part.Health)
}
