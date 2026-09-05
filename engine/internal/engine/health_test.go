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

		cluster, err := eng.GetClusterState()
		assert.NoError(t, err)

		for _, node := range cluster.Nodes {
			if h, ok := node.Health["health-step"]; ok {
				assert.Equal(t, api.HealthHealthy, h.Status)
				return
			}
		}
		assert.Fail(t, "health-step not found in any node")
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

		cluster, err := eng.GetClusterState()
		assert.NoError(t, err)

		for _, node := range cluster.Nodes {
			if h, ok := node.Health["unhealthy-step"]; ok {
				assert.Equal(t, api.HealthUnhealthy, h.Status)
				assert.Equal(t, "connection refused", h.Error)
				return
			}
		}
		assert.Fail(t, "unhealthy-step not found in any node")
	})
}

func TestFlowHealth(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goalA := helpers.NewSimpleStep("goal-a")
		goalB := helpers.NewSimpleStep("goal-b")
		st := &api.Step{
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
		assert.NoError(t, eng.RegisterStep(st))

		assert.NoError(t,
			eng.UpdateStepHealth(goalA.ID, api.HealthHealthy, ""),
		)
		assert.NoError(t,
			eng.UpdateStepHealth(goalB.ID, api.HealthUnknown, ""),
		)

		healthByStepID := resolveHealth(t, eng)
		h, ok := healthByStepID[st.ID]
		assert.True(t, ok)
		assert.Equal(t, api.HealthHealthy, h.Status)

		assert.NoError(t,
			eng.UpdateStepHealth(goalB.ID, api.HealthUnhealthy, "boom"),
		)

		healthByStepID = resolveHealth(t, eng)
		h, ok = healthByStepID[st.ID]
		assert.True(t, ok)
		assert.Equal(t, api.HealthUnhealthy, h.Status)
		assert.Contains(t, h.Error, "goal-b")
	})
}

func TestFlowHealthUnknownGoalError(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goal := helpers.NewSimpleStep("goal-unknown")
		st := &api.Step{
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
		assert.NoError(t, eng.RegisterStep(st))
		assert.NoError(t,
			eng.UpdateStepHealth(
				goal.ID, api.HealthUnknown, "goal check failed",
			),
		)

		h := resolveHealth(t, eng)
		assert.Equal(t, api.HealthUnknown, h[st.ID].Status)
		assert.Contains(t, h[st.ID].Error, "goal-unknown")
	})
}

func TestGetHealthFlowWorstGoal(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goalA := helpers.NewSimpleStep("goal-health-a")
		goalB := helpers.NewSimpleStep("goal-health-b")
		st := &api.Step{
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
		assert.NoError(t, eng.RegisterStep(st))
		assert.NoError(t,
			eng.UpdateStepHealth(goalA.ID, api.HealthHealthy, ""),
		)
		assert.NoError(t,
			eng.UpdateStepHealth(goalB.ID, api.HealthUnhealthy, "goal down"),
		)

		h := resolveHealth(t, eng)
		assert.Equal(t, api.HealthUnhealthy, h[st.ID].Status)
		assert.Contains(t, h[st.ID].Error, "goal-health-b")
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
		st := &api.Step{
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
		assert.NoError(t, eng.RegisterStep(st))

		assert.NoError(t, eng.UpdateStepHealth(goal.ID, api.HealthHealthy, ""))
		assert.NoError(t,
			eng.UpdateStepHealth(
				provider.ID, api.HealthUnhealthy, "provider down",
			),
		)

		h := resolveHealth(t, eng)
		assert.Equal(t, api.HealthUnhealthy, h[st.ID].Status)
		assert.Contains(t, h[st.ID].Error, "provider")
	})
}

func TestGetStepHealthNotFound(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		h := resolveHealth(t, eng)
		_, ok := h["missing-step"]
		assert.False(t, ok)
	})
}

func TestMergeNodeHealth(t *testing.T) {
	cluster := api.ClusterState{
		Nodes: map[api.NodeID]api.NodeState{
			"node-b": {Health: map[api.StepID]api.HealthState{
				"step-a": {Status: api.HealthHealthy},
			}},
			"node-a": {Health: map[api.StepID]api.HealthState{
				"step-a": {
					Status: api.HealthUnhealthy,
					Error:  "connection refused",
				},
			}},
		},
	}
	h := engine.MergeNodeHealth(cluster)

	if assert.Contains(t, h, api.StepID("step-a")) {
		assert.Equal(t, api.HealthUnhealthy, h["step-a"].Status)
		assert.Equal(t,
			"node node-a: connection refused",
			h["step-a"].Error,
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
				Language: api.ScriptLangLua,
				Script:   "return {result = 42}",
			},
		}

		assert.NoError(t, eng.RegisterStep(st))

		cat, err := eng.GetCatalogState()
		assert.NoError(t, err)

		h := eng.ResolveHealth(
			helpers.Matcher(), cat, map[api.StepID]api.HealthState{},
		)
		if assert.Contains(t, h, st.ID) {
			assert.Equal(t, api.HealthHealthy, h[st.ID].Status)
		}
	})
}

func TestScriptHealthOnRegister(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := &api.Step{
			ID:   "script-step",
			Name: "Script Step",
			Type: api.StepTypeScript,
			Attributes: api.AttributeSpecs{
				"result": {Role: api.RoleOutput},
			},
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {result = 42}",
			},
		}

		assert.NoError(t, eng.RegisterStep(st))

		cluster, err := eng.GetClusterState()
		assert.NoError(t, err)

		found := false
		for _, node := range cluster.Nodes {
			h, ok := node.Health[st.ID]
			if !ok {
				continue
			}
			found = true
			assert.Equal(t, api.HealthHealthy, h.Status)
			assert.Empty(t, h.Error)
		}
		assert.True(t, found)
	})
}

func TestResolveHealthNilCat(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		h := eng.ResolveHealth(
			helpers.Matcher(), api.CatalogState{},
			map[api.StepID]api.HealthState{},
		)
		assert.Empty(t, h)
	})
}

func TestResolveHealthPreviewFail(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		cat := api.CatalogState{
			Steps: api.Steps{
				"flow-step": {
					ID:   "flow-step",
					Name: "Flow Step",
					Type: api.StepTypeFlow,
					Flow: &api.FlowConfig{
						Goals: []api.StepID{"missing-goal"},
					},
					Attributes: api.AttributeSpecs{
						"out": {Role: api.RoleOutput, Type: api.TypeString},
					},
				},
			},
			Attributes: api.AttributeGraph{},
		}

		h := eng.ResolveHealth(
			helpers.Matcher(), cat, map[api.StepID]api.HealthState{},
		)
		if assert.Contains(t, h, api.StepID("flow-step")) {
			assert.Equal(t, api.HealthUnknown, h["flow-step"].Status)
			assert.Contains(t, h["flow-step"].Error, "preview failed")
		}
	})
}

func TestResolveHealthSimpleUnknown(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		cat := api.CatalogState{
			Steps: api.Steps{
				"step-a": helpers.NewSimpleStep("step-a"),
			},
			Attributes: api.AttributeGraph{},
		}

		h := eng.ResolveHealth(
			helpers.Matcher(), cat, map[api.StepID]api.HealthState{},
		)
		if assert.Contains(t, h, api.StepID("step-a")) {
			assert.Equal(t, api.HealthUnknown, h["step-a"].Status)
			assert.Empty(t, h["step-a"].Error)
		}
	})
}

func TestResolveHealthScriptError(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		cat := api.CatalogState{
			Steps: api.Steps{
				"script-step": {
					ID:   "script-step",
					Name: "Script Step",
					Type: api.StepTypeScript,
					Attributes: api.AttributeSpecs{
						"result": {Role: api.RoleOutput},
					},
					Script: &api.ScriptConfig{
						Language: api.ScriptLangLua,
						Script:   "return {result = 42}",
					},
				},
			},
			Attributes: api.AttributeGraph{},
		}
		base := map[api.StepID]api.HealthState{
			"script-step": {
				Status: api.HealthUnknown,
				Error:  "compile failed",
			},
		}

		h := eng.ResolveHealth(helpers.Matcher(), cat, base)
		if assert.Contains(t, h, api.StepID("script-step")) {
			assert.Equal(t, api.HealthUnknown, h["script-step"].Status)
			assert.Equal(t, "compile failed", h["script-step"].Error)
		}
	})
}

func TestResolveHealthFlowUnknown(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goal := helpers.NewSimpleStep("goal-a")
		st := &api.Step{
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
		cat := api.CatalogState{
			Steps: api.Steps{
				goal.ID: goal,
				st.ID:   st,
			},
			Attributes: api.AttributeGraph{},
		}
		base := map[api.StepID]api.HealthState{
			goal.ID: {Status: api.HealthUnknown},
		}

		h := eng.ResolveHealth(helpers.Matcher(), cat, base)
		if assert.Contains(t, h, st.ID) {
			assert.Equal(t, api.HealthHealthy, h[st.ID].Status)
			assert.Empty(t, h[st.ID].Error)
		}
	})
}

func TestMergeNodeHealthUnknown(t *testing.T) {
	cluster := api.ClusterState{
		Nodes: map[api.NodeID]api.NodeState{
			"node-a": {Health: map[api.StepID]api.HealthState{
				"step-a": {Status: api.HealthUnknown},
			}},
			"node-b": {Health: map[api.StepID]api.HealthState{
				"step-a": {
					Status: api.HealthUnknown,
					Error:  "late report",
				},
			}},
		},
	}

	h := engine.MergeNodeHealth(cluster)
	if assert.Contains(t, h, api.StepID("step-a")) {
		assert.Equal(t, api.HealthUnknown, h["step-a"].Status)
		assert.Equal(t, "node node-b: late report", h["step-a"].Error)
	}
}

func TestResolveHealthFlowUnhealthyChild(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goal := helpers.NewSimpleStep("unhealthy-goal")
		st := &api.Step{
			ID:   "unhealthy-flow-step",
			Name: "Flow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{goal.ID},
			},
			Attributes: api.AttributeSpecs{
				"out": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}
		cat := api.CatalogState{
			Steps: api.Steps{
				goal.ID: goal,
				st.ID:   st,
			},
			Attributes: api.AttributeGraph{},
		}
		base := map[api.StepID]api.HealthState{
			goal.ID: {Status: api.HealthUnhealthy, Error: "endpoint down"},
		}

		h := eng.ResolveHealth(helpers.Matcher(), cat, base)
		if assert.Contains(t, h, st.ID) {
			assert.Equal(t, api.HealthUnhealthy, h[st.ID].Status)
			assert.Equal(t,
				"step unhealthy-goal: endpoint down", h[st.ID].Error)
		}
	})
}

func TestResolveHealthUnhealthyWithoutReason(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		goal := helpers.NewSimpleStep("silent-goal")
		st := &api.Step{
			ID:   "silent-flow-step",
			Name: "Flow Step",
			Type: api.StepTypeFlow,
			Flow: &api.FlowConfig{
				Goals: []api.StepID{goal.ID},
			},
			Attributes: api.AttributeSpecs{
				"out": {Role: api.RoleOutput, Type: api.TypeString},
			},
		}
		cat := api.CatalogState{
			Steps: api.Steps{
				goal.ID: goal,
				st.ID:   st,
			},
			Attributes: api.AttributeGraph{},
		}
		base := map[api.StepID]api.HealthState{
			goal.ID: {Status: api.HealthUnhealthy},
		}

		h := eng.ResolveHealth(helpers.Matcher(), cat, base)
		if assert.Contains(t, h, st.ID) {
			assert.Equal(t, api.HealthUnhealthy, h[st.ID].Status)
			assert.Equal(t, "step silent-goal unhealthy", h[st.ID].Error)
		}
	})
}

func TestResolveHealthBaseWins(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("reported-step")
		cat := api.CatalogState{
			Steps:      api.Steps{st.ID: st},
			Attributes: api.AttributeGraph{},
		}
		base := map[api.StepID]api.HealthState{
			st.ID: {Status: api.HealthHealthy},
		}

		// A node has reported on this step, so no handler is consulted
		h := eng.ResolveHealth(helpers.Matcher(), cat, base)
		if assert.Contains(t, h, st.ID) {
			assert.Equal(t, api.HealthHealthy, h[st.ID].Status)
			assert.Empty(t, h[st.ID].Error)
		}
	})
}

func TestGetClusterEvents(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := helpers.NewSimpleStep("cluster-event-step")
		assert.NoError(t, eng.RegisterStep(st))
		assert.NoError(t, eng.UpdateStepHealth(
			st.ID, api.HealthUnhealthy, "down",
		))

		evs, err := eng.GetClusterEvents()
		assert.NoError(t, err)
		assert.NotEmpty(t, evs)
	})
}

func TestStepHealthFromHandler(t *testing.T) {
	helpers.WithEngine(t, func(eng *engine.Engine) {
		st := &api.Step{
			ID:   "handler-health-step",
			Name: "Script Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}
		assert.NoError(t, eng.RegisterStep(st))

		h, err := eng.StepHealth(st)
		assert.NoError(t, err)
		assert.Equal(t, api.HealthHealthy, h.Status)
	})
}

func resolveHealth(
	t *testing.T, eng *engine.Engine,
) map[api.StepID]api.HealthState {
	t.Helper()

	cat, err := eng.GetCatalogState()
	assert.NoError(t, err)

	cluster, err := eng.GetClusterState()
	assert.NoError(t, err)

	merged := engine.MergeNodeHealth(cluster)
	return eng.ResolveHealth(eng.Matcher, cat, merged)
}
