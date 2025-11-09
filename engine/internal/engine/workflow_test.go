package engine_test

import (
	"context"
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	as "github.com/kode4food/spuds/engine/internal/assert"
	"github.com/kode4food/spuds/engine/internal/assert/helpers"
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
		Goals: []timebox.ID{"step-1"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-dup", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-dup", plan, api.Args{}, api.Metadata{},
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow exists")
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
		Goals: []timebox.ID{"step-needs-input"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
		Required: []api.Name{"required_value"},
	}

	err := env.Engine.StartWorkflow(
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

	_, err := env.Engine.GetWorkflowState(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")
}

func TestSetAttribute(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("simple-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"simple-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-attr", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.SetAttribute(
		context.Background(), "wf-attr", "step-attr", "test_key", "test_value",
	)
	require.NoError(t, err)

	a := as.New(t)
	a.WorkflowStateEquals(
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
		Goals: []timebox.ID{"step-attrs"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-getattrs", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.SetAttribute(
		context.Background(), "wf-getattrs", "step-attrs", "key1", "value1",
	)
	require.NoError(t, err)

	err = env.Engine.SetAttribute(
		context.Background(), "wf-getattrs", "step-attrs", "key2", 42,
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
		Goals: []timebox.ID{"step-dup-attr"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-dup-attr", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.SetAttribute(
		context.Background(), "wf-dup-attr", "step-dup-attr", "dup_key",
		"first",
	)
	require.NoError(t, err)

	err = env.Engine.SetAttribute(
		context.Background(), "wf-dup-attr", "step-dup-attr", "dup_key",
		"second",
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attribute already set")
}

func TestCompleteWorkflow(t *testing.T) {
	a := as.New(t)
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("complete-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"complete-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-complete", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	result := api.Args{"final": "result"}
	err = env.Engine.CompleteWorkflow(
		context.Background(), "wf-complete", result,
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-complete",
	)
	require.NoError(t, err)
	a.WorkflowStatus(workflow, api.WorkflowCompleted)
}

func TestFailWorkflow(t *testing.T) {
	a := as.New(t)
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("fail-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"fail-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-fail", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.FailWorkflow(context.Background(), "wf-fail", "test error")
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-fail",
	)
	require.NoError(t, err)
	a.WorkflowStatus(workflow, api.WorkflowFailed)
	a.Equal("test error", workflow.Error)
}

func TestStartStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("exec-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"exec-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-exec", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	inputs := api.Args{"input": "value"}
	err = env.Engine.StartStepExecution(
		context.Background(), "wf-exec", "exec-step", inputs,
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-exec",
	)
	require.NoError(t, err)

	exec, ok := workflow.Executions["exec-step"]
	require.True(t, ok)
	assert.Equal(t, api.StepActive, exec.Status)
}

func TestCompleteStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("step-complete-exec")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-complete-exec"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-complete-exec", plan, api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.StartStepExecution(
		context.Background(), "wf-complete-exec", "step-complete-exec",
		api.Args{},
	)
	require.NoError(t, err)

	outputs := api.Args{"output": "result"}
	err = env.Engine.CompleteStepExecution(
		context.Background(), "wf-complete-exec", "step-complete-exec",
		outputs, 100,
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-complete-exec",
	)
	require.NoError(t, err)

	exec, ok := workflow.Executions["step-complete-exec"]
	require.True(t, ok)
	assert.Equal(t, api.StepCompleted, exec.Status)
	assert.Equal(t, "result", exec.Outputs["output"])
}

func TestFailStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("step-fail-exec")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-fail-exec"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-fail-exec", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.StartStepExecution(
		context.Background(), "wf-fail-exec", "step-fail-exec", api.Args{},
	)
	require.NoError(t, err)

	err = env.Engine.FailStepExecution(
		context.Background(), "wf-fail-exec", "step-fail-exec",
		"execution failed",
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-fail-exec",
	)
	require.NoError(t, err)

	exec, ok := workflow.Executions["step-fail-exec"]
	require.True(t, ok)
	assert.Equal(t, api.StepFailed, exec.Status)
	assert.Equal(t, "execution failed", exec.Error)
}

func TestSkipStep(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("step-skip")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-skip"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-skip", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.SkipStepExecution(
		context.Background(), "wf-skip", "step-skip", "test skip reason",
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-skip",
	)
	require.NoError(t, err)

	exec, ok := workflow.Executions["step-skip"]
	require.True(t, ok)
	assert.Equal(t, api.StepSkipped, exec.Status)
	assert.Equal(t, "test skip reason", exec.Error)
}

func TestGetWorkflowEvents(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("simple")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"simple"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-events", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	events, err := env.Engine.GetWorkflowEvents(
		context.Background(), "wf-events", 0,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
}

func TestListWorkflows(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("test")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"test"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-list", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	workflows, err := env.Engine.ListWorkflows(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, workflows)
}

func TestIsWorkflowFailed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	stepA := helpers.NewStepWithOutputs("step-a", "value")

	stepB := helpers.NewSimpleStep("step-b")
	stepB.Attributes["value"] = &api.AttributeSpec{
		Role: api.RoleRequired,
		Type: api.TypeString,
	}

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-b"},
		Steps: map[timebox.ID]*api.StepInfo{
			stepA.ID: {Step: stepA},
			stepB.ID: {Step: stepB},
		},
	}

	err := env.Engine.RegisterStep(context.Background(), stepA)
	require.NoError(t, err)
	err = env.Engine.RegisterStep(context.Background(), stepB)
	require.NoError(t, err)

	err = env.Engine.StartWorkflow(
		context.Background(),
		"wf-failed-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	_, err = env.Engine.GetWorkflowState(
		context.Background(), "wf-failed-test",
	)
	require.NoError(t, err)

	err = env.Engine.StartStepExecution(
		context.Background(), "wf-failed-test", "step-a", api.Args{},
	)
	require.NoError(t, err)

	err = env.Engine.FailStepExecution(
		context.Background(), "wf-failed-test", "step-a", "test error",
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-failed-test",
	)
	require.NoError(t, err)

	isFailed := env.Engine.IsWorkflowFailed(workflow)
	assert.True(t, isFailed)
}

func TestIsWorkflowNotFailed(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	step := helpers.NewSimpleStep("step-ok")

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-ok"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	err = env.Engine.StartWorkflow(
		context.Background(),
		"wf-ok-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-ok-test",
	)
	require.NoError(t, err)

	isFailed := env.Engine.IsWorkflowFailed(workflow)
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
		Goals: []timebox.ID{"step-b"},
		Steps: map[timebox.ID]*api.StepInfo{
			stepA.ID: {Step: stepA},
			stepB.ID: {Step: stepB},
		},
		Attributes: map[api.Name]*api.Dependencies{
			"value": {
				Providers: []timebox.ID{stepA.ID},
				Consumers: []timebox.ID{stepB.ID},
			},
		},
	}

	err := env.Engine.RegisterStep(context.Background(), stepA)
	require.NoError(t, err)
	err = env.Engine.RegisterStep(context.Background(), stepB)
	require.NoError(t, err)

	err = env.Engine.StartWorkflow(
		context.Background(),
		"wf-provider-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-provider-test",
	)
	require.NoError(t, err)

	hasProvider := env.Engine.HasInputProvider("value", workflow)
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
		Goals: []timebox.ID{"step-alone"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
		Attributes: map[api.Name]*api.Dependencies{},
	}

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	err = env.Engine.StartWorkflow(
		context.Background(),
		"wf-no-provider-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-no-provider-test",
	)
	require.NoError(t, err)

	hasProvider := env.Engine.HasInputProvider("missing", workflow)
	assert.False(t, hasProvider)
}

func TestStepProvidesInput(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	stepA := helpers.NewStepWithOutputs("step-provider", "result")

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"step-provider"},
		Steps: map[timebox.ID]*api.StepInfo{
			stepA.ID: {Step: stepA},
		},
	}

	err := env.Engine.RegisterStep(context.Background(), stepA)
	require.NoError(t, err)

	err = env.Engine.StartWorkflow(
		context.Background(),
		"wf-provides-test",
		plan,
		api.Args{},
		api.Metadata{},
	)
	require.NoError(t, err)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-provides-test",
	)
	require.NoError(t, err)

	provides := env.Engine.StepProvidesInput(stepA, "result", workflow)
	assert.True(t, provides)

	providesOther := env.Engine.StepProvidesInput(stepA, "other", workflow)
	assert.False(t, providesOther)
}
