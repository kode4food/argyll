package helpers

import (
	"context"
	"errors"
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

// TestEngineEnv holds all the components needed for engine testing
type TestEngineEnv struct {
	Engine      *engine.Engine
	Redis       *miniredis.Miniredis
	MockClient  *MockClient
	Config      *config.Config
	EventHub    *timebox.EventHub
	Cleanup     func()
	engineStore *timebox.Store
	flowStore   *timebox.Store
}

const defaultStoreTimeout = 5 * time.Second

// NewTestConfig creates a default configuration with debug logging enabled
func NewTestConfig() *config.Config {
	cfg := config.NewDefaultConfig()
	cfg.LogLevel = "debug"
	return cfg
}

// NewTestEngine creates a fully configured test engine environment with an
// in-memory Redis backend and mock HTTP client
func NewTestEngine(t *testing.T) *TestEngineEnv {
	t.Helper()

	server, err := miniredis.Run()
	assert.NoError(t, err)

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)

	engineConfig := config.NewDefaultConfig().EngineStore
	engineConfig.Addr = server.Addr()
	engineConfig.Prefix = "test-engine"

	engineStore, err := tb.NewStore(engineConfig)
	assert.NoError(t, err)

	flowConfig := config.NewDefaultConfig().FlowStore
	flowConfig.Addr = server.Addr()
	flowConfig.Prefix = "test-flow"

	flowStore, err := tb.NewStore(flowConfig)
	assert.NoError(t, err)

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
			Backoff:     1000,
			MaxBackoff:  60000,
			BackoffType: api.BackoffTypeExponential,
		},
	}

	hub := tb.GetHub()
	eng := engine.New(engineStore, flowStore, mockCli, hub, cfg)

	cleanup := func() {
		_ = eng.Stop()
		_ = tb.Close()
		server.Close()
	}

	return &TestEngineEnv{
		Engine:      eng,
		Redis:       server,
		MockClient:  mockCli,
		Config:      cfg,
		EventHub:    hub,
		Cleanup:     cleanup,
		engineStore: engineStore,
		flowStore:   flowStore,
	}
}

// NewEngineInstance creates a new engine instance sharing the same stores
// and mock client. Used to simulate process restart after crash
func (e *TestEngineEnv) NewEngineInstance() *engine.Engine {
	return engine.New(
		e.engineStore, e.flowStore, e.MockClient, e.EventHub, e.Config,
	)
}

// AppendFlowEvents appends flow events directly to the flow store
func (e *TestEngineEnv) AppendFlowEvents(
	flowID api.FlowID, evs ...*timebox.Event,
) error {
	ctx, cancel := context.WithTimeout(
		context.Background(), defaultStoreTimeout,
	)
	defer cancel()

	aggregateID := timebox.NewAggregateID(
		events.FlowPrefix, timebox.ID(flowID),
	)
	seq, err := e.getFlowSequence(ctx, aggregateID)
	if err != nil {
		return err
	}

	for i, ev := range evs {
		ev.AggregateID = aggregateID
		ev.Sequence = seq + int64(i)
		if ev.Timestamp.IsZero() {
			ev.Timestamp = time.Now()
		}
	}

	err = e.flowStore.AppendEvents(ctx, aggregateID, seq, evs)
	if err == nil {
		return nil
	}

	conflict := new(timebox.VersionConflictError)
	if !errors.As(err, &conflict) {
		return err
	}

	seq = conflict.ActualSequence
	for i, ev := range evs {
		ev.Sequence = seq + int64(i)
	}

	return e.flowStore.AppendEvents(ctx, aggregateID, seq, evs)
}

func (e *TestEngineEnv) getFlowSequence(
	ctx context.Context, aggregateID timebox.AggregateID,
) (int64, error) {
	eventsInStore, err := e.flowStore.GetEvents(ctx, aggregateID, 0)
	if err != nil {
		return 0, err
	}
	return int64(len(eventsInStore)), nil
}

// WithTestEnv creates a test engine environment, executes the provided
// function with it, and ensures cleanup happens automatically
func WithTestEnv(t *testing.T, fn func(*TestEngineEnv)) {
	t.Helper()
	testEnv := NewTestEngine(t)
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

// WithStartedEngine creates a test engine, starts it, executes the provided
// function with the engine, and ensures cleanup happens automatically
func WithStartedEngine(t *testing.T, fn func(*engine.Engine)) {
	t.Helper()
	WithEngine(t, func(eng *engine.Engine) {
		eng.Start()
		fn(eng)
	})
}
