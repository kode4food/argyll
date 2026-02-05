package engine

import (
	"context"
	"errors"
	"log/slog"
	"time"

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
		engineExec *Executor
		flowExec   *FlowExecutor
		config     *config.Config
		cancel     context.CancelFunc
		scripts    *ScriptRegistry
		retryQueue *RetryQueue
		memoCache  *MemoCache
		eventHub   *timebox.EventHub
	}

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
	ErrFlowNotFound          = errors.New("flow not found")
	ErrFlowExists            = errors.New("flow exists")
	ErrStepNotFound          = errors.New("step not found")
	ErrStepExists            = errors.New("step exists")
	ErrScriptCompileFailed   = errors.New("failed to compile scripts for plan")
	ErrStepNotInPlan         = errors.New("step not in execution plan")
	ErrWorkItemNotFound      = errors.New("work item not found")
	ErrInvalidWorkTransition = errors.New("invalid work state transition")
)

// New creates a new orchestrator instance with the specified stores, client,
// event hub, and configuration
func New(
	engine, flow *timebox.Store, client client.Client, hub *timebox.EventHub,
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
		retryQueue: NewRetryQueue(),
		memoCache:  NewMemoCache(cfg.MemoCacheSize),
		eventHub:   hub,
	}
	e.scripts = NewScriptRegistry()

	return e
}

// Start begins processing flows and events
func (e *Engine) Start() {
	slog.Info("Engine starting")
	go e.startProjection()

	if err := e.RecoverFlows(); err != nil {
		slog.Error("Failed to recover flows",
			log.Error(err))
	}

	go e.retryLoop()
}

// Stop gracefully shuts down the engine
func (e *Engine) Stop() error {
	e.cancel()
	e.retryQueue.Stop()
	e.saveEngineSnapshot()
	slog.Info("Engine stopped")
	return nil
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

	return e.flowTx(flowID, func(tx *flowTx) error {
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowStarted,
			api.FlowStartedEvent{
				FlowID:   flowID,
				Plan:     plan,
				Init:     initState,
				Metadata: meta,
			},
		); err != nil {
			return err
		}
		parentID, _ := api.GetMetaString[api.FlowID](
			meta, api.MetaParentFlowID,
		)
		if err := events.Raise(tx.FlowAggregator,
			api.EventTypeFlowActivated,
			api.FlowActivatedEvent{
				FlowID:       flowID,
				ParentFlowID: parentID,
			},
		); err != nil {
			return err
		}
		if flowTransitions.IsTerminal(tx.Value().Status) {
			return nil
		}

		for _, stepID := range tx.findInitialSteps(tx.Value()) {
			err := tx.prepareStep(stepID)
			if err != nil {
				slog.Warn("Failed to prepare step",
					log.StepID(stepID),
					log.Error(err))
				continue
			}
		}
		return nil
	})
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

func (e *Engine) startProjection() {
	handlers := e.projectionHandlers()
	consumer := e.eventHub.NewAggregateConsumer(
		timebox.NewAggregateID(events.FlowPrefix),
		handlerEventTypes(handlers)...,
	)
	defer consumer.Close()
	dispatch := events.MakeDispatcher(handlers)

	for {
		select {
		case <-e.ctx.Done():
			return
		case ev, ok := <-consumer.Receive():
			if !ok {
				return
			}
			if ev == nil {
				continue
			}
			if err := dispatch(ev); err != nil {
				slog.Error("Engine projection failed",
					slog.String("event_type", string(ev.Type)),
					slog.String("aggregate_id", ev.AggregateID.Join("/")),
					log.Error(err))
			}
		}
	}
}

func (e *Engine) projectionHandlers() map[api.EventType]timebox.Handler {
	flowActivated := timebox.MakeHandler(e.handleFlowActivated)
	flowDeactivated := timebox.MakeHandler(e.handleFlowDeactivated)
	flowCompleted := timebox.MakeHandler(e.handleFlowCompleted)
	flowFailed := timebox.MakeHandler(e.handleFlowFailed)

	return map[api.EventType]timebox.Handler{
		api.EventTypeFlowActivated:   flowActivated,
		api.EventTypeFlowDeactivated: flowDeactivated,
		api.EventTypeFlowCompleted:   flowCompleted,
		api.EventTypeFlowFailed:      flowFailed,
	}
}

func (e *Engine) handleFlowActivated(
	_ *timebox.Event, data api.FlowActivatedEvent,
) error {
	return e.raiseEngineEvent(api.EventTypeFlowActivated, data)
}

func (e *Engine) handleFlowDeactivated(
	_ *timebox.Event, data api.FlowDeactivatedEvent,
) error {
	return e.raiseEngineEvent(api.EventTypeFlowDeactivated, data)
}

func (e *Engine) handleFlowCompleted(
	ev *timebox.Event, data api.FlowCompletedEvent,
) error {
	return e.raiseEngineEvent(
		api.EventTypeFlowDigestUpdated,
		api.FlowDigestUpdatedEvent{
			FlowID:      data.FlowID,
			Status:      api.FlowCompleted,
			CompletedAt: ev.Timestamp,
		},
	)
}

func (e *Engine) handleFlowFailed(
	ev *timebox.Event, data api.FlowFailedEvent,
) error {
	return e.raiseEngineEvent(
		api.EventTypeFlowDigestUpdated,
		api.FlowDigestUpdatedEvent{
			FlowID:      data.FlowID,
			Status:      api.FlowFailed,
			CompletedAt: ev.Timestamp,
			Error:       data.Error,
		},
	)
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

func (e *Engine) raiseEngineEvent(eventType api.EventType, data any) error {
	_, err := e.execEngine(
		func(st *api.EngineState, ag *Aggregator) error {
			return events.Raise(ag, eventType, data)
		},
	)
	return err
}

func (e *Engine) execEngine(
	cmd timebox.Command[*api.EngineState],
) (*api.EngineState, error) {
	return e.engineExec.Exec(e.ctx, events.EngineID, cmd)
}

func handlerEventTypes(
	handlers map[api.EventType]timebox.Handler,
) []timebox.EventType {
	eventTypes := make([]timebox.EventType, 0, len(handlers))
	for eventType := range handlers {
		eventTypes = append(eventTypes, timebox.EventType(eventType))
	}
	return eventTypes
}
