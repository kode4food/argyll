package engine_test

import (
	"context"
	"testing"
	"time"

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

	step := helpers.NewTestStepWithArgs([]api.Name{"required_input"}, nil)
	step.ID = "prep-step"

	err := env.Engine.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	plan := &api.ExecutionPlan{
		Goals: []timebox.ID{"prep-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-prep", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	t.Run("successful preparation", func(t *testing.T) {
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), "wf-prep", "prep-step",
		)
		assert.NotNil(t, execCtx)
	})

	t.Run("invalid flow id", func(t *testing.T) {
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), "invalid-flow-id", "prep-step",
		)
		assert.Nil(t, execCtx)
	})

	t.Run("invalid step id", func(t *testing.T) {
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), "wf-prep", "invalid-step-id",
		)
		assert.Nil(t, execCtx)
	})

	t.Run("step not in pending state", func(t *testing.T) {
		step2 := helpers.NewSimpleStep("active-step")
		err := env.Engine.RegisterStep(context.Background(), step2)
		require.NoError(t, err)

		plan2 := &api.ExecutionPlan{
			Goals: []timebox.ID{"active-step"},
			Steps: map[timebox.ID]*api.StepInfo{
				step2.ID: {Step: step2},
			},
		}

		err = env.Engine.StartFlow(
			context.Background(), "wf-prep-2", plan2, api.Args{}, api.Metadata{},
		)
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			flow, err := env.Engine.GetFlowState(
				context.Background(), "wf-prep-2",
			)
			if err != nil {
				return false
			}
			exec, ok := flow.Executions["active-step"]
			return ok && exec.Status != api.StepPending
		}, 5*time.Second, 100*time.Millisecond)

		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), "wf-prep-2", "active-step",
		)
		assert.Nil(t, execCtx)
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
		Goals: []timebox.ID{"enqueue-step"},
		Steps: map[timebox.ID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
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

	require.Eventually(t, func() bool {
		flow, err := env.Engine.GetFlowState(
			context.Background(), "wf-enqueue",
		)
		if err != nil {
			return false
		}

		exec, ok := flow.Executions["enqueue-step"]
		if !ok || exec.Status != api.StepCompleted {
			return false
		}

		_, hasAttr := flow.Attributes["result"]
		return hasAttr
	}, 5*time.Second, 100*time.Millisecond)

	flow, err := env.Engine.GetFlowState(
		context.Background(), "wf-enqueue",
	)
	require.NoError(t, err)

	exec, ok := flow.Executions["enqueue-step"]
	require.True(t, ok)
	assert.Equal(t, api.StepCompleted, exec.Status)

	if assert.Contains(t, flow.Attributes, api.Name("result")) {
		assert.Equal(t, float64(42), flow.Attributes["result"].Value)
	}
}
