package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestMemoizationHit(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		s := helpers.NewTestStepWithArgs(nil, nil)
		s.ID = "memo"
		s.Memoizable = true
		s.Attributes["out"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(s))
		env.MockClient.SetResponse("memo", api.Args{"out": "v1"})

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"memo"},
			Steps: api.Steps{"memo": s},
		}

		id1 := api.FlowID("f1")
		f1 := env.WaitForFlowStatus(id1, func() {
			err := env.Engine.StartFlow(id1, plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, f1.Status)
		assert.Equal(t, "v1", f1.Attributes["out"].Value)
		assert.True(t, env.MockClient.WasInvoked("memo"))

		id2 := api.FlowID("f2")
		f2 := env.WaitForFlowStatus(id2, func() {
			err := env.Engine.StartFlow(id2, plan)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, f2.Status)
		assert.Equal(t, "v1", f2.Attributes["out"].Value)

		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1, "step should only be invoked once")
	})
}

func TestMemoizationMiss(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()

		s := helpers.NewTestStepWithArgs([]api.Name{"in"}, nil)
		s.ID = "memo"
		s.Memoizable = true
		s.Attributes["out"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(s))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"memo"},
			Steps: api.Steps{"memo": s},
		}

		env.MockClient.SetResponse("memo", api.Args{"out": "a"})
		id1 := api.FlowID("f1")
		f1 := env.WaitForFlowStatus(id1, func() {
			err := env.Engine.StartFlow(id1, plan,
				flowopt.WithInit(api.Args{"in": "a"}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, f1.Status)
		assert.Equal(t, "a", f1.Attributes["out"].Value)

		env.MockClient.SetResponse("memo", api.Args{"out": "b"})
		id2 := api.FlowID("f2")
		f2 := env.WaitForFlowStatus(id2, func() {
			err := env.Engine.StartFlow(id2, plan,
				flowopt.WithInit(api.Args{"in": "b"}),
			)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, f2.Status)
		assert.Equal(t, "b", f2.Attributes["out"].Value)
	})
}
