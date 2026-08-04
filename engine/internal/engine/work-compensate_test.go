package engine_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

func TestMemoizableNoCompensate(t *testing.T) {
	st := &api.Step{
		ID:         "memo-comp-step",
		Name:       "Memoizable Compensating Step",
		Type:       api.StepTypeSync,
		Memoizable: true,
		HTTP: &api.HTTPConfig{
			Endpoint:   "http://test:8080/work",
			Compensate: "http://test:8080/compensate",
		},
		Attributes: api.AttributeSpecs{},
	}
	assert.Error(t, st.Validate())
}

func TestCompensationSucceeds(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := newCompensatingStep("comp-ok-step")
		assert.NoError(t, env.Engine.RegisterStep(st))

		id := api.FlowID("wf-comp-ok")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-a")

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:   env,
			id:    id,
			step:  st,
			token: tkn,
		})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.RecoverFlow(id))
			w.ForAll(
				wait.CompStarted(fs),
				wait.CompSucceeded(fs),
				wait.FlowDeactivated(id),
			)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompensated, work.Status)
		assert.False(t, work.CompletedAt.IsZero())
	})
}

func TestNoCompForFailedWork(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := newCompensatingStep("comp-fail-work-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetError(st.ID, errors.New("permanent"))

		id := api.FlowID("wf-no-comp")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		env.WaitFor(wait.FlowDeactivated(id), func() {
			assert.NoError(t, env.Engine.StartFlow(id, pl))
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)

		for _, work := range fl.Executions[st.ID].WorkItems {
			assert.NotEqual(t, api.WorkCompensating, work.Status)
			assert.NotEqual(t, api.WorkCompensated, work.Status)
		}
	})
}

func TestCompRetryOnTransient(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := newCompensatingStep("comp-retry-step")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		compCount := 0
		env.MockClient.SetCompHandler(st.ID,
			func(_ *api.Step, _, _ api.Args, _ api.Metadata) error {
				compCount++
				if compCount < 2 {
					return api.ErrWorkNotCompleted
				}
				return nil
			},
		)

		id := api.FlowID("wf-comp-retry")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-b")

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:   env,
			id:    id,
			step:  st,
			token: tkn,
		})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.RecoverFlow(id))
			w.ForAll(
				wait.CompRetryScheduled(fs),
				wait.CompSucceeded(fs),
				wait.FlowDeactivated(id),
			)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)
		assert.GreaterOrEqual(t, compCount, 2)

		work := fl.Executions[fs.StepID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompensated, work.Status)
	})
}

func TestCompRetriesExhausted(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := newCompensatingStep("comp-exhaust-step")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  1,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetCompError(st.ID, api.ErrWorkNotCompleted)

		id := api.FlowID("wf-comp-exhaust")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-c")

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:   env,
			id:    id,
			step:  st,
			token: tkn,
		})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.RecoverFlow(id))
			w.ForAll(
				wait.CompFailed(fs),
				wait.FlowDeactivated(id),
			)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t, api.FlowFailed, fl.Status)

		work := fl.Executions[fs.StepID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompFailed, work.Status)
		assert.NotEmpty(t, work.Error)
	})
}

func TestCompensationRecovery(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := newCompensatingStep("comp-recover-step")
		assert.NoError(t, env.Engine.RegisterStep(st))

		id := api.FlowID("wf-comp-recover")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-d")

		// State: failed flow with WorkCompensating already started
		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:     env,
			id:      id,
			step:    st,
			token:   tkn,
			started: true,
		})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.RecoverFlow(id))
			w.ForAll(
				wait.CompSucceeded(fs),
				wait.FlowDeactivated(id),
			)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompensated, work.Status)
	})
}

func TestCompCompleteDirectly(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := newCompensatingStep("comp-direct-step")
		id := api.FlowID("wf-comp-direct")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-e")

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:     env,
			id:      id,
			step:    st,
			token:   tkn,
			started: true,
		})

		err := env.Engine.CompleteCompensation(fs, tkn)
		assert.NoError(t, err)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompensated, work.Status)
		assert.False(t, work.CompletedAt.IsZero())
	})
}

func TestCompFailDirectly(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := newCompensatingStep("comp-direct-fail-step")
		id := api.FlowID("wf-comp-direct-fail")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-f")

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:     env,
			id:      id,
			step:    st,
			token:   tkn,
			started: true,
		})

		err := env.Engine.FailCompensation(fs, tkn, "comp boom")
		assert.NoError(t, err)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompFailed, work.Status)
		assert.Equal(t, "comp boom", work.Error)
	})
}

func TestCompDeferredToHealthyPeer(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cfg := util.MutableCopy(env.Config)
		cfg.Raft.LocalID = "node-comp-peer"
		cfg.Raft.Servers = append(cfg.Raft.Servers,
			raft.Server{ID: "node-comp-peer", Address: "127.0.0.1:9710"},
		)

		peer, unsub, err := env.NewEngineWithConfig(cfg, env.Dependencies())
		assert.NoError(t, err)
		if !assert.NotNil(t, peer) {
			return
		}
		defer func() {
			unsub()
			assert.NoError(t, peer.Stop())
		}()

		st := newCompensatingStep("comp-deferred-step")
		env.MockClient.SetCompHandler(st.ID,
			func(_ *api.Step, _, _ api.Args, _ api.Metadata) error {
				return nil
			},
		)

		// Primary node is unhealthy for this step; peer is healthy
		assert.NoError(t, env.Engine.UpdateStepHealth(
			st.ID, api.HealthUnhealthy, "offline",
		))
		assert.NoError(t, peer.UpdateStepHealth(st.ID, api.HealthHealthy, ""))

		assert.NoError(t, env.Engine.Start())
		assert.NoError(t, peer.Start())

		id := api.FlowID("wf-comp-deferred")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-deferred")

		// Inject flow state: step failed with one succeeded work item and
		// comp already started (WorkCompensating)
		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:     env,
			id:      id,
			step:    st,
			token:   tkn,
			started: true,
		})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.RecoverFlow(id))
			w.ForAll(
				wait.DispatchDeferred(fs),
				wait.CompSucceeded(fs),
				wait.FlowDeactivated(id),
			)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompensated, work.Status)
	})
}

func TestCompFailOnPermanentError(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := newCompensatingStep("comp-hard-fail-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetCompError(st.ID, errors.New("hard failure"))

		id := api.FlowID("wf-comp-hard-fail")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-hard")

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:   env,
			id:    id,
			step:  st,
			token: tkn,
		})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.RecoverFlow(id))
			w.ForAll(
				wait.CompFailed(fs),
				wait.FlowDeactivated(id),
			)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompFailed, work.Status)
		assert.NotEmpty(t, work.Error)
	})
}

func TestCompCompleteIdempotent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := newCompensatingStep("comp-idem-ok-step")
		id := api.FlowID("wf-comp-idem-ok")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-idem-ok")

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:     env,
			id:      id,
			step:    st,
			token:   tkn,
			started: true,
		})

		assert.NoError(t, env.Engine.CompleteCompensation(fs, tkn))

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t,
			api.WorkCompensated, fl.Executions[st.ID].WorkItems[tkn].Status,
		)

		// Second call is a no-op — work is no longer comp-active
		assert.NoError(t, env.Engine.CompleteCompensation(fs, tkn))

		fl, err = env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t,
			api.WorkCompensated, fl.Executions[st.ID].WorkItems[tkn].Status,
		)
	})
}

func TestCompFailIdempotent(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := newCompensatingStep("comp-idem-fail-step")
		id := api.FlowID("wf-comp-idem-fail")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-idem-fail")

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:     env,
			id:      id,
			step:    st,
			token:   tkn,
			started: true,
		})

		assert.NoError(t, env.Engine.FailCompensation(fs, tkn, "boom"))

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t,
			api.WorkCompFailed, fl.Executions[st.ID].WorkItems[tkn].Status,
		)

		// Second call is a no-op — work is no longer comp-active
		assert.NoError(t, env.Engine.FailCompensation(fs, tkn, "boom again"))

		fl, err = env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t,
			api.WorkCompFailed, fl.Executions[st.ID].WorkItems[tkn].Status,
		)
	})
}

func TestCompRetryNoopForMissingOrTerminalWork(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := newCompensatingStep("comp-retry-noop-step")
		id := api.FlowID("wf-comp-retry-noop")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-retry-noop")

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:     env,
			id:      id,
			step:    st,
			token:   tkn,
			started: true,
		})
		assert.NoError(t, env.Engine.CompleteCompensation(fs, tkn))

		assert.NoError(t, env.Engine.NotCompleteCompensation(
			fs, api.Token("missing"), "missing",
		))
		assert.NoError(t, env.Engine.NotCompleteCompensation(
			fs, tkn, "already terminal",
		))

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompensated, work.Status)
		assert.Equal(t, 0, work.RetryCount)
	})
}

func TestCompDispatchRecovery(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := newCompensatingStep("dispatch-recovery-step")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  2,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		compCount := 0
		env.MockClient.SetCompHandler(st.ID,
			func(_ *api.Step, _, _ api.Args, _ api.Metadata) error {
				compCount++
				return nil
			},
		)

		id := api.FlowID("wf-dispatch-recovery")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-recovery")

		// State: failed flow with comp in progress (WorkCompensating)
		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:     env,
			id:      id,
			step:    st,
			token:   tkn,
			started: true,
		})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.RecoverFlow(id))
			w.ForAll(
				wait.CompSucceeded(fs),
				wait.FlowDeactivated(id),
			)
		})

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		work := fl.Executions[st.ID].WorkItems[tkn]
		assert.Equal(t, api.WorkCompensated, work.Status)
		assert.Equal(t, 1, compCount)
	})
}

func TestFlowCompensation(t *testing.T) {
	newFlowCompStep := func(id api.StepID) (*api.Step, *api.Step) {
		producer := newCompensatingStep(id + "-producer")
		producer.Attributes = api.AttributeSpecs{
			"value": {Role: api.RoleOutput, Type: api.TypeString},
		}
		consumer := helpers.NewSimpleStep(id + "-consumer")
		consumer.Attributes = api.AttributeSpecs{
			"value": {Role: api.RoleRequired, Type: api.TypeString},
		}
		return producer, consumer
	}

	newFlowCompPlan := func(
		producer, consumer *api.Step,
	) *api.ExecutionPlan {
		return &api.ExecutionPlan{
			Goals: []api.StepID{consumer.ID},
			Steps: api.Steps{
				producer.ID: producer,
				consumer.ID: consumer,
			},
			Attributes: api.AttributeGraph{
				"value": {
					Providers: []api.StepID{producer.ID},
					Consumers: []api.StepID{consumer.ID},
				},
			},
		}
	}

	t.Run("compensates succeeded steps", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())

			producer, consumer := newFlowCompStep("flow-comp")
			assert.NoError(t, env.Engine.RegisterStep(producer))
			assert.NoError(t, env.Engine.RegisterStep(consumer))

			env.MockClient.SetResponse(producer.ID, api.Args{"value": "abc"})
			env.MockClient.SetError(consumer.ID, errors.New("permanent"))

			id := api.FlowID("wf-flow-comp")
			pl := newFlowCompPlan(producer, consumer)

			env.WaitFor(wait.FlowDeactivated(id), func() {
				assert.NoError(t, env.Engine.StartFlow(id, pl,
					flow.WithCompensate(true),
				))
			})

			fl, err := env.Engine.GetFlowState(id)
			assert.NoError(t, err)
			assert.Equal(t, api.FlowFailed, fl.Status)
			for _, work := range fl.Executions[producer.ID].WorkItems {
				assert.Equal(t, api.WorkCompensated, work.Status)
			}
		})
	})

	t.Run("leaves succeeded steps alone when unset", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())

			producer, consumer := newFlowCompStep("no-flow-comp")
			assert.NoError(t, env.Engine.RegisterStep(producer))
			assert.NoError(t, env.Engine.RegisterStep(consumer))

			env.MockClient.SetResponse(producer.ID, api.Args{"value": "abc"})
			env.MockClient.SetError(consumer.ID, errors.New("permanent"))

			id := api.FlowID("wf-no-flow-comp")
			pl := newFlowCompPlan(producer, consumer)

			env.WaitFor(wait.FlowDeactivated(id), func() {
				assert.NoError(t, env.Engine.StartFlow(id, pl))
			})

			fl, err := env.Engine.GetFlowState(id)
			assert.NoError(t, err)
			assert.Equal(t, api.FlowFailed, fl.Status)
			for _, work := range fl.Executions[producer.ID].WorkItems {
				assert.Equal(t, api.WorkSucceeded, work.Status)
			}
		})
	})
}

// newCompensatingStep returns a sync step with a compensate endpoint
func newCompensatingStep(id api.StepID) *api.Step {
	return &api.Step{
		ID:   id,
		Name: "Compensating Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint:   "http://test:8080/work",
			Compensate: "http://test:8080/compensate",
		},
		Attributes: api.AttributeSpecs{},
	}
}

// setupCompensatingFlow injects events for a flow whose step has one succeeded
// work item followed by step and flow failure, with comp started
type setupCompensatingFlowArgs struct {
	env     *helpers.TestEngineEnv
	id      api.FlowID
	step    *api.Step
	token   api.Token
	started bool
}

func setupCompensatingFlow(args setupCompensatingFlowArgs) {
	pl := &api.ExecutionPlan{
		Goals: []api.StepID{args.step.ID},
		Steps: api.Steps{args.step.ID: args.step},
	}

	evs := []helpers.FlowEvent{
		{
			Type: api.EventTypeFlowStarted,
			Data: api.FlowStartedEvent{
				FlowID: args.id,
				Plan:   pl,
				Init:   api.InitArgs{},
			},
		},
		{
			Type: api.EventTypeStepStarted,
			Data: api.StepStartedEvent{
				FlowID:    args.id,
				StepID:    args.step.ID,
				Inputs:    api.Args{},
				WorkItems: map[api.Token]api.Args{args.token: {}},
			},
		},
		{
			Type: api.EventTypeWorkStarted,
			Data: api.WorkStartedEvent{
				FlowID: args.id,
				StepID: args.step.ID,
				Token:  args.token,
				Inputs: api.Args{},
			},
		},
		{
			Type: api.EventTypeWorkSucceeded,
			Data: api.WorkSucceededEvent{
				FlowID:  args.id,
				StepID:  args.step.ID,
				Token:   args.token,
				Outputs: api.Args{"result": "ok"},
			},
		},
		{
			Type: api.EventTypeStepFailed,
			Data: api.StepFailedEvent{
				FlowID: args.id,
				StepID: args.step.ID,
				Error:  "forced failure",
			},
		},
		{
			Type: api.EventTypeFlowFailed,
			Data: api.FlowFailedEvent{
				FlowID: args.id,
				Error:  "forced failure",
			},
		},
	}

	if args.started {
		evs = append(evs, helpers.FlowEvent{
			Type: api.EventTypeCompStarted,
			Data: api.CompStartedEvent{
				FlowID: args.id,
				StepID: args.step.ID,
				Token:  args.token,
			},
		})
	}

	assert.NoError(
		args.env.T, args.env.RaiseFlowEvents(args.id, evs...),
	)
}
