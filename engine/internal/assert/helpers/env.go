package helpers

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/redis"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/scheduler"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type (
	// TestEngineEnv holds all the components needed for engine testing
	TestEngineEnv struct {
		T              *testing.T
		Engine         *engine.Engine
		Redis          *miniredis.Miniredis
		MockClient     *MockClient
		Config         *config.Config
		EventHub       *event.Hub
		Cleanup        func()
		catalogStore   *timebox.Store
		partitionStore *timebox.Store
		flowStore      *timebox.Store
		flowExec       *timebox.Executor[*api.FlowState]
	}

	FlowEvent struct {
		Type api.EventType
		Data any
	}
)

// NewTestConfig creates a default configuration with debug logging enabled
func NewTestConfig() *config.Config {
	cfg := config.NewDefaultConfig()
	cfg.LogLevel = "debug"
	return cfg
}

// NewTestEngine creates a fully configured test engine environment with an
// in-memory Redis backend and mock HTTP client
func NewTestEngine(t *testing.T) *TestEngineEnv {
	return NewTestEngineWithDeps(t, engine.Dependencies{})
}

// NewTestEngineWithDeps creates a test engine with dependency overrides
func NewTestEngineWithDeps(
	t *testing.T, overrides engine.Dependencies,
) *TestEngineEnv {
	t.Helper()

	server, err := miniredis.Run()
	assert.NoError(t, err)

	cfg := &config.Config{
		APIPort:         8080,
		APIHost:         "localhost",
		WebhookBaseURL:  "http://localhost:8080",
		StepTimeout:     5 * api.Second,
		MemoCacheSize:   100,
		ShutdownTimeout: 2 * time.Second,
		Work: api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 1000,
			MaxBackoff:  60000,
			BackoffType: api.BackoffTypeExponential,
		},
	}

	base := config.NewDefaultConfig()

	catStore, err := redis.NewStore(base.CatalogStore.With(redis.Config{
		Addr:   server.Addr(),
		Prefix: "test-catalog",
	}))
	assert.NoError(t, err)

	partStore, err := redis.NewStore(base.PartitionStore.With(redis.Config{
		Addr:   server.Addr(),
		Prefix: "test-partition",
	}))
	assert.NoError(t, err)

	flowStore, err := redis.NewStore(base.FlowStore.With(redis.Config{
		Addr:   server.Addr(),
		Prefix: "test-flow",
		Timebox: timebox.Config{
			CacheSize: 100,
		},
	}))
	assert.NoError(t, err)

	mockCli := NewMockClient()

	defaultDeps := engine.Dependencies{
		CatalogStore:     catStore,
		PartitionStore:   partStore,
		FlowStore:        flowStore,
		StepClient:       mockCli,
		Clock:            time.Now,
		TimerConstructor: scheduler.NewTimer,
	}
	deps := mergeDependencies(defaultDeps, overrides)
	eng, err := engine.New(cfg, deps)
	assert.NoError(t, err)
	if cl, ok := deps.StepClient.(*MockClient); ok {
		mockCli = cl
	}
	hub := eng.GetEventHub()
	flowExec := timebox.NewExecutor(
		flowStore,
		events.NewFlowState,
		events.FlowAppliers,
		func(_ *api.FlowState, evs []*timebox.Event) {
			hub.Publish(evs...)
		},
	)

	testEnv := &TestEngineEnv{
		T:              t,
		Engine:         eng,
		Redis:          server,
		MockClient:     mockCli,
		Config:         cfg,
		EventHub:       hub,
		catalogStore:   catStore,
		partitionStore: partStore,
		flowStore:      flowStore,
		flowExec:       flowExec,
	}

	testEnv.Cleanup = func() {
		_ = testEnv.Engine.Stop()
		_ = testEnv.flowStore.Close()
		_ = testEnv.partitionStore.Close()
		_ = testEnv.catalogStore.Close()
		testEnv.Redis.Close()
	}

	return testEnv
}

// NewEngineInstance creates a new engine instance sharing the same stores and
// mock client. Used to simulate process restart after crash
func (e *TestEngineEnv) NewEngineInstance() (*engine.Engine, error) {
	eng, err := engine.New(e.Config, e.Dependencies())
	if err != nil {
		return nil, err
	}
	e.EventHub = eng.GetEventHub()
	e.flowExec = timebox.NewExecutor(
		e.flowStore,
		events.NewFlowState,
		events.FlowAppliers,
		func(_ *api.FlowState, evs []*timebox.Event) {
			e.EventHub.Publish(evs...)
		},
	)
	return eng, nil
}

// Dependencies returns a valid dependency bundle for constructing an engine
func (e *TestEngineEnv) Dependencies() engine.Dependencies {
	return e.engineDeps(time.Now, scheduler.NewTimer)
}

// RaiseFlowEvents appends flow events via the executor
func (e *TestEngineEnv) RaiseFlowEvents(
	flowID api.FlowID, evs ...FlowEvent,
) error {
	_, err := e.flowExec.Exec(
		events.FlowKey(flowID),
		func(st *api.FlowState, ag *timebox.Aggregator[*api.FlowState]) error {
			for _, ev := range evs {
				if err := raiseFlowEvent(ag, ev); err != nil {
					return err
				}
			}
			return nil
		},
	)
	return err
}

// ListFlowsByLabel returns the flow aggregate IDs currently indexed for the
// given label/value pair.
func (e *TestEngineEnv) ListFlowsByLabel(
	label, value string,
) ([]timebox.AggregateID, error) {
	return e.flowStore.ListAggregatesByLabel(label, value)
}

func raiseFlowEvent(
	ag *timebox.Aggregator[*api.FlowState], ev FlowEvent,
) error {
	return events.Raise(ag, ev.Type, ev.Data)
}

// WithTestEnv creates a test engine environment, executes the provided
// function with it, and ensures cleanup happens automatically
func WithTestEnv(t *testing.T, fn func(*TestEngineEnv)) {
	WithTestEnvDeps(t, engine.Dependencies{}, fn)
}

// WithTestEnvDeps creates a test engine environment with dependency
// overrides and ensures cleanup happens automatically
func WithTestEnvDeps(
	t *testing.T, overrides engine.Dependencies, fn func(*TestEngineEnv),
) {
	t.Helper()
	testEnv := NewTestEngineWithDeps(t, overrides)
	defer testEnv.Cleanup()
	fn(testEnv)
}

// WithEngine creates a test engine, executes the provided function with it,
// and ensures cleanup happens automatically
func WithEngine(t *testing.T, fn func(*engine.Engine)) {
	WithEngineDeps(t, engine.Dependencies{}, fn)
}

// WithEngineDeps creates a test engine with dependency overrides and
// ensures cleanup happens automatically
func WithEngineDeps(
	t *testing.T, overrides engine.Dependencies, fn func(*engine.Engine),
) {
	t.Helper()
	WithTestEnvDeps(t, overrides, func(env *TestEngineEnv) {
		fn(env.Engine)
	})
}

// WithStartedEngine creates a test engine, starts it, executes the provided
// function with the engine, and ensures cleanup happens automatically
func WithStartedEngine(t *testing.T, fn func(*engine.Engine)) {
	t.Helper()
	WithEngine(t, func(eng *engine.Engine) {
		assert.NoError(t, eng.Start())
		fn(eng)
	})
}

func (e *TestEngineEnv) engineDeps(
	clock scheduler.Clock, makeTimer scheduler.TimerConstructor,
) engine.Dependencies {
	return engine.Dependencies{
		CatalogStore:     e.catalogStore,
		PartitionStore:   e.partitionStore,
		FlowStore:        e.flowStore,
		StepClient:       e.MockClient,
		Clock:            clock,
		TimerConstructor: makeTimer,
	}
}

func mergeDependencies(
	defaults engine.Dependencies, overrides engine.Dependencies,
) engine.Dependencies {
	if overrides.CatalogStore != nil {
		defaults.CatalogStore = overrides.CatalogStore
	}
	if overrides.PartitionStore != nil {
		defaults.PartitionStore = overrides.PartitionStore
	}
	if overrides.FlowStore != nil {
		defaults.FlowStore = overrides.FlowStore
	}
	if overrides.StepClient != nil {
		defaults.StepClient = overrides.StepClient
	}
	if overrides.Clock != nil {
		defaults.Clock = overrides.Clock
	}
	if overrides.TimerConstructor != nil {
		defaults.TimerConstructor = overrides.TimerConstructor
	}
	return defaults
}
