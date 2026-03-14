package main

import (
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/timebox/memory"

	"github.com/kode4food/argyll/engine/internal/config"
)

func TestInitStoresInvalidAddr(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.CatalogStore.URL = "://bad"

	s := &argyll{cfg: cfg}
	err := s.initializeStores()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create")
}

func TestInitializeStoresPreconfigured(t *testing.T) {
	cfg := config.NewDefaultConfig()
	s, cleanup := setupArgyllWithStores(t, cfg)
	defer cleanup()

	err := s.initializeStores()

	assert.NoError(t, err)
	assert.NotNil(t, s.catalogStore)
	assert.NotNil(t, s.partitionStore)
	assert.NotNil(t, s.flowStore)
}

func TestSamePostgresBackend(t *testing.T) {
	cfg := config.NewDefaultConfig()

	assert.True(t, samePostgresBackend(
		cfg.CatalogStore,
		cfg.PartitionStore,
		cfg.FlowStore,
	))

	cfg.FlowStore.Prefix = "other"
	assert.False(t, samePostgresBackend(
		cfg.CatalogStore,
		cfg.PartitionStore,
		cfg.FlowStore,
	))
}

func TestSetupLogging(t *testing.T) {
	prevLogger := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(prevLogger)
	})

	tests := []struct {
		name     string
		logLevel string
		expected slog.Level
	}{
		{"debug level", "debug", slog.LevelDebug},
		{"info level", "info", slog.LevelInfo},
		{"warn level", "warn", slog.LevelWarn},
		{"error level", "error", slog.LevelError},
		{"invalid defaults to info", "invalid", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewDefaultConfig()
			cfg.LogLevel = tt.logLevel

			s := &argyll{cfg: cfg}
			s.setupLogging()

			handler := slog.Default().Handler()
			ctx := t.Context()

			assert.True(t, handler.Enabled(ctx, tt.expected))
			assert.False(t, handler.Enabled(ctx, tt.expected-1))
			assert.True(t, handler.Enabled(ctx, tt.expected+1))
		})
	}
}

func TestInitializeEngine(t *testing.T) {
	cfg := config.NewDefaultConfig()
	s, cleanup := setupArgyllWithStores(t, cfg)
	defer cleanup()

	err := s.initializeEngine()
	assert.NoError(t, err)

	assert.NotNil(t, s.stepClient)
	assert.NotNil(t, s.engine)

	_ = s.engine.Stop()
}

func TestStartServer(t *testing.T) {
	s, cleanup := setupServerTest(t)
	defer cleanup()

	assert.NotNil(t, s.health)
	assert.NotNil(t, s.httpServer)

	s.shutdown()
}

func TestShutdown(t *testing.T) {
	s, cleanup := setupServerTest(t)
	defer cleanup()

	// Shutdown should not panic
	s.shutdown()
}

func TestRun(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.APIPort = availablePort(t)
	cfg.ShutdownTimeout = 100 * time.Millisecond

	s, cleanup := setupArgyllWithStores(t, cfg)
	defer cleanup()
	s.quit = make(chan os.Signal, 1)

	done := make(chan error, 1)
	go func() {
		done <- s.run()
	}()

	s.quit <- os.Interrupt

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for run to exit")
	}
}

func TestLogLevels(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, logLevels["debug"])
	assert.Equal(t, slog.LevelInfo, logLevels["info"])
	assert.Equal(t, slog.LevelWarn, logLevels["warn"])
	assert.Equal(t, slog.LevelError, logLevels["error"])
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func setupServerTest(t *testing.T) (*argyll, func()) {
	t.Helper()

	cfg := config.NewDefaultConfig()
	cfg.APIPort = availablePort(t)

	s, cleanup := setupArgyllWithStores(t, cfg)

	err := s.initializeEngine()
	assert.NoError(t, err)
	s.startServer()

	return s, cleanup
}

func setupArgyllWithStores(
	t *testing.T, cfg *config.Config,
) (*argyll, func()) {
	t.Helper()

	catStore, err := memory.NewStore(cfg.CatalogStore.Timebox)
	assert.NoError(t, err)

	partStore, err := memory.NewStore(cfg.PartitionStore.Timebox)
	assert.NoError(t, err)

	flowStore, err := memory.NewStore(cfg.FlowStore.Timebox)
	assert.NoError(t, err)

	s := &argyll{
		cfg:            cfg,
		catalogStore:   catStore,
		partitionStore: partStore,
		flowStore:      flowStore,
		quit:           make(chan os.Signal, 1),
	}

	cleanup := func() {
		if s.flowStore != nil {
			_ = s.flowStore.Close()
		}
		if s.partitionStore != nil {
			_ = s.partitionStore.Close()
		}
		if s.catalogStore != nil {
			_ = s.catalogStore.Close()
		}
	}

	return s, cleanup
}

func availablePort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	addr, ok := ln.Addr().(*net.TCPAddr)
	assert.True(t, ok)
	return addr.Port
}
