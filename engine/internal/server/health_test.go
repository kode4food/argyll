package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kode4food/timebox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kode4food/spuds/engine/internal/assert/helpers"
	"github.com/kode4food/spuds/engine/internal/config"
	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/internal/server"
	"github.com/kode4food/spuds/engine/pkg/api"
)

func TestStartStop(t *testing.T) {
	redis, err := miniredis.Run()
	require.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
	})
	require.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	require.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	require.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	checker := server.NewHealthChecker(eng, tb.GetHub())
	assert.NotNil(t, checker)

	checker.Start()
	checker.Stop()
}

func TestGetStepHealth(t *testing.T) {
	redis, err := miniredis.Run()
	require.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
	})
	require.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	require.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	require.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	step := helpers.NewSimpleStep("health-test-step")

	err = eng.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	engineState, err := eng.GetEngineState(context.Background())
	require.NoError(t, err)
	health, ok := engineState.Health["health-test-step"]
	require.True(t, ok, "expected step health to exist")
	assert.NotNil(t, health)
	assert.Equal(t, api.HealthUnknown, health.Status)
}

func TestGetStepHealthNotFound(t *testing.T) {
	redis, err := miniredis.Run()
	require.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
	})
	require.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	require.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	require.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	engineState, err := eng.GetEngineState(context.Background())
	require.NoError(t, err)
	_, ok := engineState.Health["nonexistent-step"]
	assert.False(t, ok, "expected step health not to exist")
}

func TestWithRealHealthCheck(t *testing.T) {
	healthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(
				map[string]string{"status": "healthy"},
			)
		}),
	)
	defer healthServer.Close()

	redis, err := miniredis.Run()
	require.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
	})
	require.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	require.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	require.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	step := &api.Step{
		ID:   "real-health-step",
		Name: "Real Health Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint:    healthServer.URL + "/execute",
			HealthCheck: healthServer.URL + "/health",
		},
		Version: "1.0.0",
	}

	err = eng.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	checker := server.NewHealthChecker(eng, tb.GetHub())
	checker.Start()
	defer checker.Stop()

	engineState, err := eng.GetEngineState(context.Background())
	require.NoError(t, err)
	health, ok := engineState.Health["real-health-step"]
	require.True(t, ok, "expected step health to exist")
	assert.NotNil(t, health)
}

func TestRecentSuccess(t *testing.T) {
	redis, err := miniredis.Run()
	require.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
	})
	require.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	require.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	require.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	step := helpers.NewSimpleStep("recent-success-step")

	err = eng.RegisterStep(context.Background(), step)
	require.NoError(t, err)

	checker := server.NewHealthChecker(eng, tb.GetHub())
	checker.Start()
	defer checker.Stop()

	producer := tb.GetHub().NewProducer()
	defer producer.Close()

	completedData, _ := json.Marshal(api.StepCompletedEvent{
		StepID: "recent-success-step",
		FlowID: "wf-test",
	})

	event := &timebox.Event{
		Type:        api.EventTypeStepCompleted,
		AggregateID: timebox.NewAggregateID("flow", "wf-test"),
		Timestamp:   time.Now(),
		Data:        completedData,
	}

	producer.Send() <- event

	engineState, err := eng.GetEngineState(context.Background())
	require.NoError(t, err)
	health, ok := engineState.Health["recent-success-step"]
	require.True(t, ok, "expected step health to exist")
	assert.NotNil(t, health)
}
