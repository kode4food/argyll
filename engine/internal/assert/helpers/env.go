package helpers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
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
		EventHub       *timebox.EventHub
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
	return NewTestEngineWithTime(t, time.Now, engine.NewTimer)
}

// NewTestEngineWithTime creates a test engine with explicit time wiring
func NewTestEngineWithTime(
	t *testing.T, clock engine.Clock, makeTimer engine.TimerConstructor,
) *TestEngineEnv {
	t.Helper()

	server, err := miniredis.Run()
	assert.NoError(t, err)

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)

	catCfg := config.NewDefaultConfig().CatalogStore
	catCfg.Addr = server.Addr()
	catCfg.Prefix = "test-catalog"

	catStore, err := tb.NewStore(catCfg)
	assert.NoError(t, err)

	partCfg := config.NewDefaultConfig().PartitionStore
	partCfg.Addr = server.Addr()
	partCfg.Prefix = "test-partition"

	partStore, err := tb.NewStore(partCfg)
	assert.NoError(t, err)

	flowConfig := config.NewDefaultConfig().FlowStore
	flowConfig.Addr = server.Addr()
	flowConfig.Prefix = "test-flow"

	flowStore, err := tb.NewStore(flowConfig)
	assert.NoError(t, err)

	flowExec := timebox.NewExecutor(
		flowStore, events.NewFlowState, events.FlowAppliers,
	)

	mockCli := NewMockClient()

	cfg := &config.Config{
		APIPort:         8080,
		APIHost:         "localhost",
		WebhookBaseURL:  "http://localhost:8080",
		StepTimeout:     5 * api.Second,
		FlowCacheSize:   100,
		MemoCacheSize:   100,
		ShutdownTimeout: 2 * time.Second,
		Work: api.WorkConfig{
			MaxRetries:  3,
			InitBackoff: 1000,
			MaxBackoff:  60000,
			BackoffType: api.BackoffTypeExponential,
		},
	}

	hub := tb.GetHub()
	eng, err := engine.New(cfg, engine.Dependencies{
		CatalogStore:     catStore,
		PartitionStore:   partStore,
		FlowStore:        flowStore,
		StepClient:       mockCli,
		Clock:            clock,
		TimerConstructor: makeTimer,
	})
	assert.NoError(t, err)

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
		_ = tb.Close()
		testEnv.Redis.Close()
	}

	return testEnv
}

// NewEngineInstance creates a new engine instance sharing the same stores and
// mock client. Used to simulate process restart after crash
func (e *TestEngineEnv) NewEngineInstance() (*engine.Engine, error) {
	return engine.New(e.Config, e.Dependencies())
}

// Dependencies returns a valid dependency bundle for constructing an engine
func (e *TestEngineEnv) Dependencies() engine.Dependencies {
	return e.engineDeps(time.Now, engine.NewTimer)
}

// RaiseFlowEvents appends flow events via the executor
func (e *TestEngineEnv) RaiseFlowEvents(
	flowID api.FlowID, evs ...FlowEvent,
) error {
	_, err := e.flowExec.Exec(
		context.Background(), events.FlowKey(flowID),
		func(st *api.FlowState, ag *timebox.Aggregator[*api.FlowState]) error {
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

// WithTestEnv creates a test engine environment, executes the provided
// function with it, and ensures cleanup happens automatically
func WithTestEnv(t *testing.T, fn func(*TestEngineEnv)) {
	t.Helper()
	testEnv := NewTestEngine(t)
	defer testEnv.Cleanup()
	fn(testEnv)
}

// WithTestEnvWithTime creates a test engine environment with explicit time
// wiring and ensures cleanup happens automatically
func WithTestEnvWithTime(
	t *testing.T, clock engine.Clock, makeTimer engine.TimerConstructor,
	fn func(*TestEngineEnv),
) {
	t.Helper()
	testEnv := NewTestEngineWithTime(t, clock, makeTimer)
	defer testEnv.Cleanup()
	fn(testEnv)
}

// WithEngine creates a test engine, executes the provided function with it,
// and ensures cleanup happens automatically
func WithEngine(t *testing.T, fn func(*engine.Engine)) {
	t.Helper()
	WithTestEnv(t, func(env *TestEngineEnv) {
		fn(env.Engine)
	})
}

// WithEngineWithTime creates a test engine with explicit time wiring and
// ensures cleanup happens automatically
func WithEngineWithTime(
	t *testing.T, clock engine.Clock, makeTimer engine.TimerConstructor,
	fn func(*engine.Engine),
) {
	t.Helper()
	WithTestEnvWithTime(t, clock, makeTimer, func(env *TestEngineEnv) {
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
	clock engine.Clock, makeTimer engine.TimerConstructor,
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
