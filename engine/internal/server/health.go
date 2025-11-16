package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kode4food/caravan/topic"
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

// HealthChecker monitors the health of registered step services
type HealthChecker struct {
	engine      *engine.Engine
	eventHub    timebox.EventHub
	ctx         context.Context
	cancel      context.CancelFunc
	client      *http.Client
	consumer    topic.Consumer[*timebox.Event]
	lastSuccess map[timebox.ID]time.Time
	mu          sync.RWMutex
}

const (
	successWindow       = 60 * time.Second
	healthCheckTimeout  = 3 * time.Second
	healthCheckInterval = 30 * time.Second
	httpErrorThreshold  = 400
)

// NewHealthChecker creates a health checker that periodically monitors HTTP
// step service availability and updates their health status
func NewHealthChecker(eng *engine.Engine, hub timebox.EventHub) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		engine:      eng,
		eventHub:    hub,
		ctx:         ctx,
		cancel:      cancel,
		consumer:    hub.NewConsumer(),
		lastSuccess: map[timebox.ID]time.Time{},
		client: &http.Client{
			Timeout: healthCheckTimeout,
		},
	}
}

func (h *HealthChecker) Start() {
	go h.healthCheckLoop()
	go h.eventLoop()
}

func (h *HealthChecker) Stop() {
	h.cancel()
	h.consumer.Close()
}

func (h *HealthChecker) GetStepHealth(stepID timebox.ID) (*api.HealthState, error) {
	state, err := h.engine.GetEngineState(context.Background())
	if err != nil {
		return nil, err
	}

	health, ok := state.Health[stepID]
	if !ok {
		return nil, fmt.Errorf("step health not found: %s", stepID)
	}

	return health, nil
}

func (h *HealthChecker) eventLoop() {
	for {
		select {
		case <-h.ctx.Done():
			return

		case event, ok := <-h.consumer.Receive():
			if !ok {
				return
			}
			h.handleStepCompleted(event)
		}
	}
}

func (h *HealthChecker) handleStepCompleted(event *timebox.Event) {
	if event.Type != api.EventTypeStepCompleted {
		return
	}

	var sc api.StepCompletedEvent
	if err := json.Unmarshal(event.Data, &sc); err != nil {
		slog.Error("Failed to unmarshal event",
			slog.Any("error", err))
		return
	}

	h.mu.Lock()
	h.lastSuccess[sc.StepID] = time.Now()
	h.mu.Unlock()
}

func (h *HealthChecker) healthCheckLoop() {
	slog.Info("Health checker started")
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	h.checkAllSteps()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.checkAllSteps()
		}
	}
}

func (h *HealthChecker) checkAllSteps() {
	engState, err := h.engine.GetEngineState(h.ctx)
	if err != nil {
		slog.Error("Failed to get engine state",
			slog.Any("error", err))
		return
	}

	var httpSteps []*api.Step
	for _, step := range engState.Steps {
		if step.HTTP != nil && step.HTTP.HealthCheck != "" {
			httpSteps = append(httpSteps, step)
		}
	}
	httpCount := len(httpSteps)

	var delay time.Duration
	if httpCount > 1 {
		delay = healthCheckInterval / time.Duration(httpCount)
	}

	for _, step := range httpSteps {
		h.checkStepHealth(step)
		if delay > 0 {
			time.Sleep(delay)
		}
	}
}

func (h *HealthChecker) checkStepHealth(step *api.Step) {
	h.mu.RLock()
	lastSuccess, hasRecent := h.lastSuccess[step.ID]
	h.mu.RUnlock()

	if hasRecent && time.Since(lastSuccess) < successWindow {
		_ = h.engine.UpdateStepHealth(h.ctx, step.ID, api.HealthHealthy, "")
		return
	}

	status := api.HealthHealthy
	errorMsg := ""

	resp, err := h.client.Get(step.HTTP.HealthCheck)
	if err != nil {
		status = api.HealthUnhealthy
		errorMsg = err.Error()
		slog.Error("Health check failed",
			slog.Any("step_id", step.ID),
			slog.String("error", err.Error()))
		_ = h.engine.UpdateStepHealth(h.ctx, step.ID, status, errorMsg)
		return
	}

	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= httpErrorThreshold {
		status = api.HealthUnhealthy
		errorMsg = "HTTP " + resp.Status
		slog.Error("Health check failed",
			slog.Any("step_id", step.ID),
			slog.String("status", resp.Status))
	}

	_ = h.engine.UpdateStepHealth(h.ctx, step.ID, status, errorMsg)
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, api.HealthResponse{
		Service: "spuds-engine",
		Version: "1.0.0",
		HealthState: api.HealthState{
			Status: api.HealthHealthy,
		},
	})
}

func (s *Server) handleEngineHealth(c *gin.Context) {
	engState, err := s.engine.GetEngineState(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetEngineState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, api.HealthListResponse{
		Health: engState.Health,
		Count:  len(engState.Health),
	})
}

func (s *Server) handleEngineHealthByID(c *gin.Context) {
	stepID := timebox.ID(c.Param("stepID"))

	engState, err := s.engine.GetEngineState(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetEngineState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	health, ok := engState.Health[stepID]
	if !ok {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("Health not found for step: %s", stepID),
			Status: http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, health)
}
