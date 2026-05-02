package engine_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestIncompleteWorkFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("retry-stop")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetError(st.ID, api.ErrWorkNotCompleted)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-not-complete")
		fl := env.WaitForFlowStatus(id, func() {
			env.WaitFor(wait.WorkFailed(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}), func() {
				err := env.Engine.StartFlow(id, pl)
				assert.NoError(t, err)
			})
		})
		assert.Equal(t, api.FlowFailed, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepFailed, ex.Status)
		assert.Len(t, ex.WorkItems, 1)
		for _, item := range ex.WorkItems {
			assert.Equal(t, api.WorkFailed, item.Status)
			assert.Equal(t, api.ErrWorkNotCompleted.Error(), item.Error)
		}
	})
}

func TestWorkFailure(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("failure-step")

		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetError(st.ID, errors.New("boom"))

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("wf-failure")
		env.WaitFor(wait.FlowFailed(id), func() {
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)

		ex := fl.Executions[st.ID]
		assert.Equal(t, api.StepFailed, ex.Status)
		assert.Len(t, ex.WorkItems, 1)
		for _, item := range ex.WorkItems {
			assert.Equal(t, api.WorkFailed, item.Status)
			assert.Contains(t, item.Error, "boom")
		}
	})
}

func TestPendingRetryCanComplete(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewStepWithOutputs("pending-complete", "output")
		id := api.FlowID("wf-pending-complete")
		tkn := api.Token("logical-work")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		assert.NoError(t, env.RaiseFlowEvents(
			id,
			helpers.FlowEvent{
				Type: api.EventTypeFlowStarted,
				Data: api.FlowStartedEvent{
					FlowID: id,
					Plan:   pl,
					Init:   api.InitArgs{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeStepStarted,
				Data: api.StepStartedEvent{
					FlowID: id,
					StepID: st.ID,
					Inputs: api.Args{},
					WorkItems: map[api.Token]api.Args{
						tkn: {},
					},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeRetryScheduled,
				Data: api.RetryScheduledEvent{
					FlowID:      id,
					StepID:      st.ID,
					Token:       tkn,
					RetryCount:  1,
					NextRetryAt: time.Now().Add(time.Minute),
					Error:       "retry",
				},
			},
		))

		err := env.Engine.CompleteWork(
			api.FlowStep{FlowID: id, StepID: st.ID},
			tkn,
			api.Args{"output": "ok"},
		)
		assert.NoError(t, err)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, api.WorkSucceeded,
			fl.Executions[st.ID].WorkItems[tkn].Status)
	})
}

func TestPendingRetryCanFail(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("pending-fail")
		id := api.FlowID("wf-pending-fail")
		tkn := api.Token("logical-work")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		assert.NoError(t, env.RaiseFlowEvents(
			id,
			helpers.FlowEvent{
				Type: api.EventTypeFlowStarted,
				Data: api.FlowStartedEvent{
					FlowID: id,
					Plan:   pl,
					Init:   api.InitArgs{},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeStepStarted,
				Data: api.StepStartedEvent{
					FlowID: id,
					StepID: st.ID,
					Inputs: api.Args{},
					WorkItems: map[api.Token]api.Args{
						tkn: {},
					},
				},
			},
			helpers.FlowEvent{
				Type: api.EventTypeRetryScheduled,
				Data: api.RetryScheduledEvent{
					FlowID:      id,
					StepID:      st.ID,
					Token:       tkn,
					RetryCount:  1,
					NextRetryAt: time.Now().Add(time.Minute),
					Error:       "retry",
				},
			},
		))

		err := env.Engine.FailWork(
			api.FlowStep{FlowID: id, StepID: st.ID},
			tkn,
			"permanent",
		)
		assert.NoError(t, err)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)
		assert.Equal(t, api.WorkFailed,
			fl.Executions[st.ID].WorkItems[tkn].Status)
	})
}

func TestWorkFailed(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewStepWithOutputs("fail-step", "output")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 10,
			MaxBackoff:  10,
			BackoffType: api.BackoffTypeFixed,
		}

		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetError("fail-step", assert.AnError)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{"fail-step"},
			Steps: api.Steps{st.ID: st},
		}

		fl := env.WaitForFlowStatus("wf-fail", func() {
			err = env.Engine.StartFlow("wf-fail", pl)
			assert.NoError(t, err)
		})

		assert.Equal(t, api.FlowFailed, fl.Status)
	})
}
