package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestConstAttribute verifies const attributes always use their default value
func TestConstAttribute(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		step := helpers.NewScriptStep(
			"step-const",
			api.ScriptLangAle,
			`{:result const_value}`,
			"result",
		)
		step.Attributes["const_value"] = &api.AttributeSpec{
			Role:    api.RoleConst,
			Type:    api.TypeString,
			Default: `"fixed"`,
		}
		step.Attributes["result"].Type = api.TypeString

		assert.NoError(t, env.Engine.RegisterStep(step))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
			Attributes: api.AttributeGraph{
				"result": {
					Providers: []api.StepID{step.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		flowID := api.FlowID("test-const-attribute")
		flow := env.WaitForFlowStatus(flowID, func() {
			err := env.Engine.StartFlow(flowID, plan,
				flowopt.WithInit(api.Args{"const_value": "override"}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, flow.Status)
		assert.Equal(t, "fixed", flow.Attributes["result"].Value)
	})
}
