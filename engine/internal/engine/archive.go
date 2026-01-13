package engine

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

// ArchiveWorker monitors memory pressure and age to archive flows
type ArchiveWorker struct {
	engine      *Engine
	redisClient *redis.Client
	config      *config.Config
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewArchiveWorker creates a worker that monitors the flows Redis for memory
// pressure and archives deactivated flows accordingly
func NewArchiveWorker(e *Engine, cfg *config.Config) *ArchiveWorker {
	ctx, cancel := context.WithCancel(context.Background())

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.FlowStore.Addr,
		Password: cfg.FlowStore.Password,
		DB:       cfg.FlowStore.DB,
	})

	return &ArchiveWorker{
		engine:      e,
		redisClient: client,
		config:      cfg,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the archiving monitoring loop
func (w *ArchiveWorker) Start() {
	w.wg.Add(1)
	go w.run()
}

// Stop gracefully shuts down the archiving worker
func (w *ArchiveWorker) Stop() {
	w.cancel()
	w.wg.Wait()
	_ = w.redisClient.Close()
}

func (w *ArchiveWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.Archive.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.checkAndArchive()
		}
	}
}

func (w *ArchiveWorker) checkAndArchive() {
	memoryPressure := w.checkMemoryPressure()
	now := time.Now()

	state, err := w.engine.GetEngineState(w.ctx)
	if err != nil || state == nil || len(state.Deactivated) == 0 {
		return
	}

	for _, info := range state.Deactivated {
		if info == nil {
			continue
		}

		shouldArchive := false
		reason := ""

		if memoryPressure {
			shouldArchive = true
			reason = "memory pressure"
		} else if now.Sub(info.DeactivatedAt) > w.config.Archive.MaxAge {
			shouldArchive = true
			reason = "max age exceeded"
		}

		if shouldArchive {
			w.archiveFlow(info.FlowID, reason)
		}

		if memoryPressure {
			break
		}
	}
}

func (w *ArchiveWorker) checkMemoryPressure() bool {
	info, err := w.redisClient.Info(w.ctx, "memory").Result()
	if err != nil {
		slog.Warn("Failed to get Redis memory info", log.Error(err))
		return false
	}

	usedMemory, maxMemory := parseMemoryInfo(info)
	if maxMemory == 0 {
		return false
	}

	usedPercent := (float64(usedMemory) / float64(maxMemory)) * 100
	return usedPercent >= w.config.Archive.MemoryPercent
}

func (w *ArchiveWorker) archiveFlow(flowID api.FlowID, reason string) {
	w.engine.archiveFlow(flowID)

	err := w.engine.raiseEngineEvent(
		w.ctx,
		api.EventTypeFlowArchived,
		api.FlowArchivedEvent{FlowID: flowID},
	)
	if err != nil {
		slog.Warn("Failed to raise flow archived event",
			log.FlowID(flowID), log.Error(err))
		return
	}

	slog.Info("Flow archived by worker",
		log.FlowID(flowID),
		slog.String("reason", reason))
}

func parseMemoryInfo(info string) (used, max int64) {
	lines := strings.SplitSeq(info, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "used_memory:"); ok {
			val := after
			used, _ = strconv.ParseInt(val, 10, 64)
		} else if after, ok := strings.CutPrefix(line, "maxmemory:"); ok {
			val := after
			max, _ = strconv.ParseInt(val, 10, 64)
		}
	}
	return
}
