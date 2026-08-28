package engine_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type flowCompSteps struct {
	producer *api.Step
	consumer *api.Step
}

func TestCompensateHandling(t *testing.T) {
	st := &api.Step{
		ID:       "memo-comp-step",
		Name:     "Memoized Compensating Step",
		Type:     api.StepTypeService,
		Handling: api.HandlingMemoized,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://test:8080/work"},
			Compensate: &api.HTTPAction{
				Endpoint: "http://test:8080/compensate",
			},
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
		invoked := make(chan api.Metadata, 1)
		env.MockClient.SetCompHandler(st.ID,
			func(req client.CompensateRequest) error {
				invoked <- req.Metadata
				return nil
			})

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

		meta := <-invoked
		assert.Equal(t, id, meta[api.MetaFlowID])
		assert.Equal(t, st.ID, meta[api.MetaStepID])
		assert.Equal(t, tkn, meta[api.MetaReceiptToken])
		assert.NotContains(t, meta, api.MetaWebhookURL)
	})
}

func TestCompensationAsyncAwaitsCallback(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := newCompensatingStep("comp-async-step")
		st.HTTP.Compensate.Mode = api.ActionModeAsync
		assert.NoError(t, env.Engine.RegisterStep(st))

		id := api.FlowID("wf-comp-async")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-a")

		invoked := make(chan api.Metadata, 1)
		env.MockClient.SetCompHandler(st.ID,
			func(req client.CompensateRequest) error {
				invoked <- req.Metadata
				return nil
			})

		setupCompensatingFlow(setupCompensatingFlowArgs{
			env:   env,
			id:    id,
			step:  st,
			token: tkn,
		})

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.RecoverFlow(id))
			w.ForAll(wait.CompStarted(fs))
		})

		var meta api.Metadata
		select {
		case meta = <-invoked:
		case <-time.After(time.Second):
			t.Fatal("compensation handler was not invoked")
		}
		assert.Contains(t, meta, api.MetaWebhookURL)
		assert.Equal(t, id, meta[api.MetaFlowID])
		assert.Equal(t, st.ID, meta[api.MetaStepID])
		assert.Equal(t, tkn, meta[api.MetaReceiptToken])
		assert.Equal(t,
			"http://localhost:8080/callbacks/wf-comp-async/"+
				"comp-async-step/work-a/compensate",
			meta[api.MetaWebhookURL],
		)

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t,
			api.WorkCompensating, fl.Executions[st.ID].WorkItems[tkn].Status)

		env.WithConsumer(func(consumer *event.Consumer) {
			w := wait.On(t, consumer)
			assert.NoError(t, env.Engine.CompleteCompensation(fs, tkn))
			w.ForAll(wait.CompSucceeded(fs))
		})

		fl, err = env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t,
			api.WorkCompensated, fl.Executions[st.ID].WorkItems[tkn].Status)
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
			func(client.CompensateRequest) error {
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

func TestCompDeferred(t *testing.T) {
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
			func(client.CompensateRequest) error {
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

func TestCompPermanentFailure(t *testing.T) {
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

func TestCompRetryNoop(t *testing.T) {
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
			fs, "missing", "missing",
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
			func(client.CompensateRequest) error {
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
	newFlowCompStep := func(id api.StepID) flowCompSteps {
		producer := newCompensatingStep(id + "-producer")
		producer.Attributes = api.AttributeSpecs{
			"value": {Role: api.RoleOutput, Type: api.TypeString},
		}
		consumer := helpers.NewSimpleStep(id + "-consumer")
		consumer.Attributes = api.AttributeSpecs{
			"value": {Role: api.RoleRequired, Type: api.TypeString},
		}
		return flowCompSteps{producer: producer, consumer: consumer}
	}

	newFlowCompPlan := func(steps flowCompSteps) *api.ExecutionPlan {
		return &api.ExecutionPlan{
			Goals: []api.StepID{steps.consumer.ID},
			Steps: api.Steps{
				steps.producer.ID: steps.producer,
				steps.consumer.ID: steps.consumer,
			},
			Attributes: api.AttributeGraph{
				"value": {
					Providers: []api.StepID{steps.producer.ID},
					Consumers: []api.StepID{steps.consumer.ID},
				},
			},
		}
	}

	t.Run("compensates succeeded steps", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())

			steps := newFlowCompStep("flow-comp")
			assert.NoError(t, env.Engine.RegisterStep(steps.producer))
			assert.NoError(t, env.Engine.RegisterStep(steps.consumer))

			env.MockClient.SetResponse(steps.producer.ID,
				api.Args{"value": "abc"})
			env.MockClient.SetError(steps.consumer.ID, errors.New("permanent"))

			id := api.FlowID("wf-flow-comp")
			pl := newFlowCompPlan(steps)

			env.WaitFor(wait.FlowDeactivated(id), func() {
				assert.NoError(t, env.Engine.StartFlow(id, pl,
					flow.WithCompensate(true),
				))
			})

			fl, err := env.Engine.GetFlowState(id)
			assert.NoError(t, err)
			assert.Equal(t, api.FlowFailed, fl.Status)
			for _, work := range fl.Executions[steps.producer.ID].WorkItems {
				assert.Equal(t, api.WorkCompensated, work.Status)
			}
		})
	})

	t.Run("unwinds steps in reverse dependency order", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())

			first := newCompensatingStep("saga-first")
			first.Attributes = api.AttributeSpecs{
				"one": {Role: api.RoleOutput, Type: api.TypeString},
			}
			second := newCompensatingStep("saga-second")
			second.Attributes = api.AttributeSpecs{
				"one": {Role: api.RoleRequired, Type: api.TypeString},
				"two": {Role: api.RoleOutput, Type: api.TypeString},
			}
			last := helpers.NewSimpleStep("saga-last")
			last.Attributes = api.AttributeSpecs{
				"two": {Role: api.RoleRequired, Type: api.TypeString},
			}

			for _, st := range []*api.Step{first, second, last} {
				assert.NoError(t, env.Engine.RegisterStep(st))
			}
			env.MockClient.SetResponse(first.ID, api.Args{"one": "a"})
			env.MockClient.SetResponse(second.ID, api.Args{"two": "b"})
			env.MockClient.SetError(last.ID, errors.New("permanent"))

			var mu sync.Mutex
			var order []api.StepID
			record := func(req client.CompensateRequest) error {
				mu.Lock()
				defer mu.Unlock()
				order = append(order, req.Step.ID)
				return nil
			}
			env.MockClient.SetCompHandler(first.ID, record)
			env.MockClient.SetCompHandler(second.ID, record)

			id := api.FlowID("wf-saga-order")
			pl := &api.ExecutionPlan{
				Goals: []api.StepID{last.ID},
				Steps: api.Steps{
					first.ID:  first,
					second.ID: second,
					last.ID:   last,
				},
				Attributes: api.AttributeGraph{
					"one": {
						Providers: []api.StepID{first.ID},
						Consumers: []api.StepID{second.ID},
					},
					"two": {
						Providers: []api.StepID{second.ID},
						Consumers: []api.StepID{last.ID},
					},
				},
			}

			env.WaitFor(wait.FlowDeactivated(id), func() {
				assert.NoError(t, env.Engine.StartFlow(id, pl,
					flow.WithCompensate(true),
				))
			})

			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []api.StepID{second.ID, first.ID}, order)
		})
	})

	t.Run("unwinds independent steps together", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())

			left := newCompensatingStep("saga-left")
			left.Attributes = api.AttributeSpecs{
				"one": {Role: api.RoleOutput, Type: api.TypeString},
			}
			right := newCompensatingStep("saga-right")
			right.Attributes = api.AttributeSpecs{
				"two": {Role: api.RoleOutput, Type: api.TypeString},
			}
			last := helpers.NewSimpleStep("saga-join")
			last.Attributes = api.AttributeSpecs{
				"one": {Role: api.RoleRequired, Type: api.TypeString},
				"two": {Role: api.RoleRequired, Type: api.TypeString},
			}

			for _, st := range []*api.Step{left, right, last} {
				assert.NoError(t, env.Engine.RegisterStep(st))
			}
			env.MockClient.SetResponse(left.ID, api.Args{"one": "a"})
			env.MockClient.SetResponse(right.ID, api.Args{"two": "b"})
			env.MockClient.SetError(last.ID, errors.New("permanent"))

			// Each blocks until the other arrives; serialized never settles
			var wg sync.WaitGroup
			wg.Add(2)
			together := func(client.CompensateRequest) error {
				wg.Done()
				wg.Wait()
				return nil
			}
			env.MockClient.SetCompHandler(left.ID, together)
			env.MockClient.SetCompHandler(right.ID, together)

			id := api.FlowID("wf-saga-parallel")
			pl := &api.ExecutionPlan{
				Goals: []api.StepID{last.ID},
				Steps: api.Steps{
					left.ID:  left,
					right.ID: right,
					last.ID:  last,
				},
				Attributes: api.AttributeGraph{
					"one": {
						Providers: []api.StepID{left.ID},
						Consumers: []api.StepID{last.ID},
					},
					"two": {
						Providers: []api.StepID{right.ID},
						Consumers: []api.StepID{last.ID},
					},
				},
			}

			env.WaitFor(wait.FlowDeactivated(id), func() {
				assert.NoError(t, env.Engine.StartFlow(id, pl,
					flow.WithCompensate(true),
				))
			})

			fl, err := env.Engine.GetFlowState(id)
			assert.NoError(t, err)
			for _, sid := range []api.StepID{left.ID, right.ID} {
				for _, work := range fl.Executions[sid].WorkItems {
					assert.Equal(t, api.WorkCompensated, work.Status)
				}
			}
		})
	})

	t.Run("resumes a partly unwound flow after restart", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())

			first := newCompensatingStep("saga-resume-first")
			first.Attributes = api.AttributeSpecs{
				"one": {Role: api.RoleOutput, Type: api.TypeString},
			}
			second := newCompensatingStep("saga-resume-second")
			second.Attributes = api.AttributeSpecs{
				"one": {Role: api.RoleRequired, Type: api.TypeString},
			}
			assert.NoError(t, env.Engine.RegisterStep(first))
			assert.NoError(t, env.Engine.RegisterStep(second))

			id := api.FlowID("wf-saga-resume")
			fs := api.FlowStep{FlowID: id, StepID: first.ID}
			tkn := api.Token("work-first")

			// State as left by a crash mid-unwind
			setupPartlyUnwoundFlow(setupPartlyUnwoundFlowArgs{
				env:    env,
				id:     id,
				first:  first,
				second: second,
				token:  tkn,
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
			assert.Equal(t,
				api.WorkCompensated,
				fl.Executions[first.ID].WorkItems[tkn].Status,
			)
		})
	})

	t.Run("skips steps with no compensate endpoint", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())

			producer := helpers.NewStepWithOutputs("plain-producer", "value")
			consumer := helpers.NewSimpleStep("plain-consumer")
			consumer.Attributes = api.AttributeSpecs{
				"value": {Role: api.RoleRequired, Type: api.TypeString},
			}
			assert.NoError(t, env.Engine.RegisterStep(producer))
			assert.NoError(t, env.Engine.RegisterStep(consumer))

			env.MockClient.SetResponse(producer.ID, api.Args{"value": "abc"})
			env.MockClient.SetError(consumer.ID, errors.New("permanent"))

			id := api.FlowID("wf-no-comp-endpoint")
			pl := &api.ExecutionPlan{
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

			env.WaitFor(wait.FlowDeactivated(id), func() {
				assert.NoError(t, env.Engine.StartFlow(id, pl,
					flow.WithCompensate(true),
				))
			})

			fl, err := env.Engine.GetFlowState(id)
			assert.NoError(t, err)
			for _, work := range fl.Executions[producer.ID].WorkItems {
				assert.Equal(t, api.WorkSucceeded, work.Status)
			}
		})
	})

	t.Run("leaves succeeded steps alone when unset", func(t *testing.T) {
		helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
			assert.NoError(t, env.Engine.Start())

			steps := newFlowCompStep("no-flow-comp")
			assert.NoError(t, env.Engine.RegisterStep(steps.producer))
			assert.NoError(t, env.Engine.RegisterStep(steps.consumer))

			env.MockClient.SetResponse(steps.producer.ID,
				api.Args{"value": "abc"})
			env.MockClient.SetError(steps.consumer.ID, errors.New("permanent"))

			id := api.FlowID("wf-no-flow-comp")
			pl := newFlowCompPlan(steps)

			env.WaitFor(wait.FlowDeactivated(id), func() {
				assert.NoError(t, env.Engine.StartFlow(id, pl))
			})

			fl, err := env.Engine.GetFlowState(id)
			assert.NoError(t, err)
			assert.Equal(t, api.FlowFailed, fl.Status)
			for _, work := range fl.Executions[steps.producer.ID].WorkItems {
				assert.Equal(t, api.WorkSucceeded, work.Status)
			}
		})
	})
}

// setupPartlyUnwoundFlow compensates the downstream step, not the upstream
type setupPartlyUnwoundFlowArgs struct {
	env    *helpers.TestEngineEnv
	id     api.FlowID
	first  *api.Step
	second *api.Step
	token  api.Token
}

func setupPartlyUnwoundFlow(args setupPartlyUnwoundFlowArgs) {
	first, second, id := args.first, args.second, args.id
	downstream := api.Token("work-second")

	pl := &api.ExecutionPlan{
		Goals: []api.StepID{second.ID},
		Steps: api.Steps{first.ID: first, second.ID: second},
		Attributes: api.AttributeGraph{
			"one": {
				Providers: []api.StepID{first.ID},
				Consumers: []api.StepID{second.ID},
			},
		},
	}

	evs := []helpers.FlowEvent{
		{
			Type: api.EventTypeFlowStarted,
			Data: api.FlowStartedEvent{
				FlowID:     id,
				Plan:       pl,
				Init:       api.InitArgs{},
				Compensate: true,
			},
		},
	}
	evs = append(evs, succeededWorkEvents(id, first.ID, args.token)...)
	evs = append(evs, completedStepEvent(id, first.ID))
	evs = append(evs, succeededWorkEvents(id, second.ID, downstream)...)
	evs = append(evs, completedStepEvent(id, second.ID))
	evs = append(evs,
		helpers.FlowEvent{
			Type: api.EventTypeFlowFailed,
			Data: api.FlowFailedEvent{FlowID: id, Error: "forced failure"},
		},
		helpers.FlowEvent{
			Type: api.EventTypeCompStarted,
			Data: api.CompStartedEvent{
				FlowID: id, StepID: second.ID, Token: downstream,
			},
		},
		helpers.FlowEvent{
			Type: api.EventTypeCompSucceeded,
			Data: api.CompSucceededEvent{
				FlowID: id, StepID: second.ID, Token: downstream,
			},
		},
	)

	assert.NoError(args.env.T, args.env.RaiseFlowEvents(id, evs...))
}

func completedStepEvent(id api.FlowID, sid api.StepID) helpers.FlowEvent {
	return helpers.FlowEvent{
		Type: api.EventTypeStepCompleted,
		Data: api.StepCompletedEvent{
			FlowID: id, StepID: sid, Outputs: api.Args{},
		},
	}
}

// succeededWorkEvents leaves the step active for the caller to finish
func succeededWorkEvents(
	id api.FlowID, sid api.StepID, tkn api.Token,
) []helpers.FlowEvent {
	return []helpers.FlowEvent{
		{
			Type: api.EventTypeStepStarted,
			Data: api.StepStartedEvent{
				FlowID:    id,
				StepID:    sid,
				Inputs:    api.Args{},
				WorkItems: map[api.Token]api.Args{tkn: {}},
			},
		},
		{
			Type: api.EventTypeWorkStarted,
			Data: api.WorkStartedEvent{
				FlowID: id, StepID: sid, Token: tkn, Inputs: api.Args{},
			},
		},
		{
			Type: api.EventTypeWorkSucceeded,
			Data: api.WorkSucceededEvent{
				FlowID:  id,
				StepID:  sid,
				Token:   tkn,
				Outputs: api.Args{"result": "ok"},
			},
		},
	}
}

// newCompensatingStep returns a sync step with a compensate endpoint
func newCompensatingStep(id api.StepID) *api.Step {
	return &api.Step{
		ID:       id,
		Name:     "Compensating Step",
		Type:     api.StepTypeService,
		Handling: api.HandlingCompensated,
		HTTP: &api.HTTPConfig{
			Invoke: api.HTTPAction{Endpoint: "http://test:8080/work"},
			Compensate: &api.HTTPAction{
				Endpoint: "http://test:8080/compensate",
			},
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
	}
	evs = append(evs, succeededWorkEvents(args.id, args.step.ID, args.token)...)
	evs = append(evs, []helpers.FlowEvent{
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
	}...)

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
