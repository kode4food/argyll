package helpers_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestWaitForFlowCompletedEvent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("completed-step")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("flow-completed-event")
		env.WithConsumer(func(consumer *event.Consumer) {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)

			wait.On(t, consumer).ForEvent(wait.FlowCompleted(id))
		})

		flow, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowCompleted, flow.Status)
	})
}

func TestWaitForFlowFailedEvent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("failed-step")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetError(st.ID, assert.AnError)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("flow-failed-event")
		env.WithConsumer(func(consumer *event.Consumer) {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)

			wait.On(t, consumer).ForEvent(wait.FlowFailed(id))
		})

		flow, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, flow.Status)
	})
}

func TestWaitForStepStartedEvent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("started-step")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("flow-step-started")
		env.WithConsumer(func(consumer *event.Consumer) {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)

			wait.On(t, consumer).ForEvent(wait.StepStarted(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}))
		})
	})
}

func TestWaitForStepTerminalEvent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("terminal-step")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("flow-step-terminal")
		env.WithConsumer(func(consumer *event.Consumer) {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)

			wait.On(t, consumer).ForEvent(wait.StepTerminal(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}))
		})

		flow, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		exec := flow.Executions[st.ID]
		assert.NotNil(t, exec)
		assert.Equal(t, api.StepCompleted, exec.Status)
	})
}

func TestWaitForWorkSucceededEvent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("work-succeeded")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("flow-work-succeeded")
		env.WithConsumer(func(consumer *event.Consumer) {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)

			wait.On(t, consumer).ForEvent(wait.WorkSucceeded(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}))
		})
	})
}

func TestWaitForWorkFailedEvent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("work-failed")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetError(st.ID, assert.AnError)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("flow-work-failed")
		env.WithConsumer(func(consumer *event.Consumer) {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)

			wait.On(t, consumer).ForEvent(wait.WorkFailed(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}))
		})
	})
}

func TestWaitForWorkRetryScheduledEvent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("work-retry")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			InitBackoff: 10,
			MaxBackoff:  10,
			BackoffType: api.BackoffTypeFixed,
		}
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetError(st.ID, api.ErrWorkNotCompleted)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("flow-work-retry")
		env.WithConsumer(func(consumer *event.Consumer) {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)

			wait.On(t, consumer).ForEvent(wait.WorkRetryScheduled(api.FlowStep{
				FlowID: id,
				StepID: st.ID,
			}))
		})
	})
}

func TestWaitForEngineEvents(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("engine-events")
		env.WithConsumer(func(consumer *event.Consumer) {
			err := env.Engine.RegisterStep(st)
			assert.NoError(t, err)

			wait.On(t, consumer).ForEvent(wait.EngineEvent(
				api.EventTypeStepRegistered,
			))
		})
	})
}

func TestWaitFlowCompleted(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("simple-step")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-flow-completed")
		finalState := env.WaitForFlowStatus(id, func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.NotNil(t, finalState)
		assert.Equal(t, api.FlowCompleted, finalState.Status)
	})
}

func TestWaitFlowFailed(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("failing-step")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetError(st.ID, assert.AnError)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-flow-failed")
		finalState := env.WaitForFlowStatus(id, func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.NotNil(t, finalState)
		assert.Equal(t, api.FlowFailed, finalState.Status)
	})
}

func TestWaitFlowStatusTerminal(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("polling-step")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  -1,
			InitBackoff: 200,
			MaxBackoff:  200,
			BackoffType: api.BackoffTypeFixed,
		}
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetError(st.ID, api.ErrWorkNotCompleted)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-flow-polling")
		env.WaitForStepStarted(
			api.FlowStep{FlowID: id, StepID: st.ID},
			func() {
				err = env.Engine.StartFlow(id, pl)
				assert.NoError(t, err)
			},
		)

		finalState := env.WaitForFlowStatus(id, func() {
			env.MockClient.ClearError(st.ID)
			env.MockClient.SetResponse(st.ID, api.Args{})
		})
		assert.NotNil(t, finalState)
		assert.Equal(t, api.FlowCompleted, finalState.Status)
	})
}

func TestWaitStepCompleted(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("step-complete")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetResponse(st.ID, api.Args{"result": "done"})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-step-complete")
		execState := env.WaitForStepStatus(id, st.ID, func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.NotNil(t, execState)
		assert.Equal(t, api.StepCompleted, execState.Status)
	})
}

func TestWaitStepFailed(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("step-fail")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		env.MockClient.SetError(st.ID, assert.AnError)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-step-fail")
		execState := env.WaitForStepStatus(id, st.ID, func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.NotNil(t, execState)
		assert.Equal(t, api.StepFailed, execState.Status)
	})
}

func TestWaitStepSkipped(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewStepWithPredicate(
			"skip-step", api.ScriptLangAle, "false",
		)
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		id := api.FlowID("test-step-skipped")
		execState := env.WaitForStepStatus(id, st.ID, func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
		assert.NotNil(t, execState)
		assert.Equal(t, api.StepSkipped, execState.Status)
	})
}

func TestWaitForHelper(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("waitfor-step")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)
		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}
		id := api.FlowID("waitfor-flow")

		env.WaitFor(wait.FlowCompleted(id), func() {
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
		})
	})
}

func TestWaitForCountHelper(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		env.WaitForCount(
			2, wait.EngineEvent(api.EventTypeStepRegistered), func() {
				err := env.Engine.RegisterStep(helpers.NewSimpleStep("count-step-1"))
				assert.NoError(t, err)
				err = env.Engine.RegisterStep(helpers.NewSimpleStep("count-step-2"))
				assert.NoError(t, err)
			},
		)
	})
}

func TestWaitAfterAllHelper(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())
		defer func() { _ = env.Engine.Stop() }()

		st := helpers.NewSimpleStep("waitafterall-step")
		err := env.Engine.RegisterStep(st)
		assert.NoError(t, err)
		env.MockClient.SetResponse(st.ID, api.Args{})

		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}
		id := api.FlowID("waitafterall-flow")

		env.WaitAfterAll(2, func(waits []*wait.Wait) {
			assert.Len(t, waits, 2)
			err = env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
			for _, w := range waits {
				w.ForEvent(wait.FlowCompleted(id))
			}
		})
	})
}

func TestWaitForInvocation(t *testing.T) {
	cl := helpers.NewMockClient()
	stepID := api.StepID("invoked-step")
	step := &api.Step{ID: stepID}

	go func() {
		time.Sleep(10 * time.Millisecond)
		_, _ = cl.Invoke(step, api.Args{}, api.Metadata{})
	}()

	assert.True(t, cl.WaitForInvocation(stepID, time.Second))
	assert.False(t, cl.WaitForInvocation("never-invoked", 5*time.Millisecond))
}
