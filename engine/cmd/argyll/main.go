package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/raft"

	app "github.com/kode4food/argyll/engine"
	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type argyll struct {
	cfg         *config.Config
	engStore    *timebox.Store
	flowStore   *timebox.Store
	persistence *raft.Persistence
	engine      *engine.Engine
	health      *server.HealthChecker
	apiServer   *server.Server
	httpServer  *http.Server
	quit        chan os.Signal
}

const defaultStoreReadyTimeout = 5 * time.Second

var (
	ErrCreateStore = errors.New("failed to create raft store")
)

var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func main() {
	cfg := config.NewDefaultConfig()
	if err := cfg.LoadFromEnv(); err != nil {
		slog.Error("Invalid configuration", log.Error(err))
		os.Exit(1)
	}

	s := &argyll{
		cfg:  cfg,
		quit: make(chan os.Signal, 1),
	}
	s.setupLogging()

	if err := s.run(); err != nil {
		slog.Error("Failed to start application", log.Error(err))
		os.Exit(1)
	}
}

func (a *argyll) run() error {
	hub := event.NewHub()
	a.cfg.Raft.Publisher = func(evs ...*timebox.Event) {
		a.engine.HandleCommitted(evs...)
		hub.Publish(evs...)
	}

	if err := a.initializeStores(); err != nil {
		return err
	}

	if err := a.initializeEngine(hub); err != nil {
		return err
	}
	a.startServer()

	signal.Notify(a.quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(a.quit)
	<-a.quit

	a.shutdown()
	return nil
}

func (a *argyll) setupLogging() {
	level, ok := logLevels[a.cfg.LogLevel]
	if !ok {
		level = slog.LevelInfo
	}

	env := os.Getenv("ENV")
	logger := log.NewWithLevel(app.Name, env, app.Version, level)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(level)

	slog.Info("Argyll Engine starting",
		slog.String("log_level", a.cfg.LogLevel))

	slog.Info("Configuration loaded",
		slog.String("raft_node_id", a.cfg.Raft.LocalID),
		slog.String("raft_address", a.cfg.Raft.Address),
		slog.String("raft_data_dir", a.cfg.Raft.DataDir),
		slog.Int("raft_log_tail_size", a.cfg.Raft.LogTailSize),
		slog.String("raft_servers", formatRaftServers(a.cfg.Raft.Servers)),
		slog.String("api_host", a.cfg.APIHost),
		slog.Int("api_port", a.cfg.APIPort))
}

func (a *argyll) initializeStores() error {
	p, err := raft.NewPersistence(a.cfg.Raft)
	if err != nil {
		return errors.Join(ErrCreateStore, err)
	}
	engStore, err := p.NewStore(a.cfg.EngineStoreConfig())
	if err != nil {
		_ = p.Close()
		return errors.Join(ErrCreateStore, err)
	}
	flowStore, err := p.NewStore(a.cfg.FlowStoreConfig())
	if err != nil {
		_ = engStore.Close()
		return errors.Join(ErrCreateStore, err)
	}
	ctx, cancel := context.WithTimeout(
		context.Background(), defaultStoreReadyTimeout,
	)
	defer cancel()
	if err := flowStore.WaitReady(ctx); err != nil {
		_ = engStore.Close()
		return errors.Join(ErrCreateStore, err)
	}

	a.persistence = p
	a.engStore = engStore
	a.flowStore = flowStore
	return nil
}

func (a *argyll) initializeEngine(hub *event.Hub) error {
	stepClient := client.NewHTTPClient(
		time.Duration(a.cfg.StepTimeout) * time.Millisecond,
	)

	eng, err := engine.New(a.cfg, engine.Dependencies{
		EngineStore: a.engStore,
		FlowStore:   a.flowStore,
		StepClient:  stepClient,
		EventHub:    hub,
	})
	if err != nil {
		return err
	}
	a.engine = eng
	return a.engine.Start()
}

func (a *argyll) startServer() {
	a.health = server.NewHealthChecker(a.engine)
	a.health.Start()

	a.apiServer = server.NewServer(
		a.engine, a.engine.GetEventHub(),
		server.NewRaftStatusProvider(a.persistence),
	)
	mux := a.apiServer.SetupRoutes()

	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.cfg.APIHost, a.cfg.APIPort),
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

func (a *argyll) shutdown() {
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

	a.closeStores()

	slog.Info("Server exited")
}

func (a *argyll) closeStores() {
	if a.flowStore == nil {
		return
	}

	_ = a.flowStore.Close()
	_ = a.engStore.Close()
	a.engStore = nil
	a.flowStore = nil
	a.persistence = nil
}

func formatRaftServers(srvs []raft.Server) string {
	parts := make([]string, 0, len(srvs))
	for _, srv := range srvs {
		parts = append(parts, srv.ID+"="+srv.Address)
	}
	return strings.Join(parts, ",")
}
