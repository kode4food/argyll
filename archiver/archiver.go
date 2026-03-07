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
		id       api.FlowID
		statusAt time.Time
	}
)

var (
	ErrFlowStoreRequired   = errors.New("flow store is required")
	ErrRedisClientRequired = errors.New("redis client is required")
	ErrSelectFlowsFailed   = errors.New("failed to select flows")
	ErrInvalidFlowStatus   = errors.New("invalid flow status entry")
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

	entries, err := a.flowStore.ListAggregatesByStatus(
		context.Background(), events.FlowStatusCompleted,
	)
	if err != nil {
		return nil, errors.Join(ErrSelectFlowsFailed, err)
	}

	failed, err := a.flowStore.ListAggregatesByStatus(
		context.Background(), events.FlowStatusFailed,
	)
	if err != nil {
		return nil, errors.Join(ErrSelectFlowsFailed, err)
	}

	selected := make([]flowCandidate, 0, len(entries)+len(failed))
	for _, group := range [][]timebox.StatusEntry{entries, failed} {
		for _, entry := range group {
			flowID, ok := events.ParseFlowID(entry.ID)
			if !ok {
				return nil, errors.Join(
					ErrSelectFlowsFailed,
					fmt.Errorf("%w: %s", ErrInvalidFlowStatus,
						entry.ID.Join(":")),
				)
			}
			selected = append(selected, flowCandidate{
				id:       flowID,
				statusAt: entry.Timestamp,
			})
		}
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].statusAt.Before(selected[j].statusAt)
	})

	if opts.maxAge <= 0 {
		return selected, nil
	}

	ready := make([]flowCandidate, 0, len(selected))
	for _, cand := range selected {
		if now.Sub(cand.statusAt) <= opts.maxAge {
			break
		}
		ready = append(ready, cand)
	}

	return ready, nil
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
