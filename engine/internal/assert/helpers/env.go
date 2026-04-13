package helpers

import (
	"testing"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/memory"
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
		T          *testing.T
		Engine     *engine.Engine
		MockClient *MockClient
		Config     *config.Config
		EventHub   *event.Hub
		Cleanup    func()
		engStore   *timebox.Store
		flowStore  *timebox.Store
		flowExec   *timebox.Executor[api.FlowState]
	}

	FlowEvent struct {
		Type api.EventType
		Data any
	}

	publishingBackend struct {
		timebox.Backend
		publish func(...*timebox.Event)
	}
)

// NewTestConfig creates a default configuration with debug logging enabled
func NewTestConfig() *config.Config {
	cfg := config.NewDefaultConfig()
	cfg.LogLevel = "debug"
	return cfg
}

// NewTestEngine creates a fully configured test engine environment with an
// in-memory Timebox backend and mock HTTP client
func NewTestEngine(t *testing.T) *TestEngineEnv {
	return NewTestEngineWithDeps(t, engine.Dependencies{})
}

// NewTestEngineWithDeps creates a test engine with dependency overrides
func NewTestEngineWithDeps(
	t *testing.T, overrides engine.Dependencies,
) *TestEngineEnv {
	t.Helper()

	cfg := NewTestConfig()
	cfg.APIPort = 8080
	cfg.APIHost = "localhost"
	cfg.WebhookBaseURL = "http://localhost:8080"
	cfg.StepTimeout = 5 * api.Second
	cfg.MemoCacheSize = 100
	cfg.ShutdownTimeout = 2 * time.Second
	cfg.Work = api.WorkConfig{
		MaxRetries:  3,
		InitBackoff: 1000,
		MaxBackoff:  60000,
		BackoffType: api.BackoffTypeExponential,
	}

	mockCli := NewMockClient()
	hub := event.NewHub()
	backend := publishingBackend{
		Backend: memory.NewPersistence(),
		publish: hub.Publish,
	}
	engStore, err := backend.NewStore(cfg.EngineStoreConfig())
	assert.NoError(t, err)
	flowStore, err := backend.NewStore(cfg.FlowStoreConfig())
	assert.NoError(t, err)

	defaultDeps := engine.Dependencies{
		EngineStore:      engStore,
		FlowStore:        flowStore,
		StepClient:       mockCli,
		Clock:            time.Now,
		TimerConstructor: scheduler.NewTimer,
		EventHub:         hub,
	}
	deps := mergeDependencies(defaultDeps, overrides)
	eng, err := engine.New(cfg, deps)
	assert.NoError(t, err)
	if cl, ok := deps.StepClient.(*MockClient); ok {
		mockCli = cl
	}
	flowExec := timebox.NewExecutor(
		deps.FlowStore, events.NewFlowState, events.FlowAppliers,
	)

	testEnv := &TestEngineEnv{
		T:          t,
		Engine:     eng,
		MockClient: mockCli,
		Config:     cfg,
		EventHub:   deps.EventHub,
		engStore:   deps.EngineStore,
		flowStore:  deps.FlowStore,
		flowExec:   flowExec,
	}

	testEnv.Cleanup = func() {
		_ = testEnv.Engine.Stop()
		_ = testEnv.engStore.Close()
		_ = testEnv.flowStore.Close()
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
	e.flowExec = timebox.NewExecutor(
		e.flowStore, events.NewFlowState, events.FlowAppliers,
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
		func(_ api.FlowState, ag *timebox.Aggregator[api.FlowState]) error {
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

// AppendEvents appends raw events to the shared test store
func (e *TestEngineEnv) AppendEvents(
	id timebox.AggregateID, atSeq int64, evs ...*timebox.Event,
) error {
	return e.flowStore.AppendEvents(id, atSeq, evs)
}

// ListFlowsByLabel returns the flow aggregate IDs currently indexed for the
// given label/value pair
func (e *TestEngineEnv) ListFlowsByLabel(
	label, value string,
) ([]timebox.AggregateID, error) {
	return e.flowStore.ListAggregatesByLabel(label, value)
}

func raiseFlowEvent(
	ag *timebox.Aggregator[api.FlowState], ev FlowEvent,
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
		EngineStore:      e.engStore,
		FlowStore:        e.flowStore,
		StepClient:       e.MockClient,
		Clock:            clock,
		TimerConstructor: makeTimer,
		EventHub:         e.EventHub,
	}
}

func mergeDependencies(
	defaults engine.Dependencies, overrides engine.Dependencies,
) engine.Dependencies {
	if overrides.EngineStore != nil {
		defaults.EngineStore = overrides.EngineStore
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
	if overrides.EventHub != nil {
		defaults.EventHub = overrides.EventHub
	}
	return defaults
}

func (b publishingBackend) Append(req timebox.AppendRequest) error {
	err := b.Backend.Append(req)
	if err != nil || len(req.Events) == 0 {
		return err
	}
	b.publish(req.Events...)
	return nil
}

func (b publishingBackend) NewStore(
	cfg timebox.Config,
) (*timebox.Store, error) {
	return timebox.NewStore(b, cfg)
}
