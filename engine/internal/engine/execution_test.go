package engine_test

import (
	"context"
	"testing"

	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestPrepareStepExecution(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewSimpleStep("prep-step")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		GoalSteps: []timebox.ID{"prep-step"},
		Steps:     []*api.Step{step},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-prep", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	t.Run("successful preparation", func(t *testing.T) {
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), "wf-prep", "prep-step",
		)
		assert.NotNil(t, execCtx, "should return execution context")
	})

	t.Run("invalid workflow id", func(t *testing.T) {
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), "invalid-flow-id", "prep-step",
		)
		assert.Nil(t, execCtx, "should return nil for invalid workflow")
	})

	t.Run("invalid step id", func(t *testing.T) {
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), "wf-prep", "invalid-step-id",
		)
		assert.Nil(t, execCtx, "should return nil for invalid step")
	})

	t.Run("step not in pending state", func(t *testing.T) {
		step2 := helpers.NewSimpleStep("active-step")
		err := env.Engine.RegisterStep(context.Background(), step2)
		require.NoError(t, err)

		plan2 := &api.ExecutionPlan{
			GoalSteps: []timebox.ID{"active-step"},
			Steps:     []*api.Step{step2},
		}

		err = env.Engine.StartWorkflow(
			context.Background(), "wf-prep-2", plan2, api.Args{}, api.Metadata{},
		)
		require.NoError(t, err)

		err = env.Engine.StartStepExecution(
			context.Background(), "wf-prep-2", "active-step", api.Args{},
		)
		require.NoError(t, err)

		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), "wf-prep-2", "active-step",
		)
		assert.Nil(t, execCtx, "should return nil for pending step")
	})
}

func TestEnqueueStepResult(t *testing.T) {
	env := helpers.NewTestEngine(t)
	defer env.Cleanup()

	env.Engine.Start()

	step := helpers.NewStepWithOutputs("enqueue-step", "result")

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		GoalSteps: []timebox.ID{"enqueue-step"},
		Steps:     []*api.Step{step},
	}

	err = env.Engine.StartWorkflow(
		context.Background(), "wf-enqueue", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	err = env.Engine.StartStepExecution(
		context.Background(), "wf-enqueue", "enqueue-step", api.Args{},
	)
	require.NoError(t, err)

	env.Engine.EnqueueStepResult(
		"wf-enqueue", "enqueue-step", api.Args{"result": 42}, 100,
	)

	workflow, err := env.Engine.GetWorkflowState(
		context.Background(), "wf-enqueue",
	)
	require.NoError(t, err)

	exec, ok := workflow.Executions["enqueue-step"]
	require.True(t, ok)
	assert.Equal(t, api.StepCompleted, exec.Status)

	if assert.Contains(t, workflow.Attributes, api.Name("result")) {
		assert.Equal(t, float64(42), workflow.Attributes["result"].Value)
	}
}
