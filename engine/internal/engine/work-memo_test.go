package engine_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/assert/wait"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/config"
	"github.com/kode4food/argyll/engine/pkg/flow"
)

type memoRejectBackend struct {
	timebox.Backend
	reject atomic.Bool
}

const (
	memoConcurrentWriters = 2
	memoTTL               = 1000
	memoShortTTL          = 500
	memoPollInterval      = 10 * time.Millisecond
)

var (
	ErrMemoAppend = errors.New("rejected memo success")
)

func TestMemoizedPersistence(t *testing.T) {
	for _, restart := range []bool{false, true} {
		name := "other node"
		if restart {
			name = "restart"
		}
		t.Run(name, func(t *testing.T) {
			backend := memory.Open()
			helpers.WithTestBackend(t, backend,
				func(env *helpers.TestEngineEnv) {
					st := helpers.NewTestStep()
					st.Handling = api.HandlingMemoized
					pl := &api.ExecutionPlan{
						Goals: api.Goals{Steps: []api.StepID{st.ID}},
						Steps: api.Steps{st.ID: st},
					}
					assert.NoError(t, env.Engine.RegisterStep(st))
					env.MockClient.SetResponse(
						st.ID, api.Args{"output": "saved"},
					)
					other, err := env.NewEngineInstance()
					if !assert.NoError(t, err) {
						return
					}
					assert.NoError(t, env.Engine.StartPlan(
						"source", pl, flow.WithInit(api.InitArgs{"input": {"value"}}),
					))
					fl := env.WaitForTerminalFlow("source")
					assert.Equal(t, api.FlowCompleted, fl.Status)
					if restart {
						assert.NoError(t, env.Engine.Stop())
						assert.NoError(t, other.Stop())
						other, err = env.NewEngineInstance()
					}
					if !assert.NoError(t, err) {
						return
					}
					defer func() { assert.NoError(t, other.Stop()) }()
					assert.NoError(t, other.StartPlan(
						"reused", pl, flow.WithInit(api.InitArgs{"input": {"value"}}),
					))
					fl, err = other.GetFlowState("reused")
					assert.NoError(t, err)
					assert.Equal(t, api.FlowCompleted, fl.Status)
					assert.Equal(t, "saved", fl.Executions[st.ID].Outputs["output"])
					assert.Len(t, env.MockClient.GetInvocations(), 1)
				})
		})
	}
}

func TestMemoizedConcurrentResults(t *testing.T) {
	backend := memory.Open()
	helpers.WithTestBackend(t, backend, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("concurrent-memo")
		st.Handling = api.HandlingMemoized
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))
		other, err := env.NewEngineInstance()
		if !assert.NoError(t, err) {
			return
		}
		defer func() { assert.NoError(t, other.Stop()) }()
		first := api.FlowStep{
			FlowID: "first",
			StepID: st.ID,
		}
		second := api.FlowStep{
			FlowID: "second",
			StepID: st.ID,
		}
		late := api.FlowStep{
			FlowID: "late",
			StepID: st.ID,
		}
		assert.NoError(t, env.SeedStartedWork(first, pl, "first"))
		assert.NoError(t, env.SeedStartedWork(second, pl, "second"))
		assert.NoError(t, env.SeedStartedWork(late, pl, "late"))
		ready := make(chan struct{})
		done := make(chan error, memoConcurrentWriters)
		go func() {
			<-ready
			done <- env.Engine.CompleteWork(first, "first", api.Args{"raw": 1})
		}()
		go func() {
			<-ready
			done <- other.CompleteWork(second, "second", api.Args{"raw": 2})
		}()
		close(ready)
		assert.NoError(t, <-done)
		assert.NoError(t, <-done)
		assert.NoError(t,
			env.Engine.CompleteWork(late, "late", api.Args{"raw": 3}),
		)
		assert.NoError(t, env.Engine.StartPlan("reuse-first", pl))
		assert.NoError(t, other.StartPlan("reuse-second", pl))
		var reused []any
		for _, fid := range []api.FlowID{"reuse-first", "reuse-second"} {
			fl := env.WaitForTerminalFlow(fid)
			for _, work := range fl.Executions[st.ID].WorkItems {
				reused = append(reused, work.Outputs["raw"])
			}
		}
		if assert.Len(t, reused, 2) {
			assert.Contains(t, []any{1.0, 2.0}, reused[0])
			assert.Equal(t, reused[0], reused[1])
		}
		assert.Empty(t, env.MockClient.GetInvocations())
	})
}

func TestMemoizedTTL(t *testing.T) {
	var elapsed atomic.Int64
	base := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnvDeps(t, engine.Dependencies{
		Clock: func() time.Time {
			return base.Add(time.Duration(elapsed.Load()))
		},
	}, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("ttl-memo")
		st.Handling = api.HandlingMemoized
		st.WorkConfig = &api.WorkConfig{MemoTTL: memoTTL}
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		run := func(fid api.FlowID, after time.Duration, raw string) {
			elapsed.Store(int64(after))
			env.MockClient.SetResponse(st.ID, api.Args{"raw": raw})
			assert.NoError(t, env.Engine.StartPlan(fid, pl))
			fl := env.WaitForTerminalFlow(fid)
			for _, work := range fl.Executions[st.ID].WorkItems {
				assert.Equal(t, "first", work.Outputs["raw"])
			}
		}
		run("first", 0, "first")
		run("within", memoTTL*time.Millisecond/2, "second")
		assert.Len(t, env.MockClient.GetInvocations(), 1)

		elapsed.Store(int64(memoTTL * time.Millisecond))
		env.MockClient.SetResponse(st.ID, api.Args{"raw": "fresh"})
		assert.NoError(t, env.Engine.StartPlan("expired", pl))
		fl := env.WaitForTerminalFlow("expired")
		for _, work := range fl.Executions[st.ID].WorkItems {
			assert.Equal(t, "fresh", work.Outputs["raw"])
		}
		assert.Len(t, env.MockClient.GetInvocations(), 2)
	})
}

func TestMemoizedDefaultTTL(t *testing.T) {
	var elapsed atomic.Int64
	base := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	helpers.WithTestEnvDeps(t, engine.Dependencies{
		Clock: func() time.Time {
			return base.Add(time.Duration(elapsed.Load()))
		},
	}, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("default-ttl-memo")
		st.Handling = api.HandlingMemoized
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))

		ttl := time.Duration(config.DefaultMemoTTL) * time.Millisecond
		run := func(fid api.FlowID, after time.Duration, raw, want string) {
			elapsed.Store(int64(after))
			env.MockClient.SetResponse(st.ID, api.Args{"raw": raw})
			assert.NoError(t, env.Engine.StartPlan(fid, pl))
			fl := env.WaitForTerminalFlow(fid)
			for _, work := range fl.Executions[st.ID].WorkItems {
				assert.Equal(t, want, work.Outputs["raw"])
			}
		}
		run("first", 0, "first", "first")
		run("within", ttl-time.Millisecond, "second", "first")
		assert.Len(t, env.MockClient.GetInvocations(), 1)
		run("expired", ttl, "fresh", "fresh")
		assert.Len(t, env.MockClient.GetInvocations(), 2)
	})
}

func TestMemoizedExpiryConsumed(t *testing.T) {
	if testing.Short() {
		t.Skip("waits on a real scheduler timer")
	}
	helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("expiring-memo")
		st.Handling = api.HandlingMemoized
		st.WorkConfig = &api.WorkConfig{MemoTTL: memoShortTTL}
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))
		assert.NoError(t, env.Engine.Start())
		assert.NoError(t, env.Engine.StartPlan("expiring", pl))
		env.WaitForTerminalFlow("expiring")
		memoPrefix := base64.RawURLEncoding.EncodeToString([]byte("memo"))
		hasMemo := func() bool {
			keys, err := env.ScheduleKeys()
			assert.NoError(t, err)
			for key := range keys {
				if strings.HasPrefix(string(key), memoPrefix+"/") {
					return true
				}
			}
			return false
		}
		assert.True(t, hasMemo())
		assert.Eventually(t, func() bool {
			return !hasMemo()
		}, wait.DefaultTimeout, memoPollInterval)
	})
}

func TestMemoizedRollback(t *testing.T) {
	backend := &memoRejectBackend{Backend: memory.Open()}
	helpers.WithTestBackend(t, backend, func(env *helpers.TestEngineEnv) {
		st := helpers.NewSimpleStep("rollback-memo")
		st.Handling = api.HandlingMemoized
		pl := &api.ExecutionPlan{
			Goals: api.Goals{Steps: []api.StepID{st.ID}},
			Steps: api.Steps{st.ID: st},
		}
		assert.NoError(t, env.Engine.RegisterStep(st))
		fs := api.FlowStep{
			FlowID: "rollback",
			StepID: st.ID,
		}
		assert.NoError(t, env.SeedStartedWork(fs, pl, "token"))
		backend.reject.Store(true)
		err := env.Engine.CompleteWork(fs, "token", api.Args{"raw": "lost"})
		assert.ErrorIs(t, err, ErrMemoAppend)
		env.MockClient.SetResponse(st.ID, api.Args{"raw": "fresh"})
		assert.NoError(t, env.Engine.StartPlan("after-rollback", pl))
		fl := env.WaitForTerminalFlow("after-rollback")
		assert.Equal(t, api.FlowCompleted, fl.Status)
		for _, work := range fl.Executions[st.ID].WorkItems {
			assert.Equal(t, "fresh", work.Outputs["raw"])
		}
		assert.Len(t, env.MockClient.GetInvocations(), 1)
	})
}

func TestMemoizedWorkHistory(t *testing.T) {
	for _, status := range []api.WorkStatus{
		api.WorkSucceeded, api.WorkFailed, api.WorkNotCompleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			helpers.WithTestEnv(t, func(env *helpers.TestEngineEnv) {
				st := helpers.NewTestStepWithArgs([]api.Name{"item"}, nil)
				st.Handling = api.HandlingMemoized
				st.Attributes["item"].Required = &api.RequiredConfig{
					ForEach: true,
					Collect: api.InputCollectAll,
				}
				pl := &api.ExecutionPlan{
					Goals: api.Goals{Steps: []api.StepID{st.ID}},
					Steps: api.Steps{st.ID: st},
				}
				pl.Attributes = api.AttributeGraph{
					"item": {Consumers: []api.StepID{st.ID}},
				}
				assert.NoError(t, env.Engine.RegisterStep(st))
				assert.NoError(t, env.RaiseFlowEvents(
					"history",
					helpers.FlowEvent{
						Type: api.EventTypeFlowStarted,
						Data: api.FlowStartedEvent{
							FlowID: "history",
							Plan:   pl,
						},
					},
					helpers.FlowEvent{
						Type: api.EventTypeStepStarted,
						Data: api.StepStartedEvent{
							FlowID: "history",
							StepID: st.ID,
							Inputs: api.Args{"item": []any{"a", "b"}},
							WorkItems: map[api.Token]api.Args{
								"first":  {"item": "a"},
								"second": {"item": "b"},
							},
						},
					},
				))
				ev := helpers.FlowEvent{
					Type: api.EventTypeWorkStarted,
					Data: api.WorkStartedEvent{
						FlowID: "history",
						StepID: st.ID,
						Token:  "first",
						Inputs: api.Args{"item": "a"},
					},
				}
				switch status {
				case api.WorkFailed:
					ev = helpers.FlowEvent{
						Type: api.EventTypeWorkFailed,
						Data: api.WorkFailedEvent{
							FlowID: "history",
							StepID: st.ID,
							Token:  "first",
						},
					}
				case api.WorkNotCompleted:
					ev = helpers.FlowEvent{
						Type: api.EventTypeWorkNotCompleted,
						Data: api.WorkNotCompletedEvent{
							FlowID: "history",
							StepID: st.ID,
							Token:  "first",
						},
					}
				}
				assert.NoError(t, env.RaiseFlowEvents(
					"history", ev,
					helpers.FlowEvent{
						Type: api.EventTypeStepFailed,
						Data: api.StepFailedEvent{
							FlowID: "history",
							StepID: st.ID,
						},
					},
					helpers.FlowEvent{
						Type: api.EventTypeFlowFailed,
						Data: api.FlowFailedEvent{FlowID: "history"},
					},
				))
				if status == api.WorkSucceeded {
					assert.NoError(t, env.Engine.CompleteWork(
						api.FlowStep{
							FlowID: "history",
							StepID: st.ID,
						}, "first",
						api.Args{"raw": "saved"},
					))
				}
				other, err := env.NewEngineInstance()
				if !assert.NoError(t, err) {
					return
				}
				defer func() { assert.NoError(t, other.Stop()) }()
				env.MockClient.SetResponse(st.ID, api.Args{"raw": "fresh"})
				assert.NoError(t, other.StartPlan(
					"later", pl, flow.WithInit(api.InitArgs{"item": {"a"}}),
				))
				fl := env.WaitForTerminalFlow("later")
				assert.Equal(t, api.FlowCompleted, fl.Status)
				want := "fresh"
				invocations := 1
				if status == api.WorkSucceeded {
					want = "saved"
					invocations = 0
				}
				for _, work := range fl.Executions[st.ID].WorkItems {
					assert.Equal(t, want, work.Outputs["raw"])
				}
				assert.Len(t, env.MockClient.GetInvocations(), invocations)
			})
		})
	}
}

// Append rejects one successful work append when armed
func (m *memoRejectBackend) Append(reqs ...timebox.AppendRequest) error {
	for _, req := range reqs {
		for _, ev := range req.Events {
			if api.EventType(ev.Type) == api.EventTypeWorkSucceeded &&
				m.reject.CompareAndSwap(true, false) {
				return ErrMemoAppend
			}
		}
	}
	return m.Backend.Append(reqs...)
}
