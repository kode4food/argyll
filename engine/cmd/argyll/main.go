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

	app "github.com/kode4food/argyll/engine"
	"github.com/kode4food/argyll/engine/internal/client"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type argyll struct {
	cfg         *config.Config
	timebox     *timebox.Timebox
	engineStore *timebox.Store
	flowStore   *timebox.Store
	stepClient  client.Client
	engine      *engine.Engine
	health      *server.HealthChecker
	apiServer   *server.Server
	httpServer  *http.Server
	quit        chan os.Signal
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

	s := &argyll{
		cfg:  cfg,
		quit: make(chan os.Signal, 1),
	}
	s.setupLogging()

	if err := s.run(); err != nil {
		slog.Error("Failed to start application",
			log.Error(err))
		os.Exit(1)
	}
}

func (s *argyll) run() error {
	if err := s.initializeStores(); err != nil {
		return err
	}

	s.initializeEngine()
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
		slog.String("engine_redis_addr", s.cfg.EngineStore.Addr),
		slog.Int("engine_redis_db", s.cfg.EngineStore.DB),
		slog.String("flow_redis_addr", s.cfg.FlowStore.Addr),
		slog.Int("flow_redis_db", s.cfg.FlowStore.DB),
		slog.String("api_host", s.cfg.APIHost),
		slog.Int("api_port", s.cfg.APIPort))
}

func (s *argyll) initializeStores() error {
	var err error

	s.timebox, err = timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  s.cfg.FlowCacheSize,
		Workers:    true,
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

func (s *argyll) initializeEngine() {
	s.stepClient = client.NewHTTPClient(
		time.Duration(s.cfg.StepTimeout) * time.Millisecond,
	)

	s.engine = engine.New(s.engineStore, s.flowStore, s.stepClient, s.cfg)
	s.engine.Start()
}

func (s *argyll) startServer() {
	s.health = server.NewHealthChecker(s.engine, s.timebox.GetHub())
	s.health.Start()

	s.apiServer = server.NewServer(s.engine, s.timebox.GetHub())
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
			slog.Error("HTTP server error",
				log.Error(err))
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
		slog.Error("Shutdown failed",
			log.Error(err))
	}

	s.apiServer.CloseWebSockets()
	s.health.Stop()

	if err := s.engine.Stop(); err != nil {
		slog.Error("Engine shutdown failed",
			log.Error(err))
	}

	_ = s.timebox.Close()

	slog.Info("Server exited")
}
