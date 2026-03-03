package archiver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
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
		flowExec    *timebox.Executor[*api.FlowState]
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

	flowCandidate struct {
		id            api.FlowID
		deactivatedAt time.Time
	}
)

var (
	ErrFlowStoreRequired   = errors.New("flow store is required")
	ErrRedisClientRequired = errors.New("redis client is required")
)

func NewArchiver(
	flowStore *timebox.Store, redisClient *redis.Client, cfg Config,
) (*Archiver, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if flowStore == nil {
		return nil, ErrFlowStoreRequired
	}
	if redisClient == nil {
		return nil, ErrRedisClientRequired
	}

	return &Archiver{
		flowExec: timebox.NewExecutor(
			flowStore, events.NewFlowState, events.FlowAppliers,
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
		if err := a.flowStore.Archive(bg, events.FlowKey(flowID)); err != nil {
			slog.Warn("Failed to archive flow",
				slog.String("flow_id", string(flowID)),
				slog.String("error", err.Error()))
			continue
		}

		if err := a.flowStore.RemoveAggregateFromStatus(
			bg, events.FlowKey(flowID), events.FlowStatusDeactivated,
		); err != nil {
			slog.Warn("Failed to clear archived flow index",
				slog.String("flow_id", string(flowID)),
				slog.String("error", err.Error()))
		}

		a.releaseLease(bg, flowID)
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
	cands, err := a.selectFlows(now, opts)
	if err != nil {
		return nil, err
	}

	bg := context.Background()
	res := make([]api.FlowID, 0, min(opts.limit, len(cands)))
	for _, cand := range cands {
		if len(res) >= opts.limit {
			break
		}
		ok, err := a.acquireLease(bg, cand.id, opts.leaseTimeout)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		res = append(res, cand.id)
	}

	return res, nil
}

func (a *Archiver) selectFlows(
	now time.Time, opts reserveOptions,
) ([]flowCandidate, error) {
	if opts.limit <= 0 {
		return nil, nil
	}

	ids, err := a.flowStore.ListAggregatesByStatus(
		context.Background(), events.FlowStatusDeactivated,
	)
	if err != nil {
		return nil, err
	}

	selected := make([]flowCandidate, 0, len(ids))
	for _, id := range ids {
		flowID, ok := events.ParseFlowID(id)
		if !ok {
			continue
		}
		flow, ok, err := a.loadFlow(flowID)
		if err != nil {
			return nil, err
		}
		if !ok || flow.DeactivatedAt.IsZero() {
			continue
		}
		selected = append(selected, flowCandidate{
			id:            flowID,
			deactivatedAt: flow.DeactivatedAt,
		})
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].deactivatedAt.Before(selected[j].deactivatedAt)
	})

	if opts.maxAge <= 0 {
		return selected, nil
	}

	ready := make([]flowCandidate, 0, len(selected))
	for _, cand := range selected {
		if now.Sub(cand.deactivatedAt) <= opts.maxAge {
			break
		}
		ready = append(ready, cand)
	}

	return ready, nil
}

func (a *Archiver) loadFlow(
	flowID api.FlowID,
) (*api.FlowState, bool, error) {
	flow, err := a.flowExec.Exec(context.Background(), events.FlowKey(flowID),
		func(
			st *api.FlowState, ag *timebox.Aggregator[*api.FlowState],
		) error {
			return nil
		},
	)
	if err != nil {
		return nil, false, err
	}
	if flow.ID != "" {
		return flow, true, nil
	}
	if err := events.RemoveFlowFromStatuses(
		context.Background(), a.flowStore, flowID,
	); err != nil {
		slog.Warn("Failed to clear stale flow index",
			slog.String("flow_id", string(flowID)),
			slog.String("error", err.Error()))
	}
	return nil, false, nil
}

func (a *Archiver) acquireLease(
	ctx context.Context, flowID api.FlowID, ttl time.Duration,
) (bool, error) {
	return a.redisClient.SetNX(ctx, a.leaseKey(flowID), "1", ttl).Result()
}

func (a *Archiver) releaseLease(ctx context.Context, flowID api.FlowID) {
	if err := a.redisClient.Del(ctx, a.leaseKey(flowID)).Err(); err != nil {
		slog.Warn("Failed to release archive lease",
			slog.String("flow_id", string(flowID)),
			slog.String("error", err.Error()))
	}
}

func (a *Archiver) leaseKey(flowID api.FlowID) string {
	return fmt.Sprintf("%s:archive:lease:%s", a.config.FlowStore.Prefix, flowID)
}

func parseMemoryInfo(info string) (used, max int64) {
	for line := range strings.SplitSeq(info, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "used_memory:"); ok {
			used, _ = strconv.ParseInt(after, 10, 64)
		} else if after, ok := strings.CutPrefix(line, "maxmemory:"); ok {
			max, _ = strconv.ParseInt(after, 10, 64)
		}
	}
	return
}
