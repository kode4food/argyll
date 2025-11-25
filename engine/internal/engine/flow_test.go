package engine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	as "github.com/kode4food/spuds/engine/internal/assert"
	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestStartDuplicate(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("step-1")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-1"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-dup", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.StartFlow(
		context.Background(), "wf-dup", plan, api.Args{}, api.Metadata{},
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "flow exists")
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
		Goals: []api.StepID{"step-needs-input"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "flow not found")
}

func TestSetAttribute(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("simple-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"simple-step"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-attr", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	fs := engine.FlowStep{FlowID: "wf-attr", StepID: "step-attr"}
	err = env.Engine.SetAttribute(
		context.Background(), fs, "test_key", "test_value",
	)
	require.NoError(t, err)

	a := as.New(t)
	a.FlowStateEquals(
		context.Background(), env.Engine, "wf-attr", "test_key", "test_value",
	)
}

func TestGetAttributes(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("step-attrs")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-attrs"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-getattrs", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	fs := engine.FlowStep{FlowID: "wf-getattrs", StepID: "step-attrs"}
	err = env.Engine.SetAttribute(
		context.Background(), fs, "key1", "value1",
	)
	require.NoError(t, err)

	err = env.Engine.SetAttribute(
		context.Background(), fs, "key2", 42,
	)
	require.NoError(t, err)

	attrs, err := env.Engine.GetAttributes(context.Background(), "wf-getattrs")
	require.NoError(t, err)
	assert.Len(t, attrs, 2)
	assert.Equal(t, "value1", attrs["key1"])
	assert.Equal(t, float64(42), attrs["key2"])
}

func TestSetDuplicate(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("step-dup-attr")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-dup-attr"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-dup-attr", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	fs := engine.FlowStep{FlowID: "wf-dup-attr", StepID: "step-dup-attr"}
	err = env.Engine.SetAttribute(
		context.Background(), fs, "dup_key", "first",
	)
	require.NoError(t, err)

	err = env.Engine.SetAttribute(
		context.Background(), fs, "dup_key", "second",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attribute already set")
}

func TestCompleteFlow(t *testing.T) {
	a := as.New(t)
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("complete-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"complete-step"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-complete", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	result := api.Args{"final": "result"}
	err = env.Engine.CompleteFlow(
		context.Background(), "wf-complete", result,
	)
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "wf-complete",
	)
	require.NoError(t, err)
	a.FlowStatus(flow, api.FlowCompleted)
}

func TestFailFlow(t *testing.T) {
	a := as.New(t)
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("fail-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"fail-step"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-fail", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.FailFlow(context.Background(), "wf-fail", "test error")
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "wf-fail",
	)
	require.NoError(t, err)
	a.FlowStatus(flow, api.FlowFailed)
	a.Equal("test error", flow.Error)
}

func TestSkipStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("step-skip")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{"step-skip"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-skip", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	fs := engine.FlowStep{FlowID: "wf-skip", StepID: "step-skip"}
	err = env.Engine.SkipStepExecution(
		context.Background(), fs, "test skip reason",
	)
	require.NoError(t, err)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "wf-skip",
	)
	require.NoError(t, err)

	exec, ok := flow.Executions["step-skip"]
	require.True(t, ok)
	assert.Equal(t, api.StepSkipped, exec.Status)
	assert.Equal(t, "test skip reason", exec.Error)
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
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
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
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
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
		Steps: map[api.StepID]*api.StepInfo{
			stepA.ID: {Step: stepA},
			stepB.ID: {Step: stepB},
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
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
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
		Steps: map[api.StepID]*api.StepInfo{
			stepA.ID: {Step: stepA},
			stepB.ID: {Step: stepB},
		},
		Attributes: map[api.Name]*api.Dependencies{
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
		Goals: []api.StepID{"step-alone"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
		Attributes: map[api.Name]*api.Dependencies{},
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
		Steps: map[api.StepID]*api.StepInfo{
			stepA.ID: {Step: stepA},
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
	assert.Contains(t, outputArgs, api.Name("result"), "step should provide 'result' output")
	assert.NotContains(t, outputArgs, api.Name("other"), "step should not provide 'other' output")
}
