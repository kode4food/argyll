package engine

import (
	"context"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/events"
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

const (
	archiveFlowIDPrefix = "archive"
	archiveStepID       = api.StepID("archive-flow")
	archiveStepName     = api.Name("Archive Flow")
	archiveFlowIDArg    = api.Name("flow_id")
	archiveStepScript   = "(archive-flow flow_id)"
)

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
	pressureRatio := w.checkMemoryPressure()
	memoryPressure := pressureRatio > 0
	now := time.Now()

	maxAge := w.adjustMaxAge(pressureRatio)
	flowIDs := w.selectFlows(
		now, maxAge, memoryPressure,
	)
	if len(flowIDs) > 0 {
		w.startArchiveFlow(flowIDs)
	}
}

func (w *ArchiveWorker) checkMemoryPressure() float64 {
	info, err := w.redisClient.Info(w.ctx, "memory").Result()
	if err != nil {
		slog.Warn("Failed to get Redis memory info", log.Error(err))
		return 0
	}

	usedMemory, maxMemory := parseMemoryInfo(info)
	if maxMemory == 0 {
		return 0
	}

	usedPercent := (float64(usedMemory) / float64(maxMemory)) * 100
	if usedPercent < w.config.Archive.MemoryPercent {
		return 0
	}
	return usedPercent / 100
}

func (w *ArchiveWorker) adjustMaxAge(pressureRatio float64) time.Duration {
	if pressureRatio <= 0 {
		return w.config.Archive.MaxAge
	}

	scaled := time.Duration(float64(w.config.Archive.MaxAge) *
		math.Pow(1-pressureRatio, 2))
	if scaled < time.Minute {
		scaled = time.Minute
	}
	return scaled
}

func (w *ArchiveWorker) selectFlows(
	now time.Time, maxAge time.Duration, memoryPressure bool,
) []api.FlowID {
	var flowIDs []api.FlowID
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		if st == nil || len(st.Deactivated) == 0 {
			return nil
		}

		for _, info := range st.Deactivated {
			if info == nil {
				continue
			}

			shouldArchive := false
			reason := ""

			if memoryPressure {
				shouldArchive = true
				reason = "memory pressure"
			} else if now.Sub(info.DeactivatedAt) > maxAge {
				shouldArchive = true
				reason = "max age exceeded"
			}

			if shouldArchive {
				flowIDs = append(flowIDs, info.FlowID)
				if err := events.Raise(ag,
					api.EventTypeFlowArchiving,
					api.FlowArchivingEvent{FlowID: info.FlowID},
				); err != nil {
					return err
				}
				slog.Info("Flow scheduled for archiving",
					log.FlowID(info.FlowID),
					slog.String("reason", reason))
			}

			if memoryPressure {
				break
			}
		}

		return nil
	}

	_, err := w.engine.engineExec.Exec(w.ctx, events.EngineID, cmd)
	if err != nil {
		slog.Warn("Failed to reserve flows for archiving",
			log.Error(err))
		return nil
	}
	return flowIDs
}

func (w *ArchiveWorker) startArchiveFlow(flowIDs []api.FlowID) {
	flowID := api.FlowID(
		archiveFlowIDPrefix + "-" + uuid.NewString(),
	)
	plan := buildArchivePlan()
	init := api.Args{
		archiveFlowIDArg: toFlowIDArgs(flowIDs),
	}

	if err := w.engine.StartFlow(w.ctx, flowID, plan, init, api.Metadata{}); err != nil {
		slog.Warn("Failed to start archive flow",
			log.FlowID(flowID), log.Error(err))
	}
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

func buildArchivePlan() *api.ExecutionPlan {
	step := &api.Step{
		ID:   archiveStepID,
		Name: archiveStepName,
		Type: api.StepTypeScript,
		Script: &api.ScriptConfig{
			Language: internalScriptLanguage,
			Script:   archiveStepScript,
		},
		Attributes: api.AttributeSpecs{
			archiveFlowIDArg: &api.AttributeSpec{
				Role:    api.RoleRequired,
				Type:    api.TypeString,
				ForEach: true,
			},
		},
	}

	return &api.ExecutionPlan{
		Goals:    []api.StepID{archiveStepID},
		Required: []api.Name{archiveFlowIDArg},
		Steps:    api.Steps{archiveStepID: step},
	}
}

func toFlowIDArgs(flowIDs []api.FlowID) []string {
	ids := make([]string, 0, len(flowIDs))
	for _, flowID := range flowIDs {
		ids = append(ids, string(flowID))
	}
	return ids
}
