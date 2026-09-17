package app_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/kode4food/timebox/raft"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/app"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/config"
)

func TestStartInvalidRaftConfig(t *testing.T) {
	cfg := newRaftTestConfig(t)
	cfg.Raft.Address = ""

	err := app.New(cfg).Start()
	assert.ErrorIs(t, err, engine.ErrOpenBackend)
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
			app.New(app.Config{
				LogLevel: tt.logLevel,
			}).SetupLogging()

			handler := slog.Default().Handler()
			ctx := t.Context()

			assert.True(t, handler.Enabled(ctx, tt.expected))
			assert.False(t, handler.Enabled(ctx, tt.expected-1))
			assert.True(t, handler.Enabled(ctx, tt.expected+1))
		})
	}
}

func TestStartServesHTTP(t *testing.T) {
	cfg := newRaftTestConfig(t)
	a := app.New(cfg)

	assert.NoError(t, a.Start())
	defer a.Shutdown()

	url := fmt.Sprintf("http://127.0.0.1:%d/health", cfg.APIPort)
	assert.Eventually(t, func() bool {
		res, err := http.Get(url)
		if err != nil {
			return false
		}
		_ = res.Body.Close()
		return res.StatusCode == http.StatusOK
	}, 5*time.Second, 50*time.Millisecond)
}

func TestRun(t *testing.T) {
	cfg := newRaftTestConfig(t)
	cfg.ShutdownTimeout = 100 * time.Millisecond
	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan error, 1)
	go func() {
		done <- app.New(cfg).Run(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for run to exit")
	}
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

func newRaftTestConfig(t *testing.T) app.Config {
	t.Helper()

	addr := availableAddress(t)
	port := availablePort(t)
	nid := "test-node-" + strconv.Itoa(port)

	cfg := config.NewDefaultConfig()
	cfg.NodeID = api.NodeID(nid)

	return app.Config{
		Engine: cfg,
		Raft: raft.DefaultConfig().With(raft.Config{
			LocalID: nid,
			Address: addr,
			DataDir: t.TempDir(),
			Servers: []raft.Server{{ID: nid, Address: addr}},
		}),
		APIPort:         port,
		ShutdownTimeout: app.DefaultShutdownTimeout,
	}
}
