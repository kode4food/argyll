package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine/event"
	"github.com/kode4food/argyll/engine/internal/engine/memo"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
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
