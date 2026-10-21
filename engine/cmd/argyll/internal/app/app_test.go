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

	"github.com/jackc/pgx/v5"
	"github.com/kode4food/timebox/postgres"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/app"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/config"
)

func TestStartInvalidPostgresConfig(t *testing.T) {
	err := app.New(app.Config{
		Engine: config.NewDefaultConfig(),
		Postgres: postgres.DefaultConfig().With(postgres.Config{
			URL: "://bad",
		}),
	}).Start()
	assert.ErrorIs(t, err, engine.ErrOpenBackend)
}

func TestStartOccupiedPort(t *testing.T) {
	cfg := newPostgresTestConfig(t)
	cfg.APIHost = "127.0.0.1"
	blocked, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.APIPort))
	assert.NoError(t, err)
	defer func() { _ = blocked.Close() }()

	err = app.New(cfg).Start()
	assert.Error(t, err)
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
	cfg := newPostgresTestConfig(t)
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
	cfg := newPostgresTestConfig(t)
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

// newPostgresTestConfig points the App at its own schema in the local postgres,
// dropped when the test ends
func newPostgresTestConfig(t *testing.T) app.Config {
	t.Helper()
	if testing.Short() {
		t.Skip("needs a running postgres")
	}

	port := availablePort(t)
	pc := postgres.DefaultConfig().With(postgres.Config{
		Prefix: "argyll_test_" + strconv.Itoa(port),
	})
	t.Cleanup(func() { dropSchema(t, pc) })

	return app.Config{
		Engine:          config.NewDefaultConfig(),
		Postgres:        pc,
		APIPort:         port,
		ShutdownTimeout: app.DefaultShutdownTimeout,
	}
}

func dropSchema(t *testing.T, pc postgres.Config) {
	t.Helper()

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, pc.URL)
	if !assert.NoError(t, err) {
		return
	}
	defer func() { _ = conn.Close(ctx) }()

	_, err = conn.Exec(ctx,
		"DROP SCHEMA IF EXISTS "+pgx.Identifier{pc.Prefix}.Sanitize()+
			" CASCADE",
	)
	assert.NoError(t, err)
}
