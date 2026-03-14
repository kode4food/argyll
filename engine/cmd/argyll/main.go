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
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type argyll struct {
	cfg        *config.Config
	store      *timebox.Store
	engine     *engine.Engine
	health     *server.HealthChecker
	apiServer  *server.Server
	httpServer *http.Server
	quit       chan os.Signal
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

func (s *argyll) run() error {
	if err := s.initializeStores(); err != nil {
		return err
	}

	if err := s.initializeEngine(); err != nil {
		return err
	}
	s.startServer()

	signal.Notify(s.quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(s.quit)
	<-s.quit

	s.shutdown()
	return nil
}

func (s *argyll) setupLogging() {
	level, ok := logLevels[s.cfg.LogLevel]
	if !ok {
		level = slog.LevelInfo
	}

	env := os.Getenv("ENV")
	logger := log.NewWithLevel(app.Name, env, app.Version, level)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(level)

	slog.Info("Argyll Engine starting",
		slog.String("log_level", s.cfg.LogLevel))

	slog.Info("Configuration loaded",
		slog.String("raft_node_id", s.cfg.Raft.LocalID),
		slog.String("raft_bind_address", s.cfg.Raft.BindAddress),
		slog.String("raft_advertise_address", s.cfg.Raft.ServerAddress()),
		slog.String("raft_forward_bind_address", s.cfg.Raft.ForwardBindAddress),
		slog.String("raft_forward_advertise_address",
			s.cfg.Raft.ForwardAddress(),
		),
		slog.String("raft_data_dir", s.cfg.Raft.DataDir),
		slog.String("raft_servers", formatRaftServers(s.cfg.Raft.Servers)),
		slog.String("api_host", s.cfg.APIHost),
		slog.Int("api_port", s.cfg.APIPort))
}

func (s *argyll) initializeStores() error {
	store, err := raft.NewStore(s.cfg.Raft.With(raft.Config{
		LogOutput: os.Stdout,
	}))
	if err != nil {
		return errors.Join(ErrCreateStore, err)
	}
	ctx, cancel := context.WithTimeout(
		context.Background(), defaultStoreReadyTimeout,
	)
	defer cancel()
	if err := store.WaitReady(ctx); err != nil {
		_ = store.Close()
		return errors.Join(ErrCreateStore, err)
	}

	// Raft persistence does not use logical prefixes, so one replicated store
	// holds catalog, partition, and flow aggregates
	s.store = store
	return nil
}

func (s *argyll) initializeEngine() error {
	stepClient := client.NewHTTPClient(
		time.Duration(s.cfg.StepTimeout) * time.Millisecond,
	)

	eng, err := engine.New(s.cfg, engine.Dependencies{
		Store:      s.store,
		StepClient: stepClient,
	})
	if err != nil {
		return err
	}
	s.engine = eng
	return s.engine.Start()
}

func (s *argyll) startServer() {
	hub := s.engine.GetEventHub()

	s.health = server.NewHealthChecker(s.engine, hub)
	s.health.Start()

	s.apiServer = server.NewServer(s.engine, hub)
	mux := s.apiServer.SetupRoutes()

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.cfg.APIHost, s.cfg.APIPort),
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP server starting",
			slog.String("addr", s.httpServer.Addr))
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", log.Error(err))
		}
	}()
}

func (s *argyll) shutdown() {
	slog.Info("Shutting down")

	ctx, cancel := context.WithTimeout(
		context.Background(), s.cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		slog.Error("Shutdown failed", log.Error(err))
	}

	s.apiServer.CloseWebSockets()
	s.health.Stop()

	if err := s.engine.Stop(); err != nil {
		slog.Error("Engine shutdown failed", log.Error(err))
	}

	s.closeStores()

	slog.Info("Server exited")
}

func (s *argyll) closeStores() {
	if s.store == nil {
		return
	}

	_ = s.store.Close()
	s.store = nil
}

func formatRaftServers(srvs []raft.Server) string {
	parts := make([]string, 0, len(srvs))
	for _, srv := range srvs {
		part := srv.ID + "=" + srv.Address
		if srv.ForwardAddress != "" {
			part += "|" + srv.ForwardAddress
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ",")
}
