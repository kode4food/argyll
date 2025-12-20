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
	"github.com/kode4food/argyll/engine/internal/events"
	"github.com/kode4food/argyll/engine/pkg/api"
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
		scripts:    NewScriptRegistry(),
	}
	e.handler = e.createEventHandler()
	return e
}

func (e *Engine) createEventHandler() timebox.Handler {
	const (
		flowStarted   = timebox.EventType(api.EventTypeFlowStarted)
		flowCompleted = timebox.EventType(api.EventTypeFlowCompleted)
		flowFailed    = timebox.EventType(api.EventTypeFlowFailed)
	)

	return timebox.MakeDispatcher(map[timebox.EventType]timebox.Handler{
		flowStarted:   timebox.MakeHandler(e.handleFlowStarted),
		flowCompleted: timebox.MakeHandler(e.handleFlowCompleted),
		flowFailed:    timebox.MakeHandler(e.handleFlowFailed),
	})
}

// Start begins processing flows and events
func (e *Engine) Start() {
	slog.Info("Engine starting")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.RecoverFlows(ctx); err != nil {
		slog.Error("Failed to recover flows",
			log.Error(err))
	}

	go e.eventLoop()
	go e.retryLoop()
}

// Stop gracefully shuts down the engine
func (e *Engine) Stop() error {
	e.cancel()
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
	ctx context.Context, flowID api.FlowID, plan *api.ExecutionPlan,
	initState api.Args, meta api.Metadata,
) error {
	existing, err := e.GetFlowState(ctx, flowID)
	if err == nil && existing.ID != "" {
		return ErrFlowExists
	}

	if err := plan.ValidateInputs(initState); err != nil {
		return err
	}

	return e.raiseFlowEvent(ctx, flowID, api.EventTypeFlowStarted,
		api.FlowStartedEvent{
			FlowID:   flowID,
			Plan:     plan,
			Init:     initState,
			Metadata: meta,
		})
}

// UnregisterStep removes a step from the engine registry
func (e *Engine) UnregisterStep(ctx context.Context, stepID api.StepID) error {
	return e.raiseEngineEvent(ctx, api.EventTypeStepUnregistered,
		api.StepUnregisteredEvent{StepID: stepID})
}

// GetEngineState retrieves the current engine state including registered steps
// and active flows
func (e *Engine) GetEngineState(ctx context.Context) (*api.EngineState, error) {
	return e.engineExec.Exec(ctx, events.EngineID,
		func(st *api.EngineState, ag *Aggregator) error {
			return nil
		},
	)
}

// ListSteps returns all currently registered steps in the engine
func (e *Engine) ListSteps(ctx context.Context) ([]*api.Step, error) {
	engState, err := e.GetEngineState(ctx)
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
	return e.raiseEngineEvent(context.Background(),
		api.EventTypeFlowActivated,
		api.FlowActivatedEvent{FlowID: data.FlowID})
}

func (e *Engine) handleFlowCompleted(
	_ *timebox.Event, data api.FlowCompletedEvent,
) error {
	return e.raiseEngineEvent(context.Background(),
		api.EventTypeFlowDeactivated,
		api.FlowDeactivatedEvent{FlowID: data.FlowID})
}

func (e *Engine) handleFlowFailed(
	_ *timebox.Event, data api.FlowFailedEvent,
) error {
	return e.raiseEngineEvent(context.Background(),
		api.EventTypeFlowDeactivated,
		api.FlowDeactivatedEvent{FlowID: data.FlowID})
}

func (e *Engine) raiseEngineEvent(
	ctx context.Context, eventType api.EventType, data any,
) error {
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		return events.Raise(ag, eventType, data)
	}
	_, err := e.engineExec.Exec(ctx, events.EngineID, cmd)
	return err
}
