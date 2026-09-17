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
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestInitializeEngineInvalidRaftConfig(t *testing.T) {
	s := newRaftTest(t)
	s.raft.Address = ""

	err := s.initializeEngine()

	assert.ErrorIs(t, err, engine.ErrOpenBackend)
	assert.Nil(t, s.backend)
}

func TestSetupLogging(t *testing.T) {
	prevLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prevLogger) })

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
	s := newRaftTest(t)

	err := s.initializeEngine()
	assert.NoError(t, err)

	assert.NotNil(t, s.engine)
	assert.NotNil(t, s.backend)

	assert.NoError(t, s.engine.Stop())
}

func TestStartServer(t *testing.T) {
	s := setupServerTest(t)

	assert.NotNil(t, s.health)
	assert.NotNil(t, s.httpServer)

	s.shutdown()
}

func TestShutdown(t *testing.T) {
	s := setupServerTest(t)

	// Shutdown should not panic
	s.shutdown()
}

func TestRun(t *testing.T) {
	s := newRaftTest(t)
	s.cfg.ShutdownTimeout = 100 * time.Millisecond
	s.quit = make(chan os.Signal, 1)

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

func setupServerTest(t *testing.T) *argyll {
	t.Helper()

	s := newRaftTest(t)
	assert.NoError(t, s.initializeEngine())
	s.startServer()
	return s
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

func newRaftTest(t *testing.T) *argyll {
	t.Helper()

	addr := availableAddress(t)
	port := availablePort(t)
	nid := "test-node-" + strconv.Itoa(port)

	cfg := config.NewDefaultConfig()
	cfg.APIPort = port
	cfg.NodeID = api.NodeID(nid)

	raftCfg := config.DefaultRaftConfig(cfg.NodeID)
	raftCfg.Address = addr
	raftCfg.DataDir = t.TempDir()
	raftCfg.Servers = []raft.Server{{
		ID:      nid,
		Address: addr,
	}}
	return &argyll{cfg: cfg, raft: raftCfg}
}
