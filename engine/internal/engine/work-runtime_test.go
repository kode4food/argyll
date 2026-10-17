package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/flow"
)

func TestInlineScriptChain(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		double := &api.Step{
			ID:   "double",
			Name: "Double",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return { y = x * 2 }",
			},
			Attributes: api.AttributeSpecs{
				"x": {Role: api.RoleRequired, Type: api.TypeNumber},
				"y": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
		}
		bad := &api.Step{
			ID:   "bad",
			Name: "Bad",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "error('boom')",
			},
			Attributes: api.AttributeSpecs{
				"y": {Role: api.RoleRequired, Type: api.TypeNumber},
				"z": {Role: api.RoleOutput, Type: api.TypeNumber},
			},
		}
		assert.NoError(t, env.Engine.RegisterStep(double))
		assert.NoError(t, env.Engine.RegisterStep(bad))

		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{bad.ID}},
			Steps: api.Steps{double.ID: double, bad.ID: bad},
			Attributes: api.AttributeGraph{
				"x": {Consumers: []api.StepID{double.ID}},
				"y": {
					Providers: []api.StepID{double.ID},
					Consumers: []api.StepID{bad.ID},
				},
				"z": {Providers: []api.StepID{bad.ID}},
			},
			Required: []api.Name{"x"},
		}
		assert.NoError(t, env.Engine.StartPlan("wf-inline", pl,
			flow.WithInit(api.InitArgs{"x": {float64(2)}}),
		))

		fl, err := env.Engine.GetFlowState("wf-inline")
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)
		assert.Equal(t, api.StepCompleted, fl.Executions[double.ID].Status)
		assert.EqualValues(t, 4, fl.Executions[double.ID].Outputs["y"])
		assert.Equal(t, api.StepFailed, fl.Executions[bad.ID].Status)
	})
}
