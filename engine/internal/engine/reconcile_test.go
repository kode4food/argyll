package engine_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

var errTransient = fmt.Errorf(
	"%w: transient failure", api.ErrWorkNotCompleted,
)

// TestCommittedEventWakesIdleReplica proves a scheduled retry is reconstructed
// from the committed batch alone, with no call to RecoverFlow
func TestCommittedEventWakesIdleReplica(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("reconcile-retry-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		id := api.FlowID("wf-reconcile-retry")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-a")

		env.WaitFor(wait.WorkStarted(fs), func() {
			seedRetryScheduledFlow(env, fs, tkn)
		})

		assert.True(t,
			env.MockClient.WaitForInvocation(st.ID, wait.DefaultTimeout),
		)
	})
}

// TestDuplicateCommittedBatchStartsWorkOnce proves the deterministic task keys
// collapse a batch that a replica sees more than once
func TestDuplicateCommittedBatchStartsWorkOnce(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("reconcile-dup-step")
		assert.NoError(t, env.Engine.RegisterStep(st))

		invocations := make(chan api.StepID, 4)
		env.MockClient.SetHandler(st.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				invocations <- st.ID
				return api.Args{}, nil
			},
		)

		id := api.FlowID("wf-reconcile-dup")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-a")

		env.WaitFor(wait.WorkStarted(fs), func() {
			seedRetryScheduledFlow(env, fs, tkn)
			// The same batch reaching a replica twice must not run it twice
			evs, err := env.Engine.GetFlowEvents(id)
			assert.NoError(t, err)
			env.Engine.HandleCommitted(evs...)
		})

		assert.Equal(t, st.ID, <-invocations)
		select {
		case <-invocations:
			t.Fatal("work was started more than once")
		case <-time.After(300 * time.Millisecond):
		}
	})
}

// TestBatchUsesFinalState proves a handler reads the projection at the end of
// the committed batch. The intermediate state carries a lapsed retry, so a
// per-event handler would dispatch work the final state has already settled
func TestBatchUsesFinalState(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("reconcile-batch-step")
		assert.NoError(t, env.Engine.RegisterStep(st))

		invocations := make(chan api.StepID, 4)
		env.MockClient.SetHandler(st.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				invocations <- st.ID
				return api.Args{}, nil
			},
		)

		id := api.FlowID("wf-reconcile-batch")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-a")

		seedRetryScheduledFlow(env, fs, tkn, helpers.FlowEvent{
			Type: api.EventTypeWorkSucceeded,
			Data: api.WorkSucceededEvent{
				FlowID:  id,
				StepID:  st.ID,
				Token:   tkn,
				Outputs: api.Args{},
			},
		})

		select {
		case <-invocations:
			t.Fatal("settled work was dispatched from an intermediate state")
		case <-time.After(300 * time.Millisecond):
		}

		fl, err := env.Engine.GetFlowState(id)
		assert.NoError(t, err)
		assert.Equal(t,
			api.WorkSucceeded, fl.Executions[st.ID].WorkItems[tkn].Status,
		)
	})
}

// TestConcurrentReplicasStartWorkOnce proves optimistic concurrency lets only
// one replica commit the claim when both reconcile the same condition
func TestConcurrentReplicasStartWorkOnce(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		cfg := util.MutableCopy(env.Config)
		cfg.Raft.LocalID = "node-concurrent"
		cfg.Raft.Servers = append(cfg.Raft.Servers,
			raft.Server{ID: "node-concurrent", Address: "127.0.0.1:9721"},
		)

		peer, unsub, err := env.NewEngineWithConfig(cfg, env.Dependencies())
		assert.NoError(t, err)
		defer func() {
			unsub()
			assert.NoError(t, peer.Stop())
		}()

		st := helpers.NewSimpleStep("reconcile-concurrent-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})

		assert.NoError(t, env.Engine.Start())
		assert.NoError(t, peer.Start())

		id := api.FlowID("wf-reconcile-concurrent")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-a")

		env.WaitFor(wait.WorkStarted(fs), func() {
			seedRetryScheduledFlow(env, fs, tkn)
		})

		// Both replicas race the same still-projected condition
		var wg sync.WaitGroup
		for _, eng := range []*engine.Engine{env.Engine, peer} {
			wg.Go(func() {
				assert.NoError(t, eng.RecoverFlow(id))
			})
		}
		wg.Wait()

		fl := helpers.WaitForFlowState(t, env.Engine, helpers.FlowStateQuery{
			FlowID:  id,
			Timeout: wait.DefaultTimeout,
			Accept: func(fl api.FlowState) bool {
				return fl.Executions[st.ID].WorkItems[tkn].Status ==
					api.WorkSucceeded
			},
		})
		assert.Equal(t,
			api.WorkSucceeded, fl.Executions[st.ID].WorkItems[tkn].Status,
		)
		assert.Len(t, env.MockClient.GetInvocations(), 1)
	})
}

// TestTransientFailureRearms proves a failing attempt keeps being retried
// while its projected retry condition stands
func TestTransientFailureRearms(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		assert.NoError(t, env.Engine.Start())

		st := helpers.NewSimpleStep("reconcile-transient-step")
		st.WorkConfig = &api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 1,
			MaxBackoff:  1,
			BackoffType: api.BackoffTypeFixed,
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		var attempts atomic.Int32
		env.MockClient.SetHandler(st.ID,
			func(*api.Step, api.Args, api.Metadata) (api.Args, error) {
				if attempts.Add(1) < 3 {
					return nil, errTransient
				}
				return api.Args{}, nil
			},
		)

		id := api.FlowID("wf-reconcile-transient")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}
		assert.NoError(t, env.Engine.StartFlow(id, pl))

		fl := env.WaitForTerminalFlow(id)
		assert.Equal(t, api.FlowCompleted, fl.Status)
		assert.Equal(t, int32(3), attempts.Load())
	})
}

// TestScriptCannotStayActive proves a script whose executor died between the
// WorkStarted commit and the invocation is settled by its recovery deadline
func TestScriptCannotStayActive(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := &api.Step{
			ID:   "reconcile-script-step",
			Name: "Script Step",
			Type: api.StepTypeScript,
			Script: &api.ScriptConfig{
				Language: api.ScriptLangLua,
				Script:   "return {}",
			},
			Attributes: api.AttributeSpecs{},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		// A short step timeout, since that is what bounds a script
		cfg := util.MutableCopy(env.Config)
		cfg.StepTimeout = 200
		cfg.Raft.LocalID = "node-script"
		cfg.Raft.Servers = append(cfg.Raft.Servers,
			raft.Server{ID: "node-script", Address: "127.0.0.1:9722"},
		)

		peer, unsub, err := env.NewEngineWithConfig(cfg, env.Dependencies())
		assert.NoError(t, err)
		defer func() {
			unsub()
			assert.NoError(t, peer.Stop())
		}()

		id := api.FlowID("wf-reconcile-script")
		fs := api.FlowStep{FlowID: id, StepID: st.ID}
		tkn := api.Token("work-a")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}

		// The state a node leaves behind when it dies before invoking
		assert.NoError(t, env.SeedStartedWork(fs, pl, tkn))
		assert.NoError(t, peer.Start())

		fl := helpers.WaitForFlowState(t, peer, helpers.FlowStateQuery{
			FlowID:  id,
			Timeout: wait.DefaultTimeout,
			Accept: func(fl api.FlowState) bool {
				return fl.Status == api.FlowCompleted
			},
		})
		assert.Equal(t,
			api.WorkSucceeded, fl.Executions[st.ID].WorkItems[tkn].Status,
		)
	})
}

// TestDeferredDispatchResumesWhenHealthy proves work committed while no replica
// could act is picked up on recovery, with no call to RecoverFlow
func TestDeferredDispatchResumesWhenHealthy(t *testing.T) {
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("reconcile-deferred-step")
		assert.NoError(t, env.Engine.RegisterStep(st))
		env.MockClient.SetResponse(st.ID, api.Args{})
		assert.NoError(t, env.Engine.UpdateStepHealth(
			st.ID, api.HealthUnhealthy, "offline",
		))
		assert.NoError(t, env.Engine.Start())

		id := api.FlowID("wf-reconcile-deferred")
		pl := &api.ExecutionPlan{
			Goals: []api.StepID{st.ID},
			Steps: api.Steps{st.ID: st},
		}
		assert.NoError(t, env.Engine.StartFlow(id, pl))

		assert.False(t,
			env.MockClient.WaitForInvocation(st.ID, 300*time.Millisecond),
		)

		assert.NoError(t, env.Engine.UpdateStepHealth(
			st.ID, api.HealthHealthy, "",
		))
		assert.True(t,
			env.MockClient.WaitForInvocation(st.ID, wait.DefaultTimeout),
		)
	})
}

func seedRetryScheduledFlow(
	env *helpers.TestEngineEnv, fs api.FlowStep, tkn api.Token,
	extra ...helpers.FlowEvent,
) {
	pl := &api.ExecutionPlan{
		Goals: []api.StepID{fs.StepID},
		Steps: api.Steps{fs.StepID: mustStep(env, fs.StepID)},
	}
	evs := []helpers.FlowEvent{
		{
			Type: api.EventTypeFlowStarted,
			Data: api.FlowStartedEvent{
				FlowID: fs.FlowID,
				Plan:   pl,
				Init:   api.InitArgs{},
			},
		},
		{
			Type: api.EventTypeStepStarted,
			Data: api.StepStartedEvent{
				FlowID:    fs.FlowID,
				StepID:    fs.StepID,
				Inputs:    api.Args{},
				WorkItems: map[api.Token]api.Args{tkn: {}},
			},
		},
		{
			Type: api.EventTypeWorkRetryScheduled,
			Data: api.WorkRetryScheduledEvent{
				FlowID:      fs.FlowID,
				StepID:      fs.StepID,
				Token:       tkn,
				RetryCount:  1,
				NextRetryAt: time.Now().Add(-time.Second),
				Error:       "retry",
			},
		},
	}
	assert.NoError(env.T,
		env.RaiseFlowEvents(fs.FlowID, append(evs, extra...)...))
}

func mustStep(env *helpers.TestEngineEnv, sid api.StepID) *api.Step {
	cat, err := env.Engine.GetCatalogState()
	assert.NoError(env.T, err)
	return cat.Steps[sid]
}
