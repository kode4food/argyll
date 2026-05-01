package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestMemoizationHit(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		s := helpers.NewTestStepWithArgs(nil, nil)
		s.ID = "memo"
		s.Memoizable = true
		s.Attributes["out"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(s))
		env.MockClient.SetResponse("memo", api.Args{"out": "v1"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"memo"},
			Steps: api.Steps{"memo": s},
		}

		id1 := api.FlowID("f1")
		err := env.Engine.StartFlow(id1, pl)
		assert.NoError(t, err)
		f1 := env.WaitForTerminalFlow(id1)
		assert.Equal(t, api.FlowCompleted, f1.Status)
		assert.Equal(t, "v1", f1.Attributes["out"][0].Value)
		assert.True(t, env.MockClient.WasInvoked("memo"))

		id2 := api.FlowID("f2")
		err = env.Engine.StartFlow(id2, pl)
		assert.NoError(t, err)
		f2 := env.WaitForTerminalFlow(id2)
		assert.Equal(t, api.FlowCompleted, f2.Status)
		assert.Equal(t, "v1", f2.Attributes["out"][0].Value)

		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1, "step should only be invoked once")
	})
}

func TestMemoizationMiss(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		s := helpers.NewTestStepWithArgs([]api.Name{"in"}, nil)
		s.ID = "memo"
		s.Memoizable = true
		s.Attributes["out"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(s))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"memo"},
			Steps: api.Steps{"memo": s},
		}

		env.MockClient.SetResponse("memo", api.Args{"out": "a"})
		id1 := api.FlowID("f1")
		err := env.Engine.StartFlow(id1, pl,
			flow.WithInit(api.InitArgs{"in": {"a"}}),
		)
		assert.NoError(t, err)
		f1 := env.WaitForTerminalFlow(id1)
		assert.Equal(t, api.FlowCompleted, f1.Status)
		assert.Equal(t, "a", f1.Attributes["out"][0].Value)

		env.MockClient.SetResponse("memo", api.Args{"out": "b"})
		id2 := api.FlowID("f2")
		err = env.Engine.StartFlow(id2, pl,
			flow.WithInit(api.InitArgs{"in": {"b"}}),
		)
		assert.NoError(t, err)
		f2 := env.WaitForTerminalFlow(id2)
		assert.Equal(t, api.FlowCompleted, f2.Status)
		assert.Equal(t, "b", f2.Attributes["out"][0].Value)
	})
}
