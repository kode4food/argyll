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
	cfg           *config.Config
	timebox       *timebox.Timebox
	engineStore   *timebox.Store
	workflowStore *timebox.Store
	stepClient    client.Client
	engine        *engine.Engine
	health        *server.HealthChecker
	httpServer    *http.Server
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

func (a *spuds) run() error {
	a.setupLogging()

	if err := a.initializeStores(); err != nil {
		return err
	}

	a.initializeEngine()
	a.startServer()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.shutdown()
	return nil
}

func (a *spuds) setupLogging() {
	level, ok := logLevels[a.cfg.LogLevel]
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
		slog.String("engine_redis_addr", a.cfg.EngineStore.Addr),
		slog.Int("engine_redis_db", a.cfg.EngineStore.DB),
		slog.String("workflow_redis_addr", a.cfg.WorkflowStore.Addr),
		slog.Int("workflow_redis_db", a.cfg.WorkflowStore.DB),
		slog.String("api_host", a.cfg.APIHost),
		slog.Int("api_port", a.cfg.APIPort))
}

func (a *spuds) initializeStores() error {
	var err error

	a.timebox, err = timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  a.cfg.WorkflowCacheSize,
	})
	if err != nil {
		return fmt.Errorf("failed to create timebox: %w", err)
	}

	a.engineStore, err = a.timebox.NewStore(a.cfg.EngineStore)
	if err != nil {
		_ = a.timebox.Close()
		return fmt.Errorf("failed to create engine store: %w", err)
	}

	a.workflowStore, err = a.timebox.NewStore(a.cfg.WorkflowStore)
	if err != nil {
		_ = a.timebox.Close()
		return fmt.Errorf("failed to create workflow store: %w", err)
	}

	return nil
}

func (a *spuds) initializeEngine() {
	a.stepClient = client.NewHTTPClient(
		time.Duration(a.cfg.StepTimeout) * time.Millisecond,
	)

	a.engine = engine.New(
		a.engineStore, a.workflowStore, a.stepClient, a.timebox.GetHub(), a.cfg,
	)
	a.engine.Start()
}

func (a *spuds) startServer() {
	a.health = server.NewHealthChecker(a.engine, a.timebox.GetHub())
	a.health.Start()

	srv := server.NewServer(a.engine, a.cfg, a.timebox.GetHub(), a.stepClient)
	mux := srv.SetupRoutes()

	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", a.cfg.APIHost, a.cfg.APIPort),
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP server starting",
			slog.String("addr", a.httpServer.Addr))
		if err := a.httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error",
				slog.Any("error", err))
		}
	}()
}

func (a *spuds) shutdown() {
	slog.Info("Shutting down")

	ctx, cancel := context.WithTimeout(
		context.Background(), a.cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := a.httpServer.Shutdown(ctx); err != nil {
		slog.Error("Shutdown failed",
			slog.Any("error", err))
	}

	a.health.Stop()

	if err := a.engine.Stop(); err != nil {
		slog.Error("Engine shutdown failed",
			slog.Any("error", err))
	}

	_ = a.timebox.Close()

	slog.Info("Server exited")
}
