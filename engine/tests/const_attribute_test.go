package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestConstAttribute verifies const attributes always use their default value
func TestConstAttribute(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewScriptStep(
			"step-const",
			api.ScriptLangAle,
			`{:result const_value}`,
			"result",
		)
		st.Attributes["const_value"] = &api.AttributeSpec{
			Role:    api.RoleConst,
			Type:    api.TypeString,
			Default: `"fixed"`,
		}
		st.Attributes["result"].Type = api.TypeString

		assert.NoError(t, env.Engine.RegisterStep(st))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
			Attributes: api.AttributeGraph{
				"result": {
					Providers: []api.StepID{st.ID},
					Consumers: []api.StepID{},
				},
			},
		}

		id := api.FlowID("test-const-attribute")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl,
				flow.WithInit(api.Args{"const_value": "override"}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, "fixed", fl.Attributes["result"].Value)
	})
}
