package main

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/config"
)

func TestInitializeStoresWithInvalidAddress(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = "127.0.0.1:0"
	cfg.FlowStore.Addr = "127.0.0.1:0"

	s := &argyll{cfg: cfg}
	err := s.initializeStores()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create")
}

func TestInitializeStoresSuccess(t *testing.T) {
	server, err := miniredis.Run()
	assert.NoError(t, err)
	defer server.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = server.Addr()
	cfg.FlowStore.Addr = server.Addr()

	s := &argyll{cfg: cfg}
	err = s.initializeStores()

	assert.NoError(t, err)
	assert.NotNil(t, s.timebox)
	assert.NotNil(t, s.engineStore)
	assert.NotNil(t, s.flowStore)

	_ = s.timebox.Close()
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
			ctx := context.Background()

			assert.True(t, handler.Enabled(ctx, tt.expected))
			assert.False(t, handler.Enabled(ctx, tt.expected-1))
			assert.True(t, handler.Enabled(ctx, tt.expected+1))
		})
	}
}

func TestInitializeEngine(t *testing.T) {
	server, err := miniredis.Run()
	assert.NoError(t, err)
	defer server.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = server.Addr()
	cfg.FlowStore.Addr = server.Addr()

	s := &argyll{cfg: cfg}
	err = s.initializeStores()
	assert.NoError(t, err)

	s.initializeEngine()

	assert.NotNil(t, s.stepClient)
	assert.NotNil(t, s.engine)

	_ = s.engine.Stop()
	_ = s.timebox.Close()
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
	server, err := miniredis.Run()
	assert.NoError(t, err)
	defer server.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = server.Addr()
	cfg.FlowStore.Addr = server.Addr()
	cfg.APIPort = 0
	cfg.ShutdownTimeout = 100 * time.Millisecond

	s := &argyll{
		cfg:  cfg,
		quit: make(chan os.Signal, 1),
	}

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

	server, err := miniredis.Run()
	assert.NoError(t, err)

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = server.Addr()
	cfg.FlowStore.Addr = server.Addr()
	cfg.APIPort = 0

	s := &argyll{cfg: cfg}
	err = s.initializeStores()
	assert.NoError(t, err)

	s.initializeEngine()
	s.startServer()

	return s, server.Close
}
