package engine

import (
	"log/slog"
	"sync"

	"github.com/kode4food/caravan"
	"github.com/kode4food/caravan/message"
	"github.com/kode4food/caravan/topic"
)

type (
	// TaskRunner executes queued tasks sequentially
	TaskRunner struct {
		queue    topic.Topic[Task]
		prod     topic.Producer[Task]
		cons     topic.Consumer[Task]
		stop     chan struct{}
		stopOnce sync.Once
		started  sync.Once
		runWG    sync.WaitGroup
	}

	Task func()
)

// NewTaskRunner creates a new task runner
func NewTaskRunner() *TaskRunner {
	queue := caravan.NewTopic[Task]()
	tr := &TaskRunner{
		queue: queue,
		prod:  queue.NewProducer(),
		cons:  queue.NewConsumer(),
		stop:  make(chan struct{}),
	}
	return tr
}

// Start begins processing queued tasks
func (t *TaskRunner) Start() {
	t.started.Do(func() {
		t.runWG.Go(func() {
			for {
				select {
				case <-t.stop:
					return
				case fn, ok := <-t.cons.Receive():
					if !ok {
						return
					}
					t.runTask(fn)
				}
			}
		})
	})
}

// Enqueue adds a task to the queue
func (t *TaskRunner) Enqueue(fn Task) {
	if fn == nil {
		return
	}
	message.Send(t.prod, fn)
}

// Flush waits for queued tasks to complete and stops the runner
func (t *TaskRunner) Flush() {
	t.stopOnce.Do(func() {
		close(t.stop)
	})
	t.runWG.Wait()
	for {
		select {
		case fn, ok := <-t.cons.Receive():
			if !ok {
				t.prod.Close()
				t.cons.Close()
				return
			}
			t.runTask(fn)
		default:
			t.prod.Close()
			t.cons.Close()
			return
		}
	}
}

func (t *TaskRunner) runTask(fn Task) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Engine task panic",
				slog.Any("panic", r))
		}
	}()
	fn()
}
