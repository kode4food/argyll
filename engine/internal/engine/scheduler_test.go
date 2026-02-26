package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

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
