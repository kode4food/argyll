package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const (
	archiveTestTimeout        = 5 * time.Second
	archivePollInterval       = 50 * time.Millisecond
	archivePollTimeout        = 10 * time.Millisecond
	archiveCheckInterval      = 10 * time.Millisecond
	archiveMaxAge             = 1 * time.Millisecond
	archiveCacheSize          = 100
	archiveAPIPort            = 8080
	archiveStepTimeout        = 5 * api.Second
	archiveShutdownTimeout    = 2 * time.Second
	archiveRetryMax           = 3
	archiveRetryBackoffMs     = 100
	archiveRetryMaxBackoffMs  = 1000
	archiveMemoryPercent      = 80.0
	archiveCompletedFlowID    = api.FlowID("archive-test")
	archiveFailedFlowID       = api.FlowID("fail-archive-test")
	archiveCompletedStepID    = api.StepID("archive-step")
	archiveFailedStepID       = api.StepID("fail-archive-step")
	archiveCompletedStepKey   = "result"
	archiveCompletedStepValue = "done"
)

type archiveTestEnv struct {
	engine     *engine.Engine
	worker     *engine.ArchiveWorker
	mockClient *helpers.MockClient
	flowStore  *timebox.Store
	cleanup    func()
}

func TestArchiveWorkerCompletedFlow(t *testing.T) {
	env := newArchiveTestEnv(t)
	defer env.cleanup()

	env.engine.Start()
	env.worker.Start()
	ctx := context.Background()

	step := helpers.NewStepWithOutputs(
		archiveCompletedStepID, archiveCompletedStepKey,
	)
	err := env.engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	env.mockClient.SetResponse(step.ID, api.Args{
		archiveCompletedStepKey: archiveCompletedStepValue,
	})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	err = env.engine.StartFlow(
		ctx, archiveCompletedFlowID, plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		flow, flowErr := env.engine.GetFlowState(
			ctx, archiveCompletedFlowID,
		)
		return flowErr == nil && flow.Status == api.FlowCompleted
	}, archiveTestTimeout, archivePollInterval)

	archived := pollArchivedRecord(t, ctx, env.flowStore)
	assert.Equal(t,
		timebox.NewAggregateID(
			"flow", timebox.ID(archiveCompletedFlowID),
		), archived.AggregateID,
	)
	assert.NotEmpty(t, archived.Events)

	assert.Eventually(t, func() bool {
		state, stateErr := env.engine.GetEngineState(ctx)
		if stateErr != nil {
			return false
		}
		return !isDeactivated(state, archiveCompletedFlowID)
	}, archiveTestTimeout, archivePollInterval)
}

func TestArchiveWorkerFailedFlow(t *testing.T) {
	env := newArchiveTestEnv(t)
	defer env.cleanup()

	env.engine.Start()
	env.worker.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep(archiveFailedStepID)
	step.WorkConfig = &api.WorkConfig{MaxRetries: 0}
	err := env.engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	env.mockClient.SetError(step.ID, api.ErrWorkNotCompleted)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	err = env.engine.StartFlow(
		ctx, archiveFailedFlowID, plan, api.Args{}, api.Metadata{},
	)
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		flow, flowErr := env.engine.GetFlowState(
			ctx, archiveFailedFlowID,
		)
		return flowErr == nil && flow.Status == api.FlowFailed
	}, archiveTestTimeout, archivePollInterval)

	archived := pollArchivedRecord(t, ctx, env.flowStore)
	assert.Equal(t,
		timebox.NewAggregateID("flow", timebox.ID(archiveFailedFlowID)),
		archived.AggregateID,
	)
	assert.NotEmpty(t, archived.Events)
}

func newArchiveTestEnv(t *testing.T) *archiveTestEnv {
	t.Helper()

	server, err := miniredis.Run()
	assert.NoError(t, err)

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  archiveCacheSize,
	})
	assert.NoError(t, err)

	engineConfig := config.NewDefaultConfig().EngineStore
	engineConfig.Addr = server.Addr()
	engineConfig.Prefix = "test-engine"

	engineStore, err := tb.NewStore(engineConfig)
	assert.NoError(t, err)

	flowConfig := config.NewDefaultConfig().FlowStore
	flowConfig.Addr = server.Addr()
	flowConfig.Prefix = "test-flow"

	mockClient := helpers.NewMockClient()

	cfg := &config.Config{
		APIPort:         archiveAPIPort,
		APIHost:         "localhost",
		WebhookBaseURL:  "http://localhost:8080",
		StepTimeout:     archiveStepTimeout,
		FlowCacheSize:   archiveCacheSize,
		ShutdownTimeout: archiveShutdownTimeout,
		Work: api.WorkConfig{
			MaxRetries:   archiveRetryMax,
			BackoffMs:    archiveRetryBackoffMs,
			MaxBackoffMs: archiveRetryMaxBackoffMs,
			BackoffType:  api.BackoffTypeFixed,
		},
		FlowStore: flowConfig,
		Archive: config.ArchiveConfig{
			Enabled:       true,
			CheckInterval: archiveCheckInterval,
			MemoryPercent: archiveMemoryPercent,
			MaxAge:        archiveMaxAge,
		},
	}

	cfg.FlowStore.Archiving = cfg.Archive.Enabled

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	hub := tb.GetHub()
	eng := engine.New(engineStore, flowStore, mockClient, hub, cfg)
	archiveWorker := engine.NewArchiveWorker(eng, cfg)

	cleanup := func() {
		archiveWorker.Stop()
		_ = eng.Stop()
		_ = tb.Close()
		server.Close()
	}

	return &archiveTestEnv{
		engine:     eng,
		worker:     archiveWorker,
		mockClient: mockClient,
		flowStore:  flowStore,
		cleanup:    cleanup,
	}
}

func pollArchivedRecord(
	t *testing.T, ctx context.Context, flowStore *timebox.Store,
) *timebox.ArchiveRecord {
	t.Helper()

	var archived *timebox.ArchiveRecord
	assert.Eventually(t, func() bool {
		_ = flowStore.PollArchive(ctx, archivePollTimeout, func(
			_ context.Context, record *timebox.ArchiveRecord,
		) error {
			archived = record
			return nil
		})
		return archived != nil
	}, archiveTestTimeout, archivePollInterval)

	return archived
}

func isDeactivated(state *api.EngineState, flowID api.FlowID) bool {
	for _, info := range state.Deactivated {
		if info != nil && info.FlowID == flowID {
			return true
		}
	}
	return false
}
