package main

import (
	"log/slog"
	"os"

	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/workers/agent/internal/openai"
	"github.com/kode4food/argyll/workers/agent/internal/worker"
)

const exitFailed = 1

func main() {
	cfg, provider, err := openai.LoadConfig()
	if err != nil {
		slog.Error("Invalid configuration", log.Error(err))
		os.Exit(exitFailed)
	}
	if err := worker.Serve(cfg, provider); err != nil {
		slog.Error("Agent worker stopped", log.Error(err))
		os.Exit(exitFailed)
	}
}
