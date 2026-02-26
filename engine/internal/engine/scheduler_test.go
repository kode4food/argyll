package engine

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type (
	testEnv struct {
		eng *Engine
		tb  *timebox.Timebox
		rd  *miniredis.Miniredis
	}

	nopClient struct{}

	flowEvent struct {
		typ  api.EventType
		data any
	}
)

var _ client.Client = (*nopClient)(nil)

func (nopClient) Invoke(*api.Step, api.Args, api.Metadata) (api.Args, error) {
	return api.Args{}, nil
}

func TestTaskHeapKeyed(t *testing.T) {
	now := time.Now()
	h := NewTaskHeap()
	noop := func() error { return nil }
	insert := func(path []string, at time.Time) {
		h.Insert(&Task{Path: path, At: at, Func: noop})
	}

	insert([]string{"a"}, now.Add(3*time.Second))
	insert([]string{"b"}, now.Add(2*time.Second))
	insert([]string{"a"}, now.Add(time.Second))

	peek := h.Peek()
	if assert.NotNil(t, peek) {
		assert.Equal(t, []string{"a"}, []string(peek.Path))
		assert.Equal(t, now.Add(time.Second).Unix(), peek.At.Unix())
	}

	h.Cancel([]string{"a"})
	peek = h.Peek()
	if assert.NotNil(t, peek) {
		assert.Equal(t, []string{"b"}, []string(peek.Path))
	}

	insert([]string{"retry", "f1", "s1", "t1"}, now)
	insert([]string{"retry", "f1", "s2", "t2"}, now)
	insert([]string{"retry", "f2", "s1", "t1"}, now)

	h.CancelPrefix([]string{"retry", "f1"})
	for {
		task := h.PopTask()
		if task == nil {
			break
		}
		assert.False(t, len(task.Path) >= 2 &&
			task.Path[0] == "retry" && task.Path[1] == "f1")
	}
}

func TestScheduleRetryTask(t *testing.T) {
	e := &Engine{
		ctx:   context.Background(),
		tasks: make(chan taskReq, 2),
	}
	at := time.Now().Add(time.Second)
	fs := api.FlowStep{FlowID: "flow-1", StepID: "step-1"}
	e.scheduleRetryTask(fs, "tok-1", at)

	select {
	case req := <-e.tasks:
		assert.Equal(t, taskReqSchedule, req.op)
		if assert.NotNil(t, req.task) {
			assert.Equal(t, retryKey(fs, "tok-1"), []string(req.task.Path))
			assert.NotNil(t, req.task.Func)
			assert.Equal(t, at.Unix(), req.task.At.Unix())
		}
	default:
		t.Fatal("expected scheduled retry task request")
	}
}

func TestCancelScheduledTaskRequests(t *testing.T) {
	e := &Engine{
		ctx:   context.Background(),
		tasks: make(chan taskReq, 2),
	}
	key := retryKey(
		api.FlowStep{FlowID: "f", StepID: "s"}, api.Token("t"),
	)
	e.CancelScheduledTask(key)
	e.CancelScheduledTaskPrefix(retryPrefix("f"))

	first := <-e.tasks
	assert.Equal(t, taskReqCancel, first.op)
	assert.Equal(t, key, []string(first.key))

	second := <-e.tasks
	assert.Equal(t, taskReqCancelPrefix, second.op)
	assert.Equal(t, retryPrefix("f"), []string(second.prefix))
}

func TestScheduleTaskRequest(t *testing.T) {
	e := &Engine{
		ctx:   context.Background(),
		tasks: make(chan taskReq, 1),
	}
	e.ScheduleTask(time.Now(), func() error { return nil })

	req := <-e.tasks
	assert.Equal(t, taskReqSchedule, req.op)
	if assert.NotNil(t, req.task) {
		assert.Nil(t, req.task.Path)
		assert.NotNil(t, req.task.Func)
		assert.False(t, req.task.At.IsZero())
	}
}

func TestTaskHeapNoOps(t *testing.T) {
	h := NewTaskHeap()
	assert.Nil(t, h.PopTask())

	h.Insert(nil)
	h.Insert(&Task{At: time.Now()})
	h.Insert(&Task{Func: func() error { return nil }})
	assert.Nil(t, h.Peek())

	h.Cancel(nil)
	h.Cancel([]string{"missing"})
	h.CancelPrefix(nil)
	h.CancelPrefix([]string{"missing"})
	assert.Nil(t, h.Peek())
}

func TestTaskHeapPopNonKeyed(t *testing.T) {
	h := NewTaskHeap()
	h.Insert(&Task{
		At:   time.Now(),
		Func: func() error { return nil },
	})

	task := h.PopTask()
	if assert.NotNil(t, task) {
		assert.Nil(t, task.Path)
	}
	assert.Nil(t, h.PopTask())
}

func TestTaskPathHelpers(t *testing.T) {
	assert.Nil(t, clonePath(nil))
	assert.Nil(t, clonePath([]string{}))

	src := []string{"a", "b"}
	cp := clonePath(src)
	src[0] = "x"
	assert.Equal(t, []string{"a", "b"}, []string(cp))

	assert.Equal(t, "", taskPathID(nil))
	assert.Equal(t, "", taskPathID([]string{}))
	assert.NotEqual(t, taskPathID([]string{"a"}), taskPathID([]string{"b"}))
}

func TestRunRetryTask(t *testing.T) {
	t.Run("missing flow returns nil", func(t *testing.T) {
		withTestEngine(t, func(te *testEnv) {
			err := te.eng.runRetryTask(api.FlowStep{
				FlowID: "missing", StepID: "step",
			}, "tok")
			assert.NoError(t, err)
		})
	})

	t.Run("terminal flow no-op", func(t *testing.T) {
		withTestEngine(t, func(te *testEnv) {
			step := simpleStep("s1")
			plan := &api.ExecutionPlan{
				Goals: []api.StepID{step.ID},
				Steps: api.Steps{step.ID: step},
			}
			flowID := api.FlowID("retry-terminal")
			err := raiseFlowEvents(te.eng, flowID,
				flowEv(api.EventTypeFlowStarted,
					api.FlowStartedEvent{FlowID: flowID, Plan: plan}),
				flowEv(api.EventTypeFlowCompleted,
					api.FlowCompletedEvent{FlowID: flowID}),
			)
			assert.NoError(t, err)

			err = te.eng.runRetryTask(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			}, "tok")
			assert.NoError(t, err)
		})
	})

	t.Run("missing token no-op", func(t *testing.T) {
		withTestEngine(t, func(te *testEnv) {
			step := simpleStep("s1")
			plan := &api.ExecutionPlan{
				Goals: []api.StepID{step.ID},
				Steps: api.Steps{step.ID: step},
			}
			flowID := api.FlowID("retry-missing-token")
			err := raiseFlowEvents(te.eng, flowID,
				flowEv(api.EventTypeFlowStarted,
					api.FlowStartedEvent{FlowID: flowID, Plan: plan}),
				flowEv(api.EventTypeStepStarted, api.StepStartedEvent{
					FlowID: flowID,
					StepID: step.ID,
					Inputs: api.Args{},
					WorkItems: map[api.Token]api.Args{
						"other": {},
					},
				}),
			)
			assert.NoError(t, err)

			err = te.eng.runRetryTask(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			}, "tok")
			assert.NoError(t, err)
		})
	})

	t.Run("missing step no-op", func(t *testing.T) {
		withTestEngine(t, func(te *testEnv) {
			step := simpleStep("s1")
			plan := &api.ExecutionPlan{
				Goals: []api.StepID{step.ID},
				Steps: api.Steps{step.ID: step},
			}
			flowID := api.FlowID("retry-missing-step")
			err := raiseFlowEvents(te.eng, flowID,
				flowEv(api.EventTypeFlowStarted,
					api.FlowStartedEvent{FlowID: flowID, Plan: plan}),
			)
			assert.NoError(t, err)

			err = te.eng.runRetryTask(api.FlowStep{
				FlowID: flowID,
				StepID: "missing-step",
			}, "tok")
			assert.NoError(t, err)
		})
	})
}

func TestRunTimeoutTask(t *testing.T) {
	t.Run("terminal flow no-op", func(t *testing.T) {
		withTestEngine(t, func(te *testEnv) {
			step := simpleStep("s1")
			plan := &api.ExecutionPlan{
				Goals: []api.StepID{step.ID},
				Steps: api.Steps{step.ID: step},
			}
			flowID := api.FlowID("timeout-terminal")
			err := raiseFlowEvents(te.eng, flowID,
				flowEv(api.EventTypeFlowStarted,
					api.FlowStartedEvent{FlowID: flowID, Plan: plan}),
				flowEv(api.EventTypeFlowCompleted,
					api.FlowCompletedEvent{FlowID: flowID}),
			)
			assert.NoError(t, err)

			err = te.eng.runTimeoutTask(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			})
			assert.NoError(t, err)
		})
	})

	t.Run("active step no-op", func(t *testing.T) {
		withTestEngine(t, func(te *testEnv) {
			step := simpleStep("s1")
			plan := &api.ExecutionPlan{
				Goals: []api.StepID{step.ID},
				Steps: api.Steps{step.ID: step},
			}
			flowID := api.FlowID("timeout-active-step")
			err := raiseFlowEvents(te.eng, flowID,
				flowEv(api.EventTypeFlowStarted,
					api.FlowStartedEvent{FlowID: flowID, Plan: plan}),
				flowEv(api.EventTypeStepStarted, api.StepStartedEvent{
					FlowID:    flowID,
					StepID:    step.ID,
					Inputs:    api.Args{},
					WorkItems: map[api.Token]api.Args{"t": {}},
				}),
			)
			assert.NoError(t, err)

			err = te.eng.runTimeoutTask(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			})
			assert.NoError(t, err)
		})
	})

	t.Run("pending but not ready no-op", func(t *testing.T) {
		withTestEngine(t, func(te *testEnv) {
			step := simpleStep("s1")
			step.Attributes = api.AttributeSpecs{
				"req": {Role: api.RoleRequired, Type: api.TypeString},
			}
			plan := &api.ExecutionPlan{
				Goals: []api.StepID{step.ID},
				Steps: api.Steps{step.ID: step},
				Attributes: api.AttributeGraph{
					"req": {Consumers: []api.StepID{step.ID}},
				},
			}
			flowID := api.FlowID("timeout-not-ready")
			err := raiseFlowEvents(te.eng, flowID,
				flowEv(api.EventTypeFlowStarted,
					api.FlowStartedEvent{FlowID: flowID, Plan: plan}),
			)
			assert.NoError(t, err)

			err = te.eng.runTimeoutTask(api.FlowStep{
				FlowID: flowID,
				StepID: step.ID,
			})
			assert.NoError(t, err)
		})
	})

	t.Run("missing flow no-op", func(t *testing.T) {
		withTestEngine(t, func(te *testEnv) {
			err := te.eng.runTimeoutTask(api.FlowStep{
				FlowID: "missing",
				StepID: "step",
			})
			assert.NoError(t, err)
		})
	})
}

func TestScheduleTimeoutsNoTasks(t *testing.T) {
	e := &Engine{
		ctx:   context.Background(),
		tasks: make(chan taskReq, 4),
	}
	flow := &api.FlowState{
		ID:         "f1",
		Status:     api.FlowCompleted,
		CreatedAt:  time.Now(),
		Plan:       &api.ExecutionPlan{Steps: api.Steps{}},
		Executions: api.Executions{},
	}
	e.scheduleTimeouts(flow, time.Now())

	first := <-e.tasks
	assert.Equal(t, taskReqCancelPrefix, first.op)
	assert.Equal(t, timeoutFlowPrefix(flow.ID), []string(first.prefix))
	select {
	case <-e.tasks:
		t.Fatal("expected no timeout schedule requests")
	default:
	}
}

func TestRecoverRetryWorkSchedulesTasks(t *testing.T) {
	e := &Engine{
		ctx:   context.Background(),
		tasks: make(chan taskReq, 8),
	}
	now := time.Now()
	flow := &api.FlowState{
		ID:     "f",
		Status: api.FlowActive,
		Plan: &api.ExecutionPlan{
			Goals: []api.StepID{"s1"},
			Steps: api.Steps{"s1": simpleStep("s1")},
		},
		Executions: api.Executions{
			"s1": {
				Status: api.StepActive,
				WorkItems: api.WorkItems{
					"a": {Status: api.WorkActive},
					"n": {Status: api.WorkNotCompleted},
					"p": {
						Status:      api.WorkPending,
						NextRetryAt: now.Add(time.Second),
					},
					"f": {
						Status:      api.WorkFailed,
						NextRetryAt: now.Add(2 * time.Second),
					},
					"x": {Status: api.WorkSucceeded},
				},
			},
		},
	}

	e.recoverRetryWork(flow)

	var reqs []taskReq
	for {
		select {
		case req := <-e.tasks:
			reqs = append(reqs, req)
		default:
			assert.Len(t, reqs, 4)
			for _, req := range reqs {
				assert.Equal(t, taskReqSchedule, req.op)
				assert.NotNil(t, req.task)
			}
			return
		}
	}
}

func withTestEngine(t *testing.T, fn func(*testEnv)) {
	t.Helper()

	rd, err := miniredis.Run()
	assert.NoError(t, err)

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)

	catCfg := config.NewDefaultConfig().CatalogStore
	catCfg.Addr = rd.Addr()
	catCfg.Prefix = "test-catalog"
	catStore, err := tb.NewStore(catCfg)
	assert.NoError(t, err)

	partCfg := config.NewDefaultConfig().PartitionStore
	partCfg.Addr = rd.Addr()
	partCfg.Prefix = "test-partition"
	partStore, err := tb.NewStore(partCfg)
	assert.NoError(t, err)

	flowCfg := config.NewDefaultConfig().FlowStore
	flowCfg.Addr = rd.Addr()
	flowCfg.Prefix = "test-flow"
	flowStore, err := tb.NewStore(flowCfg)
	assert.NoError(t, err)

	cfg := config.NewDefaultConfig()
	eng, err := New(catStore, partStore, flowStore, nopClient{}, cfg)
	assert.NoError(t, err)

	te := &testEnv{eng: eng, tb: tb, rd: rd}
	defer func() {
		_ = te.eng.Stop()
		_ = te.tb.Close()
		te.rd.Close()
	}()
	fn(te)
}

func raiseFlowEvents(
	eng *Engine, flowID api.FlowID, evs ...flowEvent,
) error {
	_, err := eng.flowExec.Exec(context.Background(), events.FlowKey(flowID),
		func(_ *api.FlowState, ag *FlowAggregator) error {
			for _, ev := range evs {
				if err := events.Raise(ag, ev.typ, ev.data); err != nil {
					return err
				}
			}
			return nil
		},
	)
	return err
}

func flowEv(typ api.EventType, data any) flowEvent {
	return flowEvent{typ: typ, data: data}
}

func simpleStep(id api.StepID) *api.Step {
	return &api.Step{ID: id, Type: api.StepTypeSync}
}
