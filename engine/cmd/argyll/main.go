package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kode4food/argyll/engine/cmd/argyll/internal/app"
	"github.com/kode4food/argyll/engine/pkg/log"
)

func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("Invalid configuration", log.Error(err))
		os.Exit(1)
	}

	a := app.New(cfg)
	a.SetupLogging()

	ctx, stop := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	if err := a.Run(ctx); err != nil {
		slog.Error("Failed to start application", log.Error(err))
		os.Exit(1)
	}
}
