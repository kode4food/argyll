package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestMixedStepTypes tests that HTTP and Script steps can work together in a
// single workflow
func TestMixedStepTypes(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	// Step A (Sync HTTP): No inputs, produces "valueA"
	stepA := helpers.NewStepWithOutputs("step-a", "valueA")

	// Step B (Script): Require "valueA", produce "valueB" by mapping input
	stepB := helpers.NewScriptStep(
		"step-b", api.ScriptLangAle,
		`{:valueB (str "transformed-" valueA)}`,
		"valueB",
	)
	stepB.Attributes["valueA"] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}
	stepB.Attributes["valueB"].Type = api.TypeString

	// Step C (Sync HTTP): Requires "valueB", produces "result"
	stepC := helpers.NewTestStepWithArgs([]api.Name{"valueB"}, nil)
	stepC.ID = "step-c"
	stepC.Attributes["result"] = &api.AttributeSpec{
		Role: api.RoleOutput,
		Type: api.TypeString,
	}

	assert.NoError(t, env.Engine.RegisterStep(ctx, stepA))
	assert.NoError(t, env.Engine.RegisterStep(ctx, stepB))
	assert.NoError(t, env.Engine.RegisterStep(ctx, stepC))

	// Set mock responses (only for HTTP steps - script executes inline)
	env.MockClient.SetResponse("step-a", api.Args{"valueA": "data"})
	env.MockClient.SetResponse("step-c", api.Args{"result": "done"})

	// Create execution plan with step-c as goal
	plan := &api.ExecutionPlan{
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

	flowID := api.FlowID("test-mixed-types")
	err := env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	// Wait for workflow completion
	flow := env.WaitForFlowStatus(t, ctx, flowID, workflowTimeout)
	assert.Equal(t, api.FlowCompleted, flow.Status)

	// Verify all steps completed
	assert.Equal(t, api.StepCompleted, flow.Executions["step-a"].Status)
	assert.Equal(t, api.StepCompleted, flow.Executions["step-b"].Status)
	assert.Equal(t, api.StepCompleted, flow.Executions["step-c"].Status)

	// Verify attribute propagation and transformation
	assert.Equal(t, "data", flow.Attributes["valueA"].Value)
	assert.Equal(t, "transformed-data", flow.Attributes["valueB"].Value)
	assert.Equal(t, "done", flow.Attributes["result"].Value)

	// Verify HTTP steps invoked (script step executes inline, not via HTTP)
	invocations := env.MockClient.GetInvocations()
	assert.Len(t, invocations, 2)
	assert.Contains(t, invocations, api.StepID("step-a"))
	assert.NotContains(t, invocations, api.StepID("step-b"))
	assert.Contains(t, invocations, api.StepID("step-c"))
}
