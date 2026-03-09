package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestDependencyChain tests that a linear dependency chain A→B→C executes
// correctly with proper attribute propagation
func TestDependencyChain(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Step A: No inputs, produces "valueA"
		stepA := helpers.NewStepWithOutputs("step-a", "valueA")

		// Step B: Requires "valueA", produces "valueB"
		stepB := helpers.NewTestStepWithArgs([]api.Name{"valueA"}, nil)
		stepB.ID = "step-b"
		stepB.Attributes["valueB"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Step C (Goal): Requires "valueB", produces "result"
		stepC := helpers.NewTestStepWithArgs([]api.Name{"valueB"}, nil)
		stepC.ID = "step-c"
		stepC.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))
		assert.NoError(t, env.Engine.RegisterStep(stepC))

		// Set mock responses
		env.MockClient.SetResponse("step-a", api.Args{"valueA": "from-A"})
		env.MockClient.SetResponse("step-b", api.Args{"valueB": "from-B"})
		env.MockClient.SetResponse("step-c", api.Args{"result": "done"})

		// Create execution plan with step-c as goal
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-c"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
				"step-c": stepC,
			},
			Attributes: api.AttributeGraph{
				"valueA": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-b"},
				},
				"valueB": &api.AttributeEdges{
					Providers: []api.StepID{"step-b"},
					Consumers: []api.StepID{"step-c"},
				},
			},
		}

		id := api.FlowID("test-dependency-chain")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		// Verify all steps completed
		assert.Equal(t, api.StepCompleted, fl.Executions["step-a"].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions["step-b"].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions["step-c"].Status)

		// Verify attribute propagation
		assert.Equal(t, "from-A", fl.Attributes["valueA"].Value)
		assert.Equal(t, "from-B", fl.Attributes["valueB"].Value)
		assert.Equal(t, "done", fl.Attributes["result"].Value)

		// Verify invocation order (all should be invoked)
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 3)
		assert.Contains(t, invocations, api.StepID("step-a"))
		assert.Contains(t, invocations, api.StepID("step-b"))
		assert.Contains(t, invocations, api.StepID("step-c"))
	})
}

// TestDiamondDependencies tests that a diamond dependency pattern A→B,C→D
// executes correctly with parallel execution of B and C
func TestDiamondDependencies(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		// Step A: Produces both "valueB" and "valueC"
		stepA := helpers.NewStepWithOutputs("step-a", "valueB", "valueC")

		// Step B: Requires "valueB", produces "outputB"
		stepB := helpers.NewTestStepWithArgs([]api.Name{"valueB"}, nil)
		stepB.ID = "step-b"
		stepB.Attributes["outputB"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Step C: Requires "valueC", produces "outputC"
		stepC := helpers.NewTestStepWithArgs([]api.Name{"valueC"}, nil)
		stepC.ID = "step-c"
		stepC.Attributes["outputC"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		// Step D (Goal): Requires both "outputB" and "outputC"
		stepD := helpers.NewTestStepWithArgs(
			[]api.Name{"outputB", "outputC"}, nil,
		)
		stepD.ID = "step-d"
		stepD.Attributes["final"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}

		assert.NoError(t, env.Engine.RegisterStep(stepA))
		assert.NoError(t, env.Engine.RegisterStep(stepB))
		assert.NoError(t, env.Engine.RegisterStep(stepC))
		assert.NoError(t, env.Engine.RegisterStep(stepD))

		// Set mock responses
		env.MockClient.SetResponse("step-a", api.Args{
			"valueB": "b-val",
			"valueC": "c-val",
		})
		env.MockClient.SetResponse("step-b", api.Args{"outputB": "B-result"})
		env.MockClient.SetResponse("step-c", api.Args{"outputC": "C-result"})
		env.MockClient.SetResponse("step-d", api.Args{"final": "done"})

		// Create execution plan with step-d as goal
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"step-d"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
				"step-c": stepC,
				"step-d": stepD,
			},
			Attributes: api.AttributeGraph{
				"valueB": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-b"},
				},
				"valueC": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-c"},
				},
				"outputB": &api.AttributeEdges{
					Providers: []api.StepID{"step-b"},
					Consumers: []api.StepID{"step-d"},
				},
				"outputC": &api.AttributeEdges{
					Providers: []api.StepID{"step-c"},
					Consumers: []api.StepID{"step-d"},
				},
			},
		}

		id := api.FlowID("test-diamond")
		fl := env.WaitForFlowStatus(id, func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.Equal(t, api.FlowCompleted, fl.Status)

		// Verify all steps completed
		assert.Equal(t, api.StepCompleted, fl.Executions["step-a"].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions["step-b"].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions["step-c"].Status)
		assert.Equal(t, api.StepCompleted, fl.Executions["step-d"].Status)

		// Verify attribute propagation
		assert.Equal(t, "b-val", fl.Attributes["valueB"].Value)
		assert.Equal(t, "c-val", fl.Attributes["valueC"].Value)
		assert.Equal(t, "B-result", fl.Attributes["outputB"].Value)
		assert.Equal(t, "C-result", fl.Attributes["outputC"].Value)
		assert.Equal(t, "done", fl.Attributes["final"].Value)

		// Verify all steps were invoked
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 4)
		assert.Contains(t, invocations, api.StepID("step-a"))
		assert.Contains(t, invocations, api.StepID("step-b"))
		assert.Contains(t, invocations, api.StepID("step-c"))
		assert.Contains(t, invocations, api.StepID("step-d"))
	})
}
