package main

import (
	"log/slog"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/kode4food/timebox/raft"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/event"
)

func TestInitStoresInvalidRaftConfig(t *testing.T) {
	cfg := newRaftTestConfig(t)
	cfg.Raft.Address = ""

	s := &argyll{cfg: cfg}
	err := s.initializeStores()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create")
}

func TestInitializeStoresSuccess(t *testing.T) {
	cfg := newRaftTestConfig(t)

	s := &argyll{cfg: cfg}
	err := s.initializeStores()

	assert.NoError(t, err)
	assert.NotNil(t, s.catStore)
	assert.NotNil(t, s.partStore)
	assert.NotNil(t, s.flowStore)

	s.closeStores()
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
	cfg := newRaftTestConfig(t)

	s := &argyll{cfg: cfg}
	err := s.initializeStores()
	assert.NoError(t, err)

	err = s.initializeEngine(event.NewHub())
	assert.NoError(t, err)

	assert.NotNil(t, s.engine)

	_ = s.engine.Stop()
	s.closeStores()
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
	cfg := newRaftTestConfig(t)
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
	case <-time.After(10 * time.Second):
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

	cfg := newRaftTestConfig(t)

	s := &argyll{cfg: cfg}
	err := s.initializeStores()
	assert.NoError(t, err)

	err = s.initializeEngine(event.NewHub())
	assert.NoError(t, err)
	s.startServer()

	return s, s.closeStores
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

func availableAddress(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	return ln.Addr().String()
}

func newRaftTestConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg := config.NewDefaultConfig()
	addr := availableAddress(t)
	port := availablePort(t)
	nodeID := "test-node-" + strconv.Itoa(port)

	cfg.APIPort = port
	cfg.Raft.LocalID = nodeID
	cfg.Raft.Address = addr
	cfg.Raft.DataDir = t.TempDir()
	cfg.Raft.Servers = []raft.Server{{
		ID:      nodeID,
		Address: addr,
	}}
	return cfg
}
