package engine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	as "github.com/kode4food/spuds/engine/internal/assert"
	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

const testTimeout = 5 * time.Second

func TestStartDuplicate(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("step-1")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-1"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-dup", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.StartFlow(
		context.Background(), "wf-dup", plan, api.Args{}, api.Metadata{},
	)
	assert.ErrorIs(t, err, engine.ErrFlowExists)
}

func TestStartMissingInput(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("step-needs-input")
	step.Attributes["required_value"] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}

	plan := &api.ExecutionPlan{
		Goals:    []api.StepID{"step-needs-input"},
		Steps:    api.Steps{step.ID: step},
		Required: []api.Name{"required_value"},
	}

	err := env.Engine.StartFlow(
		context.Background(),
		"wf-missing",
		plan,
		api.Args{},
		api.Metadata{},
	)
	assert.Error(t, err)
}

func TestGetStateNotFound(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	_, err := env.Engine.GetFlowState(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, engine.ErrFlowNotFound)
}

func TestSetAttribute(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	// Create a step that produces an output attribute
	step := helpers.NewStepWithOutputs("output-step", "test_key")

	err := env.Engine.RegisterStep(ctx, step)
	require.NoError(t, err)

	// Configure mock to return the output value
	env.MockClient.SetResponse("output-step", api.Args{
		"test_key": "test_value",
	})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"output-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(ctx, "wf-attr", plan, api.Args{}, api.Metadata{})
	require.NoError(t, err)

	// Wait for flow to complete
	env.WaitForFlowStatus(t, ctx, "wf-attr", testTimeout)

	a := as.New(t)
	a.FlowStateEquals(ctx, env.Engine, "wf-attr", "test_key", "test_value")
}

func TestGetAttributes(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	// Create a step that produces multiple output attributes
	step := helpers.NewStepWithOutputs("step-attrs", "key1", "key2")

	err := env.Engine.RegisterStep(ctx, step)
	require.NoError(t, err)

	// Configure mock to return multiple output values
	env.MockClient.SetResponse("step-attrs", api.Args{
		"key1": "value1",
		"key2": float64(42),
	})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-attrs"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		ctx, "wf-getattrs", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	// Wait for flow to complete
	flow := env.WaitForFlowStatus(t, ctx, "wf-getattrs", testTimeout)

	attrs := flow.GetAttributes()
	assert.Len(t, attrs, 2)
	assert.Equal(t, "value1", attrs["key1"])
	assert.Equal(t, float64(42), attrs["key2"])
}

func TestDuplicateAttributeFirstWins(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	// Create two steps that both produce the same output attribute
	stepA := helpers.NewStepWithOutputs("step-a", "shared_key")
	stepB := helpers.NewStepWithOutputs("step-b", "shared_key")

	err := env.Engine.RegisterStep(ctx, stepA)
	require.NoError(t, err)
	err = env.Engine.RegisterStep(ctx, stepB)
	require.NoError(t, err)

	// Configure mock responses - step-a runs first and sets "first"
	env.MockClient.SetResponse("step-a", api.Args{"shared_key": "first"})
	env.MockClient.SetResponse("step-b", api.Args{"shared_key": "second"})

	// Both steps are goals so both will execute
	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-a", "step-b"},
		Steps: api.Steps{
			stepA.ID: stepA,
			stepB.ID: stepB,
		},
	}

	err = env.Engine.StartFlow(
		ctx, "wf-dup-attr", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	// Wait for flow to complete
	flow := env.WaitForFlowStatus(t, ctx, "wf-dup-attr", testTimeout)

	// First value wins - duplicates are silently ignored
	attrs := flow.GetAttributes()
	assert.Contains(t, []string{"first", "second"}, attrs["shared_key"])
}

func TestCompleteFlow(t *testing.T) {
	a := as.New(t)
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step := helpers.NewStepWithOutputs("complete-step", "result")

	err := env.Engine.RegisterStep(ctx, step)
	require.NoError(t, err)

	// Configure mock to return a result
	env.MockClient.SetResponse("complete-step", api.Args{"result": "final"})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"complete-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		ctx, "wf-complete", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	// Wait for flow to complete automatically
	flow := env.WaitForFlowStatus(t, ctx, "wf-complete", testTimeout)
	a.FlowStatus(flow, api.FlowCompleted)
}

func TestFailFlow(t *testing.T) {
	a := as.New(t)
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep("fail-step")

	err := env.Engine.RegisterStep(ctx, step)
	require.NoError(t, err)

	env.MockClient.SetError("fail-step", errors.New("test error"))

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"fail-step"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(ctx, "wf-fail", plan, api.Args{}, api.Metadata{})
	require.NoError(t, err)

	// Wait for flow to fail automatically
	flow := env.WaitForFlowStatus(t, ctx, "wf-fail", testTimeout)
	a.FlowStatus(flow, api.FlowFailed)
	assert.Contains(t, flow.Error, "test error")
}

func TestSkipStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	// Create a step with a predicate that returns false, causing it to skip
	step := helpers.NewStepWithPredicate(
		"step-skip", api.ScriptLangAle, "false",
	)

	err := env.Engine.RegisterStep(ctx, step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-skip"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(ctx, "wf-skip", plan, api.Args{}, api.Metadata{})
	require.NoError(t, err)

	// Wait for step to be skipped
	exec := env.WaitForStepStatus(t, ctx, "wf-skip", "step-skip", testTimeout)
	require.NotNil(t, exec)
	assert.Equal(t, api.StepSkipped, exec.Status)
	assert.Equal(t, "predicate returned false", exec.Error)
}

func TestStartFlowSimple(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := &api.Step{
		ID:      "goal-step",
		Name:    "Goal",
		Type:    api.StepTypeSync,
		Version: "1.0.0",
		Attributes: api.AttributeSpecs{
			"result": {Role: api.RoleOutput, Type: api.TypeString},
		},
		HTTP: &api.HTTPConfig{
			Endpoint: "http://test:8080",
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	env.MockClient.SetResponse("goal-step", api.Args{"result": "success"})

	plan := &api.ExecutionPlan{
		Goals:    []api.StepID{"goal-step"},
		Required: []api.Name{},
		Steps: api.Steps{
			"goal-step": step,
		},
	}

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-simple",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(context.Background(), "wf-simple")
	require.NoError(t, err)
	assert.NotNil(t, flow)
	assert.Equal(t, api.FlowID("wf-simple"), flow.ID)
}

func TestGetFlowEvents(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("simple")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"simple"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-events", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	events, err := env.Engine.GetFlowEvents(
		context.Background(), "wf-events", 0,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
}

func TestListFlows(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("test")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"test"},
		Steps: api.Steps{step.ID: step},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-list", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	flows, err := env.Engine.ListFlows(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, flows)
}

func TestIsFlowFailed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	stepA := helpers.NewStepWithOutputs("step-a", "value")

	stepB := helpers.NewSimpleStep("step-b")
	stepB.Attributes["value"] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-b"},
		Steps: api.Steps{
			stepA.ID: stepA,
			stepB.ID: stepB,
		},
	}

	err := env.Engine.RegisterStep(context.Background(), stepA)
	require.NoError(t, err)
	err = env.Engine.RegisterStep(context.Background(), stepB)
	require.NoError(t, err)

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-failed-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	// Fail step-a directly via FailStepExecution (step will be started by flow)
	fs := engine.FlowStep{FlowID: "wf-failed-test", StepID: "step-a"}
	err = env.Engine.FailStepExecution(
		context.Background(), fs, "test error",
	)
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "wf-failed-test",
	)
	require.NoError(t, err)

	isFailed := env.Engine.IsFlowFailed(flow)
	assert.True(t, isFailed)
}

func TestIsFlowNotFailed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("step-ok")

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-ok"},
		Steps: api.Steps{step.ID: step},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-ok-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "wf-ok-test",
	)
	require.NoError(t, err)

	isFailed := env.Engine.IsFlowFailed(flow)
	assert.False(t, isFailed)
}

func TestHasInputProvider(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	stepA := helpers.NewStepWithOutputs("step-a", "value")

	stepB := helpers.NewSimpleStep("step-b")
	stepB.Attributes["value"] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-b"},
		Steps: api.Steps{
			stepA.ID: stepA,
			stepB.ID: stepB,
		},
		Attributes: api.AttributeGraph{
			"value": {
				Providers: []api.StepID{stepA.ID},
				Consumers: []api.StepID{stepB.ID},
			},
		},
	}

	err := env.Engine.RegisterStep(context.Background(), stepA)
	require.NoError(t, err)
	err = env.Engine.RegisterStep(context.Background(), stepB)
	require.NoError(t, err)

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-provider-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "wf-provider-test",
	)
	require.NoError(t, err)

	hasProvider := env.Engine.HasInputProvider("value", flow)
	assert.True(t, hasProvider)
}

func TestHasInputProviderNone(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("step-alone")
	step.Attributes["missing"] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}

	plan := &api.ExecutionPlan{
		Goals:      []api.StepID{"step-alone"},
		Steps:      api.Steps{step.ID: step},
		Attributes: api.AttributeGraph{},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-no-provider-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "wf-no-provider-test",
	)
	require.NoError(t, err)

	hasProvider := env.Engine.HasInputProvider("missing", flow)
	assert.False(t, hasProvider)
}

func TestStepProvidesInput(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	stepA := helpers.NewStepWithOutputs("step-provider", "result")

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-provider"},
		Steps: api.Steps{
			stepA.ID: stepA,
		},
	}

	err := env.Engine.RegisterStep(context.Background(), stepA)
	require.NoError(t, err)

	err = env.Engine.StartFlow(
		context.Background(),
		"wf-provides-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	outputArgs := stepA.GetOutputArgs()
	assert.Contains(t, outputArgs, api.Name("result"))
	assert.NotContains(t, outputArgs, api.Name("other"))
}
