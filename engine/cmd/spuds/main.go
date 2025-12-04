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

	app "github.com/kode4food/spuds/engine"
	"github.com/kode4food/spuds/engine/internal/client"
	"github.com/kode4food/spuds/engine/internal/config"
	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/internal/server"
	"github.com/kode4food/spuds/engine/pkg/log"
)

type spuds struct {
	cfg         *config.Config
	timebox     *timebox.Timebox
	engineStore *timebox.Store
	flowStore   *timebox.Store
	stepClient  client.Client
	engine      *engine.Engine
	health      *server.HealthChecker
	httpServer  *http.Server
}

var logLevels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func main() {
	cfg := config.NewDefaultConfig()
	cfg.LoadFromEnv()

	s := &spuds{cfg: cfg}
	s.setupLogging()

	if err := s.run(); err != nil {
		slog.Error("Failed to start application",
			log.Error(err))
		os.Exit(1)
	}
}

func (s *spuds) run() error {
	if err := s.initializeStores(); err != nil {
		return err
	}

	s.initializeEngine()
	s.startServer()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.shutdown()
	return nil
}

func (s *spuds) setupLogging() {
	level, ok := logLevels[s.cfg.LogLevel]
	if !ok {
		level = slog.LevelInfo
	}

	env := os.Getenv("ENV")
	logger := log.NewWithLevel(app.Name, env, app.Version, level)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(level)

	slog.Info("Spuds Engine starting",
		slog.String("log_level", s.cfg.LogLevel))

	slog.Info("Configuration loaded",
		slog.String("engine_redis_addr", s.cfg.EngineStore.Addr),
		slog.Int("engine_redis_db", s.cfg.EngineStore.DB),
		slog.String("flow_redis_addr", s.cfg.FlowStore.Addr),
		slog.Int("flow_redis_db", s.cfg.FlowStore.DB),
		slog.String("api_host", s.cfg.APIHost),
		slog.Int("api_port", s.cfg.APIPort))
}

func (s *spuds) initializeStores() error {
	var err error

	s.timebox, err = timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  s.cfg.FlowCacheSize,
	})
	if err != nil {
		return fmt.Errorf("failed to create timebox: %w", err)
	}

	s.engineStore, err = s.timebox.NewStore(s.cfg.EngineStore)
	if err != nil {
		_ = s.timebox.Close()
		return fmt.Errorf("failed to create engine store: %w", err)
	}

	s.flowStore, err = s.timebox.NewStore(s.cfg.FlowStore)
	if err != nil {
		_ = s.timebox.Close()
		return fmt.Errorf("failed to create flow store: %w", err)
	}

	return nil
}

func (s *spuds) initializeEngine() {
	s.stepClient = client.NewHTTPClient(
		time.Duration(s.cfg.StepTimeout) * time.Millisecond,
	)

	s.engine = engine.New(
		s.engineStore, s.flowStore, s.stepClient, s.timebox.GetHub(), s.cfg,
	)
	s.engine.Start()
}

func (s *spuds) startServer() {
	s.health = server.NewHealthChecker(s.engine, s.timebox.GetHub())
	s.health.Start()

	srv := server.NewServer(s.engine, s.timebox.GetHub())
	mux := srv.SetupRoutes()

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.cfg.APIHost, s.cfg.APIPort),
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP server starting",
			slog.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error",
				log.Error(err))
		}
	}()
}

func (s *spuds) shutdown() {
	slog.Info("Shutting down")

	ctx, cancel := context.WithTimeout(
		context.Background(), s.cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		slog.Error("Shutdown failed",
			log.Error(err))
	}

	s.health.Stop()

	if err := s.engine.Stop(); err != nil {
		slog.Error("Engine shutdown failed",
			log.Error(err))
	}

	_ = s.timebox.Close()

	slog.Info("Server exited")
}
