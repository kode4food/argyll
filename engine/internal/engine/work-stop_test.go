package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestIncompleteWorkFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("retry-stop")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-not-complete")
		fl := env.WaitForFlowStatus(flowID, func() {
			env.WaitFor(wait.WorkFailed(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			}), func() {
				err := env.Engine.StartFlow(flowID, plan)
				assert.NoError(t, err)
			})
		})
		assert.Equal(t, api.FlowFailed, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, api.StepFailed, exec.Status)
		assert.Len(t, exec.WorkItems, 1)
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkFailed, item.Status)
			assert.Equal(t, api.ErrWorkNotCompleted.Error(), item.Error)
		}
	})
}

func TestWorkFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		step := helpers.NewSimpleStep("failure-step")

		assert.NoError(t, env.Engine.RegisterStep(step))
		env.MockClient.SetError(step.ID, errors.New("boom"))

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{step.ID},
			Steps: api.Steps{step.ID: step},
		}

		flowID := api.FlowID("wf-failure")
		env.WaitFor(wait.FlowFailed(flowID), func() {
			err := env.Engine.StartFlow(flowID, plan)
			assert.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState(flowID)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)

		exec := fl.Executions[step.ID]
		assert.Equal(t, api.StepFailed, exec.Status)
		assert.Len(t, exec.WorkItems, 1)
		for _, item := range exec.WorkItems {
			assert.Equal(t, api.WorkFailed, item.Status)
			assert.Contains(t, item.Error, "boom")
		}
	})
}
func TestWorkFailed(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		step := helpers.NewStepWithOutputs("fail-step", "output")
		step.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 10,
			MaxBackoff:  10,
			BackoffType: api.BackoffTypeFixed,
		}

		err := env.Engine.RegisterStep(step)
		assert.NoError(t, err)

		env.MockClient.SetError("fail-step", assert.AnError)

		plan := &api.ExecutionPlan{
			Goals: []api.StepID{"fail-step"},
			Steps: api.Steps{step.ID: step},
		}

		finalState := env.WaitForFlowStatus("wf-fail", func() {
			err = env.Engine.StartFlow("wf-fail", plan)
			assert.NoError(t, err)
		})

		assert.Equal(t, api.FlowFailed, finalState.Status)
	})
}
