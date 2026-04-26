package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine/memo"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type (
	// Engine is the core flow execution engine
	Engine struct {
		stepClient  client.Client
		ctx         context.Context
		catalogExec *CatalogExecutor
		clusterExec *ClusterExecutor
		flowExec    *FlowExecutor
		engStore    *timebox.Store
		config      *config.Config
		cancel      context.CancelFunc
		scripts     *script.Registry
		mapper      *Mapper
		memoCache   *memo.Cache
		scheduler   *scheduler.Scheduler
		clock       scheduler.Clock
		eventHub    *event.Hub
		healthMu    sync.RWMutex
		health      map[api.StepID]api.HealthState
	}

	// Dependencies groups the external dependencies required by Engine
	Dependencies struct {
		EngineStore      *timebox.Store
		FlowStore        *timebox.Store
		StepClient       client.Client
		Clock            scheduler.Clock
		TimerConstructor scheduler.TimerConstructor
		EventHub         *event.Hub
	}

	// CatalogExecutor manages catalog state persistence and event sourcing
	CatalogExecutor = timebox.Executor[api.CatalogState]

	// CatalogAggregator aggregates catalog state from events
	CatalogAggregator = timebox.Aggregator[api.CatalogState]

	// ClusterExecutor manages cluster state persistence and event sourcing
	ClusterExecutor = timebox.Executor[api.ClusterState]

	// ClusterAggregator aggregates cluster state from events
	ClusterAggregator = timebox.Aggregator[api.ClusterState]

	// FlowExecutor manages flow state persistence and event sourcing
	FlowExecutor = timebox.Executor[api.FlowState]

	// FlowAggregator aggregates flow state from events
	FlowAggregator = timebox.Aggregator[api.FlowState]
)

var (
	ErrInvalidConfig     = errors.New("invalid config")
	ErrMissingDependency = errors.New("missing dependency")
)

// New creates a new orchestrator instance from configuration and dependencies
func New(cfg *config.Config, deps Dependencies) (*Engine, error) {
	cfg = cfg.WithWorkDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, errors.Join(ErrInvalidConfig, err)
	}

	if err := normalizeDependencies(&deps); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		catalogExec: timebox.NewExecutor(
			deps.EngineStore, events.NewCatalogState, events.CatalogAppliers,
		),
		clusterExec: timebox.NewExecutor(
			deps.EngineStore, events.NewClusterState, events.ClusterAppliers,
		),
		flowExec: timebox.NewExecutor(
			deps.FlowStore, events.NewFlowState, events.FlowAppliers,
		),
		scripts:    script.NewRegistry(),
		stepClient: deps.StepClient,
		engStore:   deps.EngineStore,
		config:     cfg,
		ctx:        ctx,
		cancel:     cancel,
		memoCache:  memo.NewCache(cfg.MemoCacheSize),
		scheduler:  scheduler.New(deps.Clock, deps.TimerConstructor),
		clock:      deps.Clock,
		eventHub:   deps.EventHub,
		health:     map[api.StepID]api.HealthState{},
	}
	e.mapper = NewMapper(e)

	return e, nil
}

// LocalNodeID returns the node ID of this engine instance
func (e *Engine) LocalNodeID() api.NodeID {
	return api.NodeID(e.config.Raft.LocalID)
}

// GetEventHub exposes the engine's in-process event hub
func (e *Engine) GetEventHub() *event.Hub {
	return e.eventHub
}

func normalizeDependencies(deps *Dependencies) error {
	if deps.EngineStore == nil {
		return fmt.Errorf("%w: engine store", ErrMissingDependency)
	}
	if deps.FlowStore == nil {
		return fmt.Errorf("%w: flow store", ErrMissingDependency)
	}
	if deps.StepClient == nil {
		return fmt.Errorf("%w: step client", ErrMissingDependency)
	}
	if deps.EventHub == nil {
		return fmt.Errorf("%w: event hub", ErrMissingDependency)
	}
	if deps.Clock == nil {
		deps.Clock = time.Now
	}
	if deps.TimerConstructor == nil {
		deps.TimerConstructor = scheduler.NewTimer
	}
	return nil
}
