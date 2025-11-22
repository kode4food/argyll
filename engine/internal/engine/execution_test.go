package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/internal/engine"
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
		Goals: []api.StepID{"prep-step"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-prep", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	t.Run("successful preparation", func(t *testing.T) {
		fs := engine.FlowStep{FlowID: "wf-prep", StepID: "prep-step"}
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), fs,
		)
		assert.NotNil(t, execCtx)
	})

	t.Run("invalid flow id", func(t *testing.T) {
		fs := engine.FlowStep{FlowID: "invalid-flow-id", StepID: "prep-step"}
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), fs,
		)
		assert.Nil(t, execCtx)
	})

	t.Run("invalid step id", func(t *testing.T) {
		fs := engine.FlowStep{FlowID: "wf-prep", StepID: "invalid-step-id"}
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), fs,
		)
		assert.Nil(t, execCtx)
	})

	t.Run("step not in pending state", func(t *testing.T) {
		step2 := helpers.NewSimpleStep("active-step")
		err := env.Engine.RegisterStep(context.Background(), step2)
		require.NoError(t, err)

		plan2 := &api.ExecutionPlan{
			Goals: []api.StepID{"active-step"},
			Steps: map[api.StepID]*api.StepInfo{
				step2.ID: {Step: step2},
			},
		}

		err = env.Engine.StartFlow(
			context.Background(), "wf-prep-2", plan2, api.Args{},
			api.Metadata{},
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

		fs := engine.FlowStep{FlowID: "wf-prep-2", StepID: "active-step"}
		execCtx := env.Engine.PrepareStepExecution(
			context.Background(), fs,
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
		Goals: []api.StepID{"enqueue-step"},
		Steps: map[api.StepID]*api.StepInfo{
			step.ID: {Step: step},
		},
	}

	err = env.Engine.StartFlow(
		context.Background(), "wf-enqueue", plan, api.Args{}, api.Metadata{},
	)
	require.NoError(t, err)

	fs := engine.FlowStep{FlowID: "wf-enqueue", StepID: "enqueue-step"}
	err = env.Engine.StartStepExecution(
		context.Background(), fs, step, api.Args{},
	)
	require.NoError(t, err)

	env.Engine.EnqueueStepResult(
		fs, api.Args{"result": 42}, 100,
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
