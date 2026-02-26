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

	h.Insert(&Task{Key: "a", At: now.Add(3 * time.Second),
		Func: func() error { return nil }})
	h.Insert(&Task{Key: "b", At: now.Add(2 * time.Second),
		Func: func() error { return nil }})
	h.Insert(&Task{Key: "a", At: now.Add(time.Second),
		Func: func() error { return nil }})

	peek := h.Peek()
	if assert.NotNil(t, peek) {
		assert.Equal(t, "a", peek.Key)
		assert.Equal(t, now.Add(time.Second).Unix(), peek.At.Unix())
	}

	h.Cancel("a")
	peek = h.Peek()
	if assert.NotNil(t, peek) {
		assert.Equal(t, "b", peek.Key)
	}

	h.Insert(&Task{Key: "retry/f1/s1/t1", At: now,
		Func: func() error { return nil }})
	h.Insert(&Task{Key: "retry/f1/s2/t2", At: now,
		Func: func() error { return nil }})
	h.Insert(&Task{Key: "retry/f2/s1/t1", At: now,
		Func: func() error { return nil }})

	h.CancelPrefix("retry/f1/")
	for {
		task := h.PopTask()
		if task == nil {
			break
		}
		assert.NotContains(t, task.Key, "retry/f1/")
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
			assert.Equal(t,
				retryTaskKey(fs, "tok-1"), req.task.Key,
			)
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
	key := retryTaskKey(
		api.FlowStep{FlowID: "f", StepID: "s"}, api.Token("t"),
	)
	e.CancelScheduledTask(key)
	e.CancelScheduledTaskPrefix(retryTaskPrefix("f"))

	first := <-e.tasks
	assert.Equal(t, taskReqCancel, first.op)
	assert.Equal(t, key, first.key)

	second := <-e.tasks
	assert.Equal(t, taskReqCancelPrefix, second.op)
	assert.Equal(t, retryTaskPrefix("f"), second.prefix)
}
