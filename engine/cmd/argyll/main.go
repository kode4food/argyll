package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/postgres"

	app "github.com/kode4food/argyll/engine"
	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type argyll struct {
	cfg            *config.Config
	storeBackend   io.Closer
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
	ErrPartialStores        = errors.New("stores must be configured together")
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
		slog.String("catalog_postgres_prefix", s.cfg.CatalogStore.Prefix),
		slog.String("partition_postgres_prefix", s.cfg.PartitionStore.Prefix),
		slog.String("flow_postgres_prefix", s.cfg.FlowStore.Prefix),
		slog.String("api_host", s.cfg.APIHost),
		slog.Int("api_port", s.cfg.APIPort))
}

func (s *argyll) initializeStores() error {
	if s.storeBackend != nil ||
		s.catalogStore != nil ||
		s.partitionStore != nil ||
		s.flowStore != nil {
		if s.catalogStore != nil &&
			s.partitionStore != nil &&
			s.flowStore != nil {
			return nil
		}
		return ErrPartialStores
	}

	if samePostgresBackend(
		s.cfg.CatalogStore,
		s.cfg.PartitionStore,
		s.cfg.FlowStore,
	) {
		return s.initializeSharedStores()
	}
	return s.initializeSeparateStores()
}

func (s *argyll) initializeSharedStores() error {
	p, err := postgres.NewPersistence(s.cfg.CatalogStore)
	if err != nil {
		return errors.Join(ErrCreateCatalogStore, err)
	}
	shared := sharedPersistence{Persistence: p}
	s.storeBackend = p

	s.catalogStore, err = timebox.NewStore(shared, s.cfg.CatalogStore.Timebox)
	if err != nil {
		_ = p.Close()
		s.storeBackend = nil
		return errors.Join(ErrCreateCatalogStore, err)
	}

	s.partitionStore, err = timebox.NewStore(
		shared, s.cfg.PartitionStore.Timebox,
	)
	if err != nil {
		_ = s.catalogStore.Close()
		_ = p.Close()
		s.storeBackend = nil
		return errors.Join(ErrCreatePartitionStore, err)
	}

	s.flowStore, err = timebox.NewStore(shared, s.cfg.FlowStore.Timebox)
	if err != nil {
		_ = s.partitionStore.Close()
		_ = s.catalogStore.Close()
		_ = p.Close()
		s.storeBackend = nil
		return errors.Join(ErrCreateFlowStore, err)
	}
	return nil
}

func (s *argyll) initializeSeparateStores() error {
	var err error

	s.catalogStore, err = postgres.NewStore(s.cfg.CatalogStore)
	if err != nil {
		return errors.Join(ErrCreateCatalogStore, err)
	}

	s.partitionStore, err = postgres.NewStore(s.cfg.PartitionStore)
	if err != nil {
		_ = s.catalogStore.Close()
		return errors.Join(ErrCreatePartitionStore, err)
	}

	s.flowStore, err = postgres.NewStore(s.cfg.FlowStore)
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

	closeStore(&s.flowStore)
	closeStore(&s.partitionStore)
	closeStore(&s.catalogStore)
	closeBackend(&s.storeBackend)

	slog.Info("Server exited")
}

type sharedPersistence struct {
	timebox.Persistence
}

func (s sharedPersistence) Close() error {
	return nil
}

func samePostgresBackend(cfgs ...postgres.Config) bool {
	if len(cfgs) == 0 {
		return true
	}
	base := cfgs[0]
	for _, cfg := range cfgs[1:] {
		if cfg.URL != base.URL ||
			cfg.Prefix != base.Prefix ||
			cfg.MaxConns != base.MaxConns {
			return false
		}
	}
	return true
}

func closeStore(s **timebox.Store) {
	if *s == nil {
		return
	}
	_ = (*s).Close()
	*s = nil
}

func closeBackend(c *io.Closer) {
	if *c == nil {
		return
	}
	_ = (*c).Close()
	*c = nil
}
