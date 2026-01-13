package tests

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
	testTimeout           = 5 * time.Second
	testPollInterval      = 50 * time.Millisecond
	archivePollTimeout    = 10 * time.Millisecond
	archiveCheckInterval  = 10 * time.Millisecond
	archiveMaxAge         = 1 * time.Millisecond
	testCacheSize         = 100
	testAPIPort           = 8080
	testStepTimeout       = 5 * api.Second
	testShutdownTimeout   = 2 * time.Second
	testRetryMax          = 3
	testRetryBackoffMs    = 100
	testRetryMaxBackoffMs = 1000
	testMemoryPercent     = 80.0
)

type testArchivingEnv struct {
	Engine     *engine.Engine
	Worker     *engine.ArchiveWorker
	MockClient *helpers.MockClient
	FlowStore  *timebox.Store
	Cleanup    func()
}

func newArchivingTestEnv(t *testing.T) *testArchivingEnv {
	t.Helper()

	server, err := miniredis.Run()
	assert.NoError(t, err)

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  testCacheSize,
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

	mockCli := helpers.NewMockClient()

	cfg := &config.Config{
		APIPort:         testAPIPort,
		APIHost:         "localhost",
		WebhookBaseURL:  "http://localhost:8080",
		StepTimeout:     testStepTimeout,
		FlowCacheSize:   testCacheSize,
		ShutdownTimeout: testShutdownTimeout,
		Work: api.WorkConfig{
			MaxRetries:   testRetryMax,
			BackoffMs:    testRetryBackoffMs,
			MaxBackoffMs: testRetryMaxBackoffMs,
			BackoffType:  api.BackoffTypeFixed,
		},
		FlowStore: flowConfig,
		Archive: config.ArchiveConfig{
			Enabled:       true,
			CheckInterval: archiveCheckInterval,
			MemoryPercent: testMemoryPercent,
			MaxAge:        archiveMaxAge,
		},
	}

	cfg.FlowStore.Archiving = cfg.Archive.Enabled

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	hub := tb.GetHub()
	eng := engine.New(engineStore, flowStore, mockCli, hub, cfg)
	archiveWorker := engine.NewArchiveWorker(eng, cfg)

	cleanup := func() {
		archiveWorker.Stop()
		_ = eng.Stop()
		_ = tb.Close()
		server.Close()
	}

	return &testArchivingEnv{
		Engine:     eng,
		Worker:     archiveWorker,
		MockClient: mockCli,
		FlowStore:  flowStore,
		Cleanup:    cleanup,
	}
}

func TestArchiveCompletedFlow(t *testing.T) {
	env := newArchivingTestEnv(t)
	defer env.Cleanup()

	env.Engine.Start()
	env.Worker.Start()
	ctx := context.Background()

	step := helpers.NewStepWithOutputs("archive-step", "result")
	err := env.Engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	env.MockClient.SetResponse(step.ID, api.Args{"result": "done"})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	flowID := api.FlowID("archive-test")
	err = env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		flow, flowErr := env.Engine.GetFlowState(ctx, flowID)
		return flowErr == nil && flow.Status == api.FlowCompleted
	}, testTimeout, testPollInterval)

	var archived *timebox.ArchiveRecord
	assert.Eventually(t, func() bool {
		_ = env.FlowStore.PollArchive(ctx, archivePollTimeout, func(
			_ context.Context, record *timebox.ArchiveRecord,
		) error {
			archived = record
			return nil
		})
		return archived != nil
	}, testTimeout, testPollInterval)

	expectedID := timebox.NewAggregateID("flow", timebox.ID(flowID))
	assert.Equal(t, expectedID, archived.AggregateID)
	assert.NotEmpty(t, archived.Events)

	assert.Eventually(t, func() bool {
		state, stateErr := env.Engine.GetEngineState(ctx)
		if stateErr != nil {
			return false
		}
		return !isDeactivated(state, flowID)
	}, testTimeout, testPollInterval)
}

func TestArchiveFailedFlow(t *testing.T) {
	env := newArchivingTestEnv(t)
	defer env.Cleanup()

	env.Engine.Start()
	env.Worker.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep("fail-archive-step")
	step.WorkConfig = &api.WorkConfig{MaxRetries: 0}
	err := env.Engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	flowID := api.FlowID("fail-archive-test")
	err = env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	assert.Eventually(t, func() bool {
		flow, flowErr := env.Engine.GetFlowState(ctx, flowID)
		return flowErr == nil && flow.Status == api.FlowFailed
	}, testTimeout, testPollInterval)

	var archived *timebox.ArchiveRecord
	assert.Eventually(t, func() bool {
		_ = env.FlowStore.PollArchive(ctx, archivePollTimeout, func(
			_ context.Context, record *timebox.ArchiveRecord,
		) error {
			archived = record
			return nil
		})
		return archived != nil
	}, testTimeout, testPollInterval)

	expectedID := timebox.NewAggregateID("flow", timebox.ID(flowID))
	assert.Equal(t, expectedID, archived.AggregateID)
	assert.NotEmpty(t, archived.Events)
}

func isDeactivated(state *api.EngineState, flowID api.FlowID) bool {
	for _, info := range state.Deactivated {
		if info != nil && info.FlowID == flowID {
			return true
		}
	}
	return false
}
