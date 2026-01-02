package helpers

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// TestEngineEnv holds all the components needed for engine testing
type TestEngineEnv struct {
	Engine      *engine.Engine
	Redis       *miniredis.Miniredis
	MockClient  *MockClient
	Config      *config.Config
	EventHub    timebox.EventHub
	Cleanup     func()
	engineStore *timebox.Store
	flowStore   *timebox.Store
	timebox     *timebox.Timebox
}

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
		ShutdownTimeout: 2 * time.Second,
		Work: api.WorkConfig{
			MaxRetries:   3,
			BackoffMs:    1000,
			MaxBackoffMs: 60000,
			BackoffType:  api.BackoffTypeExponential,
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
		timebox:     tb,
	}
}

// NewEngineInstance creates a new engine instance sharing the same stores
// and mock client. Used to simulate process restart after crash
func (e *TestEngineEnv) NewEngineInstance() *engine.Engine {
	return engine.New(
		e.engineStore, e.flowStore, e.MockClient, e.EventHub,
		e.Config,
	)
}
