package engine

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// Engine is the core flow execution engine
	Engine struct {
		stepClient client.Client
		ctx        context.Context
		consumer   EventConsumer
		engineExec *Executor
		flowExec   *FlowExecutor
		config     *config.Config
		cancel     context.CancelFunc
		scripts    *ScriptRegistry
		wg         sync.WaitGroup
		flows      sync.Map // map[flowID]*flowActor
		handler    timebox.Handler
		retryQueue *RetryQueue
	}

	// EventConsumer consumes events from the event hub
	EventConsumer = topic.Consumer[*timebox.Event]

	// Executor manages engine state persistence and event sourcing
	Executor = timebox.Executor[*api.EngineState]

	// Aggregator aggregates engine state from events
	Aggregator = timebox.Aggregator[*api.EngineState]

	// FlowExecutor manages flow state persistence and event sourcing
	FlowExecutor = timebox.Executor[*api.FlowState]

	// FlowAggregator aggregates flow state from events
	FlowAggregator = timebox.Aggregator[*api.FlowState]
)

var (
	ErrShutdownTimeout     = errors.New("shutdown timeout exceeded")
	ErrFlowNotFound        = errors.New("flow not found")
	ErrFlowExists          = errors.New("flow exists")
	ErrStepNotFound        = errors.New("step not found")
	ErrStepExists          = errors.New("step exists")
	ErrScriptCompileFailed = errors.New("failed to compile scripts for plan")
	ErrStepNotInPlan       = errors.New("step not in execution plan")
	ErrInvalidTransition   = errors.New("invalid step status transition")
)

// New creates a new orchestrator instance with the specified stores, client,
// event hub, and configuration
func New(
	engine, flow *timebox.Store, client client.Client, hub timebox.EventHub,
	cfg *config.Config,
) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		engineExec: timebox.NewExecutor(
			engine, events.NewEngineState, events.EngineAppliers,
		),
		flowExec: timebox.NewExecutor(
			flow, events.NewFlowState, events.FlowAppliers,
		),
		stepClient: client,
		config:     cfg,
		ctx:        ctx,
		cancel:     cancel,
		consumer:   hub.NewConsumer(),
		retryQueue: NewRetryQueue(),
	}
	e.scripts = NewScriptRegistry()
	e.handler = CreateEventHandler(e)

	return e
}

func CreateEventHandler(e *Engine) timebox.Handler {
	const (
		flowStarted    = timebox.EventType(api.EventTypeFlowStarted)
		flowCompleted  = timebox.EventType(api.EventTypeFlowCompleted)
		flowFailed     = timebox.EventType(api.EventTypeFlowFailed)
		retryScheduled = timebox.EventType(api.EventTypeRetryScheduled)
		workSucceeded  = timebox.EventType(api.EventTypeWorkSucceeded)
	)

	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		flowStarted:    timebox.MakeHandler(e.handleFlowStarted),
		flowCompleted:  timebox.MakeHandler(e.handleFlowCompleted),
		flowFailed:     timebox.MakeHandler(e.handleFlowFailed),
		retryScheduled: timebox.MakeHandler(e.handleRetryScheduled),
		workSucceeded:  timebox.MakeHandler(e.handleWorkSucceeded),
	})
}

// Start begins processing flows and events
func (e *Engine) Start() {
	slog.Info("Engine starting")

	if err := e.RecoverFlows(); err != nil {
		slog.Error("Failed to recover flows",
			log.Error(err))
	}

	go e.eventLoop()
	go e.retryLoop()
}

// Stop gracefully shuts down the engine
func (e *Engine) Stop() error {
	e.cancel()
	e.retryQueue.Stop()
	defer e.consumer.Close()

	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.saveEngineSnapshot()
		slog.Info("Engine stopped")
		return nil
	case <-time.After(e.config.ShutdownTimeout):
		return ErrShutdownTimeout
	}
}

// StartFlow begins a new flow execution with the given plan and state
func (e *Engine) StartFlow(
	flowID api.FlowID, plan *api.ExecutionPlan, initState api.Args,
	meta api.Metadata,
) error {
	existing, err := e.GetFlowState(flowID)
	if err == nil && existing.ID != "" {
		return ErrFlowExists
	}

	if err := plan.ValidateInputs(initState); err != nil {
		return err
	}

	return e.raiseFlowEvent(flowID, api.EventTypeFlowStarted,
		api.FlowStartedEvent{
			FlowID:   flowID,
			Plan:     plan,
			Init:     initState,
			Metadata: meta,
		},
	)
}

// UnregisterStep removes a step from the engine registry
func (e *Engine) UnregisterStep(stepID api.StepID) error {
	return e.raiseEngineEvent(
		api.EventTypeStepUnregistered,
		api.StepUnregisteredEvent{StepID: stepID},
	)
}

// GetEngineState retrieves the current engine state including registered steps
// and active flows
func (e *Engine) GetEngineState() (*api.EngineState, error) {
	state, _, err := e.GetEngineStateSeq()
	return state, err
}

// GetEngineStateSeq retrieves the current engine state and next sequence
func (e *Engine) GetEngineStateSeq() (*api.EngineState, int64, error) {
	var nextSeq int64
	state, err := e.execEngine(
		func(st *api.EngineState, ag *Aggregator) error {
			nextSeq = ag.NextSequence()
			return nil
		},
	)
	return state, nextSeq, err
}

// ListSteps returns all currently registered steps in the engine
func (e *Engine) ListSteps() ([]*api.Step, error) {
	engState, err := e.GetEngineState()
	if err != nil {
		return nil, err
	}

	var steps []*api.Step
	for _, step := range engState.Steps {
		steps = append(steps, step)
	}

	return steps, nil
}

func (e *Engine) eventLoop() {
	for {
		select {
		case <-e.ctx.Done():
			return

		case event, ok := <-e.consumer.Receive():
			if !ok {
				return
			}
			e.routeEvent(event)
		}
	}
}

func (e *Engine) routeEvent(event *timebox.Event) {
	if err := e.handler(event); err != nil {
		slog.Error("Failed to handle flow lifecycle event",
			slog.String("event_type", string(event.Type)),
			log.Error(err))
	}

	if !events.IsFlowEvent(event) {
		return
	}

	flowID := api.FlowID(event.AggregateID[1])

	actor, loaded := e.flows.Load(flowID)
	if !loaded {
		wa := &flowActor{
			Engine: e,
			flowID: flowID,
			events: make(chan *timebox.Event, 100),
		}
		actor, loaded = e.flows.LoadOrStore(flowID, wa)
		if !loaded {
			e.wg.Add(1)
			go wa.run()
		}
	}

	actor.(*flowActor).events <- event
}

func (e *Engine) saveEngineSnapshot() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.engineExec.SaveSnapshot(ctx, events.EngineID); err != nil {
		slog.Error("Failed to save engine snapshot",
			log.Error(err))
		return
	}
	slog.Info("Engine snapshot saved")
}

func (e *Engine) handleFlowStarted(
	_ *timebox.Event, data api.FlowStartedEvent,
) error {
	return e.raiseEngineEvent(
		api.EventTypeFlowActivated,
		api.FlowActivatedEvent{FlowID: data.FlowID},
	)
}

func (e *Engine) handleFlowCompleted(
	_ *timebox.Event, data api.FlowCompletedEvent,
) error {
	e.retryQueue.RemoveFlow(data.FlowID)
	return nil
}

func (e *Engine) handleFlowFailed(
	_ *timebox.Event, data api.FlowFailedEvent,
) error {
	e.retryQueue.RemoveFlow(data.FlowID)
	return nil
}

func (e *Engine) handleRetryScheduled(
	_ *timebox.Event, data api.RetryScheduledEvent,
) error {
	e.retryQueue.Push(&RetryItem{
		FlowID:      data.FlowID,
		StepID:      data.StepID,
		Token:       data.Token,
		NextRetryAt: data.NextRetryAt,
	})
	return nil
}

func (e *Engine) handleWorkSucceeded(
	_ *timebox.Event, data api.WorkSucceededEvent,
) error {
	e.retryQueue.Remove(data.FlowID, data.StepID, data.Token)
	return nil
}

func (e *Engine) raiseEngineEvent(eventType api.EventType, data any) error {
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		return events.Raise(ag, eventType, data)
	}
	_, err := e.execEngine(cmd)
	return err
}

func (e *Engine) execEngine(
	cmd timebox.Command[*api.EngineState],
) (*api.EngineState, error) {
	return e.engineExec.Exec(e.ctx, events.EngineID, cmd)
}
