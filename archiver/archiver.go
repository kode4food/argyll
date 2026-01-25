package archiver

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kode4food/timebox"
	"github.com/redis/go-redis/v9"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type (
	Archiver struct {
		engineExec  *timebox.Executor[*api.EngineState]
		flowStore   *timebox.Store
		redisClient *redis.Client
		config      Config
		mu          sync.Mutex
	}

	reserveOptions struct {
		limit        int
		maxAge       time.Duration
		leaseTimeout time.Duration
	}
)

func NewArchiver(
	engineStore *timebox.Store, flowStore *timebox.Store,
	redisClient *redis.Client, cfg Config,
) (*Archiver, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if engineStore == nil {
		return nil, errors.New("engine store is required")
	}
	if flowStore == nil {
		return nil, errors.New("flow store is required")
	}
	if redisClient == nil {
		return nil, errors.New("redis client is required")
	}

	return &Archiver{
		engineExec: timebox.NewExecutor(
			engineStore, events.NewEngineState, events.EngineAppliers,
		),
		flowStore:   flowStore,
		redisClient: redisClient,
		config:      cfg,
	}, nil
}

func (a *Archiver) Run(ctx context.Context) error {
	pressureTicker := time.NewTicker(a.config.MemoryCheckInterval)
	ageTicker := time.NewTicker(a.config.SweepInterval)
	defer pressureTicker.Stop()
	defer ageTicker.Stop()

	a.runPressureCycle(ctx)
	a.runAgeSweep()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-pressureTicker.C:
			a.runPressureCycle(ctx)
		case <-ageTicker.C:
			a.runAgeSweep()
		}
	}
}

func (a *Archiver) runPressureCycle(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ratio := a.checkMemoryPressure(ctx)
	if ratio <= 0 {
		return
	}

	flowIDs, err := a.reserveFlows(reserveOptions{
		limit:        a.config.PressureBatchSize,
		leaseTimeout: a.config.LeaseTimeout,
	})
	if err != nil {
		slog.Warn("Failed to reserve flows under memory pressure",
			slog.String("error", err.Error()))
		return
	}

	a.archiveFlows(flowIDs)
}

func (a *Archiver) runAgeSweep() {
	a.mu.Lock()
	defer a.mu.Unlock()

	flowIDs, err := a.reserveFlows(reserveOptions{
		limit:        a.config.SweepBatchSize,
		maxAge:       a.config.MaxAge,
		leaseTimeout: a.config.LeaseTimeout,
	})
	if err != nil {
		slog.Warn("Failed to reserve flows for age sweep",
			slog.String("error", err.Error()))
		return
	}

	a.archiveFlows(flowIDs)
}

func (a *Archiver) archiveFlows(flowIDs []api.FlowID) {
	if len(flowIDs) == 0 {
		return
	}

	bg := context.Background()
	for _, flowID := range flowIDs {
		if err := a.flowStore.Archive(bg, flowKey(flowID)); err != nil {
			slog.Warn("Failed to archive flow",
				slog.String("flow_id", string(flowID)),
				slog.String("error", err.Error()))
			continue
		}
		if err := a.raiseEngineEvent(
			api.EventTypeFlowArchived, api.FlowArchivedEvent{FlowID: flowID},
		); err != nil {
			slog.Warn("Failed to emit flow archived event",
				slog.String("flow_id", string(flowID)),
				slog.String("error", err.Error()))
		}
	}
}

func (a *Archiver) checkMemoryPressure(ctx context.Context) float64 {
	info, err := a.redisClient.Info(ctx, "memory").Result()
	if err != nil {
		slog.Warn("Failed to read Redis memory info",
			slog.String("error", err.Error()))
		return 0
	}

	usedMemory, maxMemory := parseMemoryInfo(info)
	if maxMemory == 0 {
		return 0
	}

	usedPercent := (float64(usedMemory) / float64(maxMemory)) * 100
	if usedPercent < a.config.MemoryPercent {
		return 0
	}
	return usedPercent / 100
}

func (a *Archiver) reserveFlows(opts reserveOptions) ([]api.FlowID, error) {
	now := time.Now()
	var flowIDs []api.FlowID

	cmd := func(
		st *api.EngineState, ag *timebox.Aggregator[*api.EngineState],
	) error {
		if st == nil {
			return nil
		}

		flowIDs = selectFlows(st, now, opts)
		for _, flowID := range flowIDs {
			if err := timebox.Raise(
				ag,
				timebox.EventType(api.EventTypeFlowArchiving),
				api.FlowArchivingEvent{FlowID: flowID},
			); err != nil {
				return err
			}
		}
		return nil
	}

	bg := context.Background()
	_, err := a.engineExec.Exec(bg, events.EngineID, cmd)
	return flowIDs, err
}

func (a *Archiver) raiseEngineEvent(eventType api.EventType, data any) error {
	bg := context.Background()
	_, err := a.engineExec.Exec(bg, events.EngineID,
		func(
			st *api.EngineState, ag *timebox.Aggregator[*api.EngineState],
		) error {
			return timebox.Raise(ag, timebox.EventType(eventType), data)
		},
	)
	return err
}

func selectFlows(
	st *api.EngineState, now time.Time, opts reserveOptions,
) []api.FlowID {
	if opts.limit <= 0 {
		return nil
	}

	selected := make([]api.FlowID, 0, opts.limit)

	for flowID, archivingAt := range st.Archiving {
		if now.Sub(archivingAt) <= opts.leaseTimeout {
			continue
		}
		selected = append(selected, flowID)
		if len(selected) >= opts.limit {
			return selected
		}
	}

	for _, info := range st.Deactivated {
		if info == nil {
			continue
		}
		if opts.maxAge > 0 && now.Sub(info.DeactivatedAt) <= opts.maxAge {
			continue
		}
		selected = append(selected, info.FlowID)
		if len(selected) >= opts.limit {
			return selected
		}
	}

	return selected
}

func flowKey(flowID api.FlowID) timebox.AggregateID {
	return timebox.NewAggregateID("flow", timebox.ID(flowID))
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
