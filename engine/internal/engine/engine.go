package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine/memo"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/step"
)

type (
	// Engine is the core flow execution engine
	Engine struct {
		ctx         context.Context
		backend     timebox.Backend
		scripts     *script.Registry
		steps       *step.Registry
		mapper      *Mapper
		flowExec    *FlowExecutor
		engStore    *timebox.Store
		flowStore   *timebox.Store
		config      *config.Config
		cancel      context.CancelFunc
		catalogExec *CatalogExecutor
		clusterExec *ClusterExecutor
		memoCache   *memo.Cache
		scheduler   *scheduler.Scheduler
		clock       scheduler.Clock
		eventHub    *event.Hub
		health      map[api.StepID]api.HealthState
		healthMu    sync.RWMutex
	}

	// Dependencies groups the external dependencies required by Engine
	Dependencies struct {
		Scripts          *script.Registry
		Steps            *step.Registry
		Clock            scheduler.Clock
		TimerConstructor scheduler.TimerConstructor
		EventHub         *event.Hub
	}

	// OpenBackend opens the backend the engine runs on, wired to the publisher
	// its committed events must reach
	OpenBackend func(timebox.Publisher) (timebox.Backend, error)

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

// DefaultStoreReadyTimeout bounds how long New waits for the backend's stores
const DefaultStoreReadyTimeout = 5 * time.Second

var (
	ErrInvalidConfig     = errors.New("invalid config")
	ErrMissingDependency = errors.New("missing dependency")
	ErrOpenBackend       = errors.New("failed to open backend")
)

// New creates an engine from its configuration and dependencies, then opens
// the backend it runs on. The engine exists before the backend does, so no
// committed event can arrive ahead of the engine that must handle it
func New(
	cfg *config.Config, deps Dependencies, open OpenBackend,
) (*Engine, error) {
	cfg = cfg.WithWorkDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, errors.Join(ErrInvalidConfig, err)
	}

	if err := normalizeDependencies(&deps); err != nil {
		return nil, err
	}
	if open == nil {
		return nil, fmt.Errorf("%w: backend", ErrMissingDependency)
	}

	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		scripts:   deps.Scripts,
		steps:     deps.Steps,
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		memoCache: memo.NewCache(cfg.MemoCacheSize),
		scheduler: scheduler.New(deps.Clock, deps.TimerConstructor),
		clock:     deps.Clock,
		eventHub:  deps.EventHub,
		health:    map[api.StepID]api.HealthState{},
	}
	e.mapper = NewMapper(e)

	if err := e.openStores(open); err != nil {
		cancel()
		return nil, errors.Join(ErrOpenBackend, err)
	}
	return e, nil
}

// LocalNodeID returns the node ID of this engine instance
func (e *Engine) LocalNodeID() api.NodeID {
	return e.config.NodeID
}

// GetEventHub exposes the engine's in-process event hub
func (e *Engine) GetEventHub() *event.Hub {
	return e.eventHub
}

func (e *Engine) openStores(open OpenBackend) error {
	backend, err := open(e.publish)
	if err != nil {
		return err
	}
	engStore, err := timebox.NewStore(backend, e.config.EngineStoreConfig())
	if err != nil {
		return errors.Join(err, backend.Close())
	}
	flowStore, err := timebox.NewStore(backend, e.config.FlowStoreConfig())
	if err != nil {
		return errors.Join(err, backend.Close())
	}

	ctx, cancel := context.WithTimeout(e.ctx, DefaultStoreReadyTimeout)
	defer cancel()
	if err := flowStore.WaitReady(ctx); err != nil {
		return errors.Join(err, backend.Close())
	}

	e.backend = backend
	e.engStore = engStore
	e.flowStore = flowStore
	e.catalogExec = engStore.Executor(
		events.NewCatalogState, events.CatalogAppliers,
	)
	e.clusterExec = engStore.Executor(
		events.NewClusterState, events.ClusterAppliers,
	)
	e.flowExec = flowStore.Executor(events.NewFlowState, events.FlowAppliers)
	return nil
}

// publish hands committed events to the engine and then to subscribers
func (e *Engine) publish(evs ...*timebox.Event) {
	e.HandleCommitted(evs...)
	e.eventHub.Publish(evs...)
}

func normalizeDependencies(deps *Dependencies) error {
	if deps.Scripts == nil {
		return fmt.Errorf("%w: script registry", ErrMissingDependency)
	}
	if deps.Steps == nil {
		return fmt.Errorf("%w: step registry", ErrMissingDependency)
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
