package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine/event"
	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/internal/engine/memo"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// Engine is the core flow execution engine
	Engine struct {
		stepClient  client.Client
		ctx         context.Context
		catalogExec *CatalogExecutor
		partExec    *PartitionExecutor
		flowExec    *FlowExecutor
		config      *config.Config
		cancel      context.CancelFunc
		scripts     *script.Registry
		mapper      *Mapper
		memoCache   *memo.Cache
		eventQueue  *event.Queue
		scheduler   *scheduler.Scheduler
		clock       scheduler.Clock
	}

	// Dependencies groups the external dependencies required by Engine
	Dependencies struct {
		CatalogStore     *timebox.Store
		PartitionStore   *timebox.Store
		FlowStore        *timebox.Store
		StepClient       client.Client
		Clock            scheduler.Clock
		TimerConstructor scheduler.TimerConstructor
	}

	// CatalogExecutor manages catalog state persistence and event sourcing
	CatalogExecutor = timebox.Executor[*api.CatalogState]

	// CatalogAggregator aggregates catalog state from events
	CatalogAggregator = timebox.Aggregator[*api.CatalogState]

	// PartitionExecutor manages partition state persistence and event sourcing
	PartitionExecutor = timebox.Executor[*api.PartitionState]

	// PartitionAggregator aggregates partition state from events
	PartitionAggregator = timebox.Aggregator[*api.PartitionState]

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
	ErrInvalidConfig         = errors.New("invalid config")
	ErrMissingDependency     = errors.New("missing dependency")
	ErrRecoverFlows          = errors.New("failed to recover flows")
)

const defaultBatchSize = 128

// New creates a new orchestrator instance from configuration and dependencies
func New(cfg *config.Config, deps Dependencies) (*Engine, error) {
	cfg = cfg.WithWorkDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	if err := normalizeDependencies(&deps); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		catalogExec: timebox.NewExecutor(
			deps.CatalogStore,
			events.NewCatalogState,
			events.CatalogAppliers,
		),
		partExec: timebox.NewExecutor(
			deps.PartitionStore,
			events.NewPartitionState,
			events.PartitionAppliers,
		),
		flowExec: timebox.NewExecutor(
			deps.FlowStore,
			events.NewFlowState,
			events.FlowAppliers,
		),
		scripts:    script.NewRegistry(),
		stepClient: deps.StepClient,
		config:     cfg,
		ctx:        ctx,
		cancel:     cancel,
		memoCache:  memo.NewCache(cfg.MemoCacheSize),
		scheduler:  scheduler.New(deps.Clock, deps.TimerConstructor),
		clock:      deps.Clock,
	}
	e.eventQueue = event.NewQueue(e.raisePartitionEvents, defaultBatchSize)
	e.mapper = NewMapper(e)

	return e, nil
}

// Start begins processing flows and events
func (e *Engine) Start() error {
	slog.Info("Engine starting")

	e.eventQueue.Start()
	go e.scheduler.Run(e.ctx)

	if err := e.RecoverFlows(); err != nil {
		e.eventQueue.Cancel()
		return fmt.Errorf("%w: %w", ErrRecoverFlows, err)
	}

	return nil
}

// ScheduleTask schedules a function to run at the given time
func (e *Engine) ScheduleTask(
	path []string, at time.Time, fn scheduler.TaskFunc,
) {
	e.scheduler.Schedule(e.ctx, path, at, fn)
}

// CancelTask removes a scheduled task for the exact path
func (e *Engine) CancelTask(path []string) {
	e.scheduler.Cancel(e.ctx, path)
}

// CancelPrefixedTasks removes all scheduled tasks under the given prefix
func (e *Engine) CancelPrefixedTasks(prefix []string) {
	e.scheduler.CancelPrefix(e.ctx, prefix)
}

// Now returns the current wall time from Engine's configured clock
func (e *Engine) Now() time.Time {
	return e.clock()
}

// Stop gracefully shuts down the engine
func (e *Engine) Stop() error {
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
			if err := tx.prepareStep(stepID); err != nil {
				return err
			}
		}
		tx.OnSuccess(func(flow *api.FlowState) {
			tx.Engine.scheduleTimeouts(flow, tx.Now())
		})
		return nil
	})
}

// UnregisterStep removes a step from the engine registry
func (e *Engine) UnregisterStep(stepID api.StepID) error {
	return e.raiseCatalogEvent(
		api.EventTypeStepUnregistered,
		api.StepUnregisteredEvent{StepID: stepID},
	)
}

// GetCatalogState retrieves the current catalog state
func (e *Engine) GetCatalogState() (*api.CatalogState, error) {
	state, err := e.execCatalog(
		func(st *api.CatalogState, ag *CatalogAggregator) error {
			return nil
		},
	)
	return state, err
}

// GetPartitionState retrieves the current partition state
func (e *Engine) GetPartitionState() (*api.PartitionState, error) {
	state, err := e.execPartition(
		func(st *api.PartitionState, ag *PartitionAggregator) error {
			return nil
		},
	)
	return state, err
}

// GetCatalogStateSeq retrieves catalog state and its next event sequence
func (e *Engine) GetCatalogStateSeq() (*api.CatalogState, int64, error) {
	var seq int64
	state, err := e.execCatalog(
		func(st *api.CatalogState, ag *CatalogAggregator) error {
			seq = ag.NextSequence()
			return nil
		},
	)
	return state, seq, err
}

// GetPartitionStateSeq retrieves partition state and its next event sequence
func (e *Engine) GetPartitionStateSeq() (*api.PartitionState, int64, error) {
	var seq int64
	state, err := e.execPartition(
		func(st *api.PartitionState, ag *PartitionAggregator) error {
			seq = ag.NextSequence()
			return nil
		},
	)
	return state, seq, err
}

// ListSteps returns all currently registered steps in the engine
func (e *Engine) ListSteps() ([]*api.Step, error) {
	catState, err := e.GetCatalogState()
	if err != nil {
		return nil, err
	}

	var steps []*api.Step
	for _, step := range catState.Steps {
		steps = append(steps, step)
	}

	return steps, nil
}

// EnqueueEvent schedules a partition aggregate event for sequential processing
func (e *Engine) EnqueueEvent(typ api.EventType, data any) {
	e.eventQueue.Enqueue(typ, data)
}

func (e *Engine) saveEngineSnapshot() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.catalogExec.SaveSnapshot(ctx, events.CatalogKey); err != nil {
		slog.Error("Failed to save catalog snapshot", log.Error(err))
	} else {
		slog.Info("Catalog snapshot saved")
	}

	if err := e.partExec.SaveSnapshot(ctx, events.PartitionKey); err != nil {
		slog.Error("Failed to save partition snapshot", log.Error(err))
	} else {
		slog.Info("Partition snapshot saved")
	}
}

func (e *Engine) raiseCatalogEvent(typ api.EventType, data any) error {
	_, err := e.execCatalog(
		func(st *api.CatalogState, ag *CatalogAggregator) error {
			return events.Raise(ag, typ, data)
		},
	)
	return err
}

func (e *Engine) raisePartitionEvent(typ api.EventType, data any) error {
	return e.raisePartitionEvents([]event.Event{{
		Type: typ,
		Data: data,
	}})
}

func (e *Engine) raisePartitionEvents(evs []event.Event) error {
	_, err := e.execPartition(
		func(st *api.PartitionState, ag *PartitionAggregator) error {
			for _, ev := range evs {
				if err := events.Raise(ag, ev.Type, ev.Data); err != nil {
					return err
				}
			}
			return nil
		},
	)
	return err
}

func (e *Engine) execCatalog(
	cmd timebox.Command[*api.CatalogState],
) (*api.CatalogState, error) {
	return e.catalogExec.Exec(e.ctx, events.CatalogKey, cmd)
}

func (e *Engine) execPartition(
	cmd timebox.Command[*api.PartitionState],
) (*api.PartitionState, error) {
	return e.partExec.Exec(e.ctx, events.PartitionKey, cmd)
}

func normalizeDependencies(deps *Dependencies) error {
	if deps.CatalogStore == nil {
		return fmt.Errorf("%w: catalog store", ErrMissingDependency)
	}
	if deps.PartitionStore == nil {
		return fmt.Errorf("%w: partition store", ErrMissingDependency)
	}
	if deps.FlowStore == nil {
		return fmt.Errorf("%w: flow store", ErrMissingDependency)
	}
	if deps.StepClient == nil {
		return fmt.Errorf("%w: step client", ErrMissingDependency)
	}
	if deps.Clock == nil {
		deps.Clock = time.Now
	}
	if deps.TimerConstructor == nil {
		deps.TimerConstructor = scheduler.NewTimer
	}
	return nil
}
