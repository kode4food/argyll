package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/postgres"

	build "github.com/kode4food/argyll/engine"
	"github.com/kode4food/argyll/engine/cmd/argyll/internal/server"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

// App is the Argyll server: an engine on a postgres backend behind the HTTP
// API
type App struct {
	cfg        Config
	engine     *engine.Engine
	health     *server.HealthChecker
	apiServer  *server.Server
	httpServer *http.Server
	serveErr   chan error
}

var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

// New creates an App that runs with the provided configuration
func New(cfg Config) *App {
	return &App{cfg: cfg}
}

// Run serves until cancellation or an HTTP server error, then shuts down
func (a *App) Run(ctx context.Context) error {
	if err := a.Start(); err != nil {
		return err
	}
	defer a.Shutdown()
	select {
	case <-ctx.Done():
		return nil
	case err := <-a.serveErr:
		return err
	}
}

// SetupLogging installs the default logger at the configured level
func (a *App) SetupLogging() {
	level, ok := logLevels[a.cfg.LogLevel]
	if !ok {
		level = slog.LevelInfo
	}

	env := os.Getenv("ENV")
	logger := log.NewWithLevel(build.Name, env, build.Version, level)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(level)

	slog.Info("Argyll Engine starting",
		slog.String("log_level", a.cfg.LogLevel))

	slog.Info("Configuration loaded",
		slog.String("postgres_prefix", a.cfg.Postgres.Prefix),
		slog.Int("postgres_max_conns", int(a.cfg.Postgres.MaxConns)),
		slog.String("api_host", a.cfg.APIHost),
		slog.Int("api_port", a.cfg.APIPort))
}

// Start opens the engine on its postgres backend and begins serving the HTTP
// API
func (a *App) Start() error {
	if err := a.startEngine(); err != nil {
		return err
	}
	if err := a.startServer(); err != nil {
		return errors.Join(err, a.engine.Stop())
	}
	return nil
}

// Shutdown stops the HTTP API, then the engine and the backend it owns
func (a *App) Shutdown() {
	slog.Info("Shutting down")

	ctx, cancel := context.WithTimeout(
		context.Background(), a.cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := a.httpServer.Shutdown(ctx); err != nil {
		slog.Error("Shutdown failed", log.Error(err))
	}

	a.apiServer.CloseWebSockets()
	a.health.Stop()

	if err := a.engine.Stop(); err != nil {
		slog.Error("Engine shutdown failed", log.Error(err))
	}

	slog.Info("Server exited")
}

func (a *App) startEngine() error {
	stepClient := builtins.NewHTTPClient(
		time.Duration(a.cfg.Engine.StepTimeout) * time.Millisecond,
	)
	steps := step.NewRegistry(builtins.All(
		stepClient, builtins.BaseCallbackURL(a.cfg.WebhookBaseURL),
	))

	eng, err := engine.New(a.cfg.Engine, engine.Dependencies{
		Scripts:  script.NewRegistry(),
		Steps:    steps,
		EventHub: event.NewHub(),
	}, a.openPostgres)
	if err != nil {
		return err
	}
	if err := eng.Start(); err != nil {
		return errors.Join(err, eng.Stop())
	}
	a.engine = eng
	return nil
}

// openPostgres opens the postgres backend, whose lifecycle the engine owns
func (a *App) openPostgres(pub timebox.Publisher) (timebox.Backend, error) {
	return postgres.Open(a.cfg.Postgres.With(postgres.Config{Publisher: pub}))
}

func (a *App) startServer() error {
	addr := fmt.Sprintf("%s:%d", a.cfg.APIHost, a.cfg.APIPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	a.health = server.NewHealthChecker(a.engine)
	a.health.Start()

	a.apiServer = server.NewServer(a.engine, a.engine.GetEventHub())
	mux := a.apiServer.SetupRoutes()

	a.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	a.serveErr = make(chan error, 1)
	go func() {
		slog.Info("HTTP server starting",
			slog.String("addr", a.httpServer.Addr))
		err := a.httpServer.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		a.serveErr <- err
	}()
	return nil
}
