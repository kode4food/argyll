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

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/config"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/server"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

func TestStartStop(t *testing.T) {
	redis, err := miniredis.Run()
	assert.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	checker := server.NewHealthChecker(eng, tb.GetHub())
	assert.NotNil(t, checker)

	checker.Start()
	checker.Stop()
}

func TestGetStepHealth(t *testing.T) {
	redis, err := miniredis.Run()
	assert.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	step := helpers.NewSimpleStep("health-test-step")

	err = eng.RegisterStep(step)
	assert.NoError(t, err)

	engineState, err := eng.GetEngineState()
	assert.NoError(t, err)
	health, ok := engineState.Health["health-test-step"]
	assert.True(t, ok)
	assert.NotNil(t, health)
	assert.Equal(t, api.HealthUnknown, health.Status)
}

func TestGetStepHealthNotFound(t *testing.T) {
	redis, err := miniredis.Run()
	assert.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	engineState, err := eng.GetEngineState()
	assert.NoError(t, err)
	_, ok := engineState.Health["nonexistent-step"]
	assert.False(t, ok)
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
	assert.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

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
	}

	err = eng.RegisterStep(step)
	assert.NoError(t, err)

	checker := server.NewHealthChecker(eng, tb.GetHub())
	checker.Start()
	defer checker.Stop()

	engineState, err := eng.GetEngineState()
	assert.NoError(t, err)
	health, ok := engineState.Health["real-health-step"]
	assert.True(t, ok)
	assert.NotNil(t, health)
}

func TestRecentSuccess(t *testing.T) {
	redis, err := miniredis.Run()
	assert.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	step := helpers.NewSimpleStep("recent-success-step")

	err = eng.RegisterStep(step)
	assert.NoError(t, err)

	checker := server.NewHealthChecker(eng, tb.GetHub())
	checker.Start()
	defer checker.Stop()

	completedData, _ := json.Marshal(api.StepCompletedEvent{
		StepID: "recent-success-step",
		FlowID: "wf-test",
	})

	appendFlowEvent(t, flowStore, "wf-test", &timebox.Event{
		Type: timebox.EventType(api.EventTypeStepCompleted),
		Data: completedData,
	})

	engineState, err := eng.GetEngineState()
	assert.NoError(t, err)
	health, ok := engineState.Health["recent-success-step"]
	assert.True(t, ok)
	assert.NotNil(t, health)
}

func TestHealthCheckMarksUnhealthy(t *testing.T) {
	healthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer healthServer.Close()

	redis, err := miniredis.Run()
	assert.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	step := &api.Step{
		ID:   "unhealthy-step",
		Name: "Unhealthy Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint:    healthServer.URL + "/execute",
			HealthCheck: healthServer.URL + "/health",
		},
	}

	err = eng.RegisterStep(step)
	assert.NoError(t, err)

	consumer := tb.GetHub().NewConsumer()
	checker := server.NewHealthChecker(eng, tb.GetHub())
	checker.Start()
	defer checker.Stop()

	helpers.WaitForStepHealth(t,
		consumer, "unhealthy-step", api.HealthUnhealthy, 5*time.Second,
	)

	state, err := eng.GetEngineState()
	assert.NoError(t, err)
	health, ok := state.Health["unhealthy-step"]
	assert.True(t, ok)
	assert.Equal(t, api.HealthUnhealthy, health.Status)
}

func TestEventLoopUnmarshalError(t *testing.T) {
	redis, err := miniredis.Run()
	assert.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	checker := server.NewHealthChecker(eng, tb.GetHub())
	checker.Start()
	defer checker.Stop()

	invalidEvent := &timebox.Event{
		Type: timebox.EventType(api.EventTypeStepCompleted),
		Data: []byte(`{"step_id":123}`),
	}

	appendFlowEvent(t, flowStore, "wf-test", invalidEvent)
}

func TestCheckMultipleHTTPSteps(t *testing.T) {
	healthServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer healthServer.Close()

	redis, err := miniredis.Run()
	assert.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	for i := 0; i < 3; i++ {
		step := &api.Step{
			ID:   api.StepID("multi-health-" + string(rune('a'+i))),
			Name: "Multi Health Step",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint:    healthServer.URL + "/execute",
				HealthCheck: healthServer.URL + "/health",
			},
		}

		err = eng.RegisterStep(step)
		assert.NoError(t, err)
	}

	checker := server.NewHealthChecker(eng, tb.GetHub())
	checker.Start()
	defer checker.Stop()

	engineState, err := eng.GetEngineState()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(engineState.Health), 3)
}

func TestNonStepCompletedEvent(t *testing.T) {
	redis, err := miniredis.Run()
	assert.NoError(t, err)
	defer redis.Close()

	cfg := config.NewDefaultConfig()
	cfg.EngineStore.Addr = redis.Addr()
	cfg.FlowStore.Addr = redis.Addr()

	tb, err := timebox.NewTimebox(timebox.Config{
		MaxRetries: timebox.DefaultMaxRetries,
		CacheSize:  100,
		Workers:    true,
	})
	assert.NoError(t, err)
	defer func() { _ = tb.Close() }()

	engineStore, err := tb.NewStore(cfg.EngineStore)
	assert.NoError(t, err)

	flowStore, err := tb.NewStore(cfg.FlowStore)
	assert.NoError(t, err)

	mockClient := helpers.NewMockClient()
	eng := engine.New(engineStore, flowStore, mockClient, tb.GetHub(), cfg)

	checker := server.NewHealthChecker(eng, tb.GetHub())
	checker.Start()
	defer checker.Stop()

	event := &timebox.Event{
		Type: timebox.EventType(api.EventTypeFlowStarted),
		Data: []byte("{}"),
	}

	appendFlowEvent(t, flowStore, "wf-test", event)
}

func appendFlowEvent(
	t *testing.T, store *timebox.Store, flowID api.FlowID, event *timebox.Event,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	aggregateID := timebox.NewAggregateID(
		events.FlowPrefix, timebox.ID(flowID),
	)
	eventsInStore, err := store.GetEvents(ctx, aggregateID, 0)
	assert.NoError(t, err)

	event.AggregateID = aggregateID
	event.Sequence = int64(len(eventsInStore))
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	err = store.AppendEvents(ctx, aggregateID, event.Sequence, []*timebox.Event{
		event,
	})
	assert.NoError(t, err)
}
