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
	"github.com/kode4food/argyll/engine/internal/hibernate"
	"github.com/kode4food/argyll/engine/pkg/api"

	_ "gocloud.dev/blob/memblob"
)

// testHibernationEnv creates a test environment with hibernation enabled
type testHibernationEnv struct {
	Engine      *engine.Engine
	Redis       *miniredis.Miniredis
	MockClient  *helpers.MockClient
	Hibernator  *hibernate.BlobHibernator
	Cleanup     func()
	flowStore   *timebox.Store
	engineStore *timebox.Store
	timebox     *timebox.Timebox
	hub         timebox.EventHub
}

func newHibernationTestEnv(t *testing.T) *testHibernationEnv {
	t.Helper()

	ctx := context.Background()

	server, err := miniredis.Run()
	assert.NoError(t, err)

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
	})
	assert.NoError(t, err)

	engineConfig := config.NewDefaultConfig().EngineStore
	engineConfig.Addr = server.Addr()
	engineConfig.Prefix = "test-engine"

	engineStore, err := tb.NewStore(engineConfig)
	assert.NoError(t, err)

	hibernator, err := hibernate.NewBlobHibernator(ctx, "mem://", "hibernated/")
	assert.NoError(t, err)

	flowConfig := config.NewDefaultConfig().FlowStore
	flowConfig.Addr = server.Addr()
	flowConfig.Prefix = "test-flow"
	flowConfig.Hibernator = hibernator

	flowStore, err := tb.NewStore(flowConfig)
	assert.NoError(t, err)

	mockCli := helpers.NewMockClient()

	cfg := &config.Config{
		APIPort:            8080,
		APIHost:            "localhost",
		WebhookBaseURL:     "http://localhost:8080",
		StepTimeout:        5 * api.Second,
		FlowCacheSize:      100,
		ShutdownTimeout:    2 * time.Second,
		RetryCheckInterval: 100 * time.Millisecond,
		WorkConfig: api.WorkConfig{
			MaxRetries:   3,
			BackoffMs:    100,
			MaxBackoffMs: 1000,
			BackoffType:  api.BackoffTypeFixed,
		},
	}

	hub := tb.GetHub()
	eng := engine.New(engineStore, flowStore, mockCli, hub, cfg)

	cleanup := func() {
		_ = eng.Stop()
		_ = hibernator.Close()
		_ = tb.Close()
		server.Close()
	}

	return &testHibernationEnv{
		Engine:      eng,
		Redis:       server,
		MockClient:  mockCli,
		Hibernator:  hibernator,
		Cleanup:     cleanup,
		flowStore:   flowStore,
		engineStore: engineStore,
		timebox:     tb,
		hub:         hub,
	}
}

func TestHibernateCompletedFlow(t *testing.T) {
	env := newHibernationTestEnv(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep("hibernate-step")
	err := env.Engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	env.MockClient.SetResponse(step.ID, api.Args{"result": "done"})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	flowID := api.FlowID("hibernate-test")
	err = env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	// Wait for flow to complete
	assert.Eventually(t, func() bool {
		flow, flowErr := env.Engine.GetFlowState(ctx, flowID)
		return flowErr == nil && flow.Status == api.FlowCompleted
	}, 5*time.Second, 50*time.Millisecond)

	// Give time for hibernation to occur
	time.Sleep(200 * time.Millisecond)

	// Verify flow was hibernated by checking the hibernator directly
	flowKey := timebox.NewAggregateID("flow", timebox.ID(flowID))
	rec, err := env.Hibernator.Get(ctx, flowKey)
	assert.NoError(t, err)
	assert.NotNil(t, rec)
}

func TestRetrieveHibernatedFlow(t *testing.T) {
	env := newHibernationTestEnv(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep("retrieve-step")
	err := env.Engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	env.MockClient.SetResponse(step.ID, api.Args{"output": "value"})

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	flowID := api.FlowID("retrieve-test")
	err = env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	// Wait for flow to complete
	assert.Eventually(t, func() bool {
		flow, flowErr := env.Engine.GetFlowState(ctx, flowID)
		return flowErr == nil && flow.Status == api.FlowCompleted
	}, 5*time.Second, 50*time.Millisecond)

	// Give time for hibernation to occur
	time.Sleep(200 * time.Millisecond)

	// Verify it's in hibernation storage
	flowKey := timebox.NewAggregateID("flow", timebox.ID(flowID))
	rec, err := env.Hibernator.Get(ctx, flowKey)
	assert.NoError(t, err)
	assert.NotNil(t, rec)

	// Now retrieve the flow - it should be restored from hibernation
	flow, err := env.Engine.GetFlowState(ctx, flowID)
	assert.NoError(t, err)
	assert.Equal(t, flowID, flow.ID)
	assert.Equal(t, api.FlowCompleted, flow.Status)

	// Verify attributes are preserved
	attr, ok := flow.Attributes["output"]
	assert.True(t, ok)
	assert.Equal(t, "value", attr.Value)
}

func TestHibernateFailedFlow(t *testing.T) {
	env := newHibernationTestEnv(t)
	defer env.Cleanup()

	env.Engine.Start()
	ctx := context.Background()

	step := helpers.NewSimpleStep("fail-step")
	step.WorkConfig = &api.WorkConfig{MaxRetries: 0}
	err := env.Engine.RegisterStep(ctx, step)
	assert.NoError(t, err)

	env.MockClient.SetError(step.ID, api.ErrWorkNotCompleted)

	plan := &api.ExecutionPlan{
		Goals: []api.StepID{step.ID},
		Steps: api.Steps{step.ID: step},
	}

	flowID := api.FlowID("fail-hibernate-test")
	err = env.Engine.StartFlow(ctx, flowID, plan, api.Args{}, api.Metadata{})
	assert.NoError(t, err)

	// Wait for flow to fail
	assert.Eventually(t, func() bool {
		flow, flowErr := env.Engine.GetFlowState(ctx, flowID)
		return flowErr == nil && flow.Status == api.FlowFailed
	}, 5*time.Second, 50*time.Millisecond)

	// Give time for hibernation to occur
	time.Sleep(200 * time.Millisecond)

	// Verify failed flow was also hibernated
	flowKey := timebox.NewAggregateID("flow", timebox.ID(flowID))
	rec, err := env.Hibernator.Get(ctx, flowKey)
	assert.NoError(t, err)
	assert.NotNil(t, rec)

	// Retrieve and verify state
	flow, err := env.Engine.GetFlowState(ctx, flowID)
	assert.NoError(t, err)
	assert.Equal(t, api.FlowFailed, flow.Status)
}
