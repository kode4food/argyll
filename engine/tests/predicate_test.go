package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestPredicateSkipping tests that steps with false predicates are skipped
// and that workflows fail when goal steps cannot complete due to skipped
// dependencies
func TestPredicateSkipping(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		env.Engine.Start()
		ctx := context.Background()

		// Step A: No inputs, produces "valueA"
		stepA := helpers.NewStepWithOutputs("step-a", "valueA")

		// Step B: Has predicate that returns false, produces "valueB"
		stepB := helpers.NewStepWithPredicate(
			"step-b", api.ScriptLangAle, "false", "valueB",
		)
		stepB.Attributes["valueA"] = &api.AttributeSpec{
			Role: api.RoleRequired,
			Type: api.TypeString,
		}
		stepB.Attributes["valueB"].Type = api.TypeString

		// Set mock responses (though B won't execute due to predicate)
		env.MockClient.SetResponse("step-a", api.Args{"valueA": "from-A"})
		env.MockClient.SetResponse("step-b", api.Args{"valueB": "from-B"})

		// Create execution plan with step-b as goal (it will be skipped)
		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"step-b"},
			Steps: api.Steps{
				"step-a": stepA,
				"step-b": stepB,
			},
			Attributes: api.AttributeGraph{
				"valueA": &api.AttributeEdges{
					Providers: []api.StepID{"step-a"},
					Consumers: []api.StepID{"step-b"},
				},
			},
		}

		flowID := api.FlowID("test-predicate-skip")
		err := env.Engine.RegisterStep(ctx, stepA)
		assert.NoError(t, err)
		err = env.Engine.RegisterStep(ctx, stepB)
		assert.NoError(t, err)
		err = env.Engine.StartFlow(
			ctx, flowID, plan, api.Args{}, api.Metadata{},
		)
		assert.NoError(t, err)

		// Wait for step A to complete
		env.WaitForStepStatus(t, ctx, flowID, "step-a", workflowTimeout)

		// Wait for step B to be skipped
		execB := env.WaitForStepStatus(
			t, ctx, flowID, "step-b", workflowTimeout,
		)
		assert.Equal(t, api.StepSkipped, execB.Status)
		assert.Equal(t, "predicate returned false", execB.Error)

		// Get final flow state
		flow, err := env.Engine.GetFlowState(ctx, flowID)
		assert.NoError(t, err)

		// Verify step A completed
		assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)

		// Verify step B was skipped
		assert.Equal(t, api.StepSkipped, flow.Executions["step-b"].Status)

		// Verify only step A was invoked (B's predicate prevented execution)
		invocations := env.MockClient.GetInvocations()
		assert.Len(t, invocations, 1)
		assert.Contains(t, invocations, api.StepID("step-a"))
		assert.NotContains(t, invocations, api.StepID("step-b"))

		// Verify only step A's output attribute was set
		assert.Equal(t, "from-A", flow.Attributes["valueA"].Value)
		assert.NotContains(t, flow.Attributes, "valueB")
	})
}
