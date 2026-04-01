package engine

import (
	"log/slog"

	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

// Stop gracefully shuts down the engine
func (e *Engine) Stop() error {
	e.cancel()
	e.saveEngineSnapshot()
	slog.Info("Engine stopped")
	return nil
}

func (e *Engine) saveEngineSnapshot() {
	if err := e.catalogExec.SaveSnapshot(events.CatalogKey); err != nil {
		slog.Error("Failed to save catalog snapshot", log.Error(err))
	} else {
		slog.Info("Catalog snapshot saved")
	}

	if err := e.nodeExec.SaveSnapshot(e.nodeKey()); err != nil {
		slog.Error("Failed to save node snapshot", log.Error(err))
	} else {
		slog.Info("Node snapshot saved")
	}
}
