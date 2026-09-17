package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/raft"

	build "github.com/kode4food/argyll/engine"
	"github.com/kode4food/argyll/engine/cmd/argyll/internal/server"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/step"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

// App is the Argyll server: an engine on a raft backend behind the HTTP API
type App struct {
	cfg        Config
	backend    *raft.Backend
	engine     *engine.Engine
	health     *server.HealthChecker
	apiServer  *server.Server
	httpServer *http.Server
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

// Run starts the App, waits for ctx to be done, then shuts the App down
func (a *App) Run(ctx context.Context) error {
	if err := a.Start(); err != nil {
		return err
	}
	<-ctx.Done()
	a.Shutdown()
	return nil
}

// SetupLogging installs the default logger at the configured level
func (a *App) SetupLogging() {
	level, ok := logLevels[a.cfg.Server.LogLevel]
	if !ok {
		level = slog.LevelInfo
	}

	env := os.Getenv("ENV")
	logger := log.NewWithLevel(build.Name, env, build.Version, level)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(level)

	slog.Info("Argyll Engine starting",
		slog.String("log_level", a.cfg.Server.LogLevel))

	slog.Info("Configuration loaded",
		slog.String("raft_node_id", a.cfg.Raft.LocalID),
		slog.String("raft_address", a.cfg.Raft.Address),
		slog.String("raft_data_dir", a.cfg.Raft.DataDir),
		slog.Int("raft_log_tail_size", a.cfg.Raft.LogTailSize),
		slog.String("raft_servers", formatRaftServers(a.cfg.Raft.Servers)),
		slog.String("api_host", a.cfg.Server.APIHost),
		slog.Int("api_port", a.cfg.Server.APIPort))
}

// Start opens the engine on its raft backend and begins serving the HTTP API
func (a *App) Start() error {
	if err := a.startEngine(); err != nil {
		return err
	}
	a.startServer()
	return nil
}

// Shutdown stops the HTTP API, then the engine and the backend it owns
func (a *App) Shutdown() {
	slog.Info("Shutting down")

	ctx, cancel := context.WithTimeout(
		context.Background(), a.cfg.Server.ShutdownTimeout,
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
		stepClient, builtins.BaseCallbackURL(a.cfg.Engine.WebhookBaseURL),
	))

	eng, err := engine.New(a.cfg.Engine, engine.Dependencies{
		Scripts:  script.NewRegistry(),
		Steps:    steps,
		EventHub: event.NewHub(),
	}, a.openRaft)
	if err != nil {
		return err
	}
	if err := eng.Start(); err != nil {
		return errors.Join(err, eng.Stop())
	}
	a.engine = eng
	return nil
}

// openRaft keeps the raft backend it opens, which the server reports status
// from, while the engine owns its lifecycle
func (a *App) openRaft(pub timebox.Publisher) (timebox.Backend, error) {
	b, err := raft.Open(a.cfg.Raft.With(raft.Config{Publisher: pub}))
	if err != nil {
		return nil, err
	}
	a.backend = b
	return b, nil
}

func (a *App) startServer() {
	a.health = server.NewHealthChecker(a.engine)
	a.health.Start()

	a.apiServer = server.NewServer(
		a.engine, a.engine.GetEventHub(),
		NewRaftStatusProvider(a.backend),
	)
	mux := a.apiServer.SetupRoutes()

	a.httpServer = &http.Server{
		Addr: fmt.Sprintf("%s:%d",
			a.cfg.Server.APIHost, a.cfg.Server.APIPort,
		),
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP server starting",
			slog.String("addr", a.httpServer.Addr))
		err := a.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", log.Error(err))
		}
	}()
}
