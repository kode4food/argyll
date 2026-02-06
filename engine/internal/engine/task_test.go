package engine_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine"
)

const taskTimeout = 1 * time.Second

func TestTaskRunnerExecutesTasksInOrder(t *testing.T) {
	runner := engine.NewTaskRunner()
	runner.Start()
	t.Cleanup(runner.Flush)

	var mu sync.Mutex
	var order []int
	done := make(chan struct{})

	runner.Enqueue(func() {
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
	})
	runner.Enqueue(func() {
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
	})
	runner.Enqueue(func() {
		mu.Lock()
		order = append(order, 3)
		mu.Unlock()
		close(done)
	})

	select {
	case <-done:
	case <-time.After(taskTimeout):
		assert.Fail(t, "timed out waiting for tasks")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []int{1, 2, 3}, order)
}
