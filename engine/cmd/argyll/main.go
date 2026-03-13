package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/redis"

	app "github.com/kode4food/argyll/engine"
	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type argyll struct {
	cfg            *config.Config
	catalogStore   *timebox.Store
	partitionStore *timebox.Store
	flowStore      *timebox.Store
	stepClient     client.Client
	engine         *engine.Engine
	health         *server.HealthChecker
	apiServer      *server.Server
	httpServer     *http.Server
	quit           chan os.Signal
}

var (
	ErrCreateCatalogStore   = errors.New("failed to create catalog store")
	ErrCreatePartitionStore = errors.New("failed to create partition store")
	ErrCreateFlowStore      = errors.New("failed to create flow store")
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
		slog.String("catalog_redis_addr", s.cfg.CatalogStore.Addr),
		slog.Int("catalog_redis_db", s.cfg.CatalogStore.DB),
		slog.String("partition_redis_addr", s.cfg.PartitionStore.Addr),
		slog.Int("partition_redis_db", s.cfg.PartitionStore.DB),
		slog.String("flow_redis_addr", s.cfg.FlowStore.Addr),
		slog.Int("flow_redis_db", s.cfg.FlowStore.DB),
		slog.String("api_host", s.cfg.APIHost),
		slog.Int("api_port", s.cfg.APIPort))
}

func (s *argyll) initializeStores() error {
	var err error

	s.catalogStore, err = redis.NewStore(s.cfg.CatalogStore)
	if err != nil {
		return errors.Join(ErrCreateCatalogStore, err)
	}

	s.partitionStore, err = redis.NewStore(s.cfg.PartitionStore)
	if err != nil {
		_ = s.catalogStore.Close()
		return errors.Join(ErrCreatePartitionStore, err)
	}

	s.flowStore, err = redis.NewStore(s.cfg.FlowStore)
	if err != nil {
		_ = s.partitionStore.Close()
		_ = s.catalogStore.Close()
		return errors.Join(ErrCreateFlowStore, err)
	}

	return nil
}

func (s *argyll) initializeEngine() error {
	s.stepClient = client.NewHTTPClient(
		time.Duration(s.cfg.StepTimeout) * time.Millisecond,
	)

	eng, err := engine.New(s.cfg, engine.Dependencies{
		CatalogStore:   s.catalogStore,
		PartitionStore: s.partitionStore,
		FlowStore:      s.flowStore,
		StepClient:     s.stepClient,
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

	_ = s.flowStore.Close()
	_ = s.partitionStore.Close()
	_ = s.catalogStore.Close()

	slog.Info("Server exited")
}
