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

	"github.com/kode4food/spuds/engine/internal/client"
	"github.com/kode4food/spuds/engine/internal/config"
	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/internal/server"
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

	a := &spuds{cfg: cfg}
	if err := a.run(); err != nil {
		slog.Error("Failed to start application",
			slog.Any("error", err))
		os.Exit(1)
	}
}

func (s *spuds) run() error {
	s.setupLogging()

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
	slog.SetLogLoggerLevel(level)

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("Spuds Engine starting")

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

	srv := server.NewServer(s.engine, s.cfg, s.timebox.GetHub(), s.stepClient)
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
				slog.Any("error", err))
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
			slog.Any("error", err))
	}

	s.health.Stop()

	if err := s.engine.Stop(); err != nil {
		slog.Error("Engine shutdown failed",
			slog.Any("error", err))
	}

	_ = s.timebox.Close()

	slog.Info("Server exited")
}
