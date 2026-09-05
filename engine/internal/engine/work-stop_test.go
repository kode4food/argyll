package engine_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/event"
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
		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			err := env.Engine.StartFlow(id, pl)
			assert.NoError(t, err)
			w.ForAll(
				wait.WorkFailed(api.FlowStep{
					FlowID: id,
					StepID: st.ID,
				}),
				wait.FlowTerminal(id),
			)
		})
		fl := env.WaitForTerminalFlow(id)
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
				Type: api.EventTypeWorkRetryScheduled,
				Data: api.WorkRetryScheduledEvent{
					FlowID:      id,
					StepID:      st.ID,
					Token:       tkn,
					RetryCount:  1,
					NextRetryAt: scheduler.Now().Add(time.Minute),
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
				Type: api.EventTypeWorkRetryScheduled,
				Data: api.WorkRetryScheduledEvent{
					FlowID:      id,
					StepID:      st.ID,
					Token:       tkn,
					RetryCount:  1,
					NextRetryAt: scheduler.Now().Add(time.Minute),
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

func TestLateResultsAfterFlowFails(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		st.ID = "late-result-step"
		st.HTTP.Invoke.Mode = api.ActionModeAsync
		st.HTTP.Invoke.Timeout = 60000
		st.WorkConfig = &api.WorkConfig{Parallelism: 2, MaxRetries: -1}
		st.Attributes["items"].Required = &api.RequiredConfig{ForEach: true}
		st.Attributes["items"].Type = api.TypeArray
		st.Attributes["result"] = &api.AttributeSpec{
			Role: api.RoleOutput,
			Type: api.TypeString,
		}
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		id := api.FlowID("wf-late-results")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitForCount(2, wait.WorkStartedAny(fs), func() {
			assert.NoError(t, env.Engine.StartFlow(id, pl,
				flow.WithInit(api.InitArgs{"items": {[]any{"a", "b"}}}),
			))
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		tokens := make([]api.Token, 0, 2)
		for tkn := range fl.Executions[st.ID].WorkItems {
			tokens = append(tokens, tkn)
		}
		assert.Len(t, tokens, 2)

		// One item fails permanently, which fails the step and the flow while
		// the other is still awaiting its callback
		assert.NoError(t, env.Engine.FailWork(fs, tokens[0], "permanent"))
		fl, err = env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)

		// The terminal flow leaves a late not-completed report at rest
		assert.NoError(t, env.Engine.NotCompleteWork(fs, tokens[1], "not yet"))
		fl, err = env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		late := fl.Executions[st.ID].WorkItems[tokens[1]]
		assert.Equal(t, api.WorkNotCompleted, late.Status)

		// A later success is still recorded, for the audit trail
		assert.NoError(t, env.Engine.CompleteWork(fs, tokens[1], api.Args{}))
		fl, err = env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		late = fl.Executions[st.ID].WorkItems[tokens[1]]
		assert.Equal(t, api.WorkSucceeded, late.Status)
		assert.Equal(t, api.FlowFailed, fl.Status)
	})
}

// TestSideEffectRunsOnceUnderConflict proves a command body that re-runs on an
// optimistic-concurrency conflict still performs its side effect once. The
// side effect here is the next item of a serial step, started in that command
func TestSideEffectRunsOnceUnderConflict(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewTestStepWithArgs([]api.Name{"items"}, nil)
		st.ID = "conflict-serial-step"
		st.HTTP.Invoke.Mode = api.ActionModeAsync
		st.HTTP.Invoke.Timeout = 60000
		st.WorkConfig = &api.WorkConfig{Parallelism: 1, MaxRetries: -1}
		st.Attributes["items"].Required = &api.RequiredConfig{ForEach: true}
		st.Attributes["items"].Type = api.TypeArray
		assert.NoError(t, env.Engine.RegisterStep(st))

		var invocations atomic.Int32
		env.MockClient.SetHandler(st.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				invocations.Add(1)
				return api.Args{}, nil
			},
		)

		id := api.FlowID("wf-conflict-once")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.WorkStartedAny(fs), func() {
			assert.NoError(t, env.Engine.StartFlow(id, pl,
				flow.WithInit(api.InitArgs{"items": {[]any{"a", "b"}}}),
			))
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		var first api.Token
		for tkn, work := range fl.Executions[st.ID].WorkItems {
			if work.Status == api.WorkActive {
				first = tkn
			}
		}
		// The invocation itself runs after the commit, so wait for it
		assert.Eventually(t, func() bool {
			return invocations.Load() == 1
		}, wait.DefaultTimeout, 10*time.Millisecond)

		// Another writer wins the race, so the command body runs twice
		env.ConflictOnNextAppend(id)
		assert.NoError(t, env.Engine.CompleteWork(fs, first, api.Args{}))
		assert.True(t, env.ConflictFired())

		assert.Eventually(t, func() bool {
			return invocations.Load() >= 2
		}, wait.DefaultTimeout, 10*time.Millisecond)
		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, int32(2), invocations.Load())
	})
}
