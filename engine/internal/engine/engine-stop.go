package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

// Stop gracefully shuts down the engine
func (e *Engine) Stop() error {
	e.eventQueue.Flush()
	e.cancel()
	e.saveEngineSnapshot()
	slog.Info("Engine stopped")
	return nil
}

func (e *Engine) saveEngineSnapshot() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := e.catalogExec.SaveSnapshot(ctx, events.CatalogKey); err != nil {
		slog.Error("Failed to save catalog snapshot", log.Error(err))
	} else {
		slog.Info("Catalog snapshot saved")
	}

	if err := e.partExec.SaveSnapshot(ctx, events.PartitionKey); err != nil {
		slog.Error("Failed to save partition snapshot", log.Error(err))
	} else {
		slog.Info("Partition snapshot saved")
	}
}
