package engine

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
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
		mapper     *Mapper
		retryQueue *RetryQueue
		memoCache  *MemoCache
		eventQueue *EventQueue
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
	ErrInvalidFlowCursor     = errors.New("invalid flow cursor")
)

// New creates a new orchestrator instance with the specified stores, client,
// event hub, and configuration
func New(
	engine, flow *timebox.Store, client client.Client, cfg *config.Config,
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
	}
	e.eventQueue = NewEventQueue(e.raiseEngineEvent)
	e.scripts = NewScriptRegistry()
	e.mapper = NewMapper(e)

	return e
}

// Start begins processing flows and events
func (e *Engine) Start() {
	slog.Info("Engine starting")

	e.eventQueue.Start()

	if err := e.RecoverFlows(); err != nil {
		slog.Error("Failed to recover flows",
			log.Error(err))
	}

	go e.retryLoop()
}

// Stop gracefully shuts down the engine
func (e *Engine) Stop() error {
	e.retryQueue.Stop()
	e.eventQueue.Flush()
	e.cancel()
	e.saveEngineSnapshot()
	slog.Info("Engine stopped")
	return nil
}

// StartFlow begins a new flow execution with the given plan and options
func (e *Engine) StartFlow(
	flowID api.FlowID, plan *api.ExecutionPlan, apps ...flowopt.Applier,
) error {
	existing, err := e.GetFlowState(flowID)
	if err == nil && existing.ID != "" {
		return ErrFlowExists
	}

	opts := flowopt.DefaultOptions(apps...)
	if err := plan.ValidateInputs(opts.Init); err != nil {
		return err
	}

	return e.flowTx(flowID, func(tx *flowTx) error {
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowStarted,
			api.FlowStartedEvent{
				FlowID:   flowID,
				Plan:     plan,
				Init:     opts.Init,
				Metadata: opts.Metadata,
				Labels:   opts.Labels,
			},
		); err != nil {
			return err
		}
		parentID, _ := api.GetMetaString[api.FlowID](
			opts.Metadata, api.MetaParentFlowID,
		)
		tx.OnSuccess(func(*api.FlowState) {
			tx.EnqueueEvent(api.EventTypeFlowActivated,
				api.FlowActivatedEvent{
					FlowID:       flowID,
					ParentFlowID: parentID,
					Labels:       opts.Labels,
				},
			)
		})
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

// EnqueueEvent schedules an engine aggregate event for sequential processing
func (e *Engine) EnqueueEvent(typ api.EventType, data any) {
	e.eventQueue.Enqueue(typ, data)
}

func (e *Engine) saveEngineSnapshot() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.engineExec.SaveSnapshot(ctx, events.EngineKey); err != nil {
		slog.Error("Failed to save engine snapshot",
			log.Error(err))
		return
	}
	slog.Info("Engine snapshot saved")
}

func (e *Engine) raiseEngineEvent(typ api.EventType, data any) error {
	_, err := e.execEngine(
		func(st *api.EngineState, ag *Aggregator) error {
			return events.Raise(ag, typ, data)
		},
	)
	return err
}

func (e *Engine) execEngine(
	cmd timebox.Command[*api.EngineState],
) (*api.EngineState, error) {
	return e.engineExec.Exec(e.ctx, events.EngineKey, cmd)
}
