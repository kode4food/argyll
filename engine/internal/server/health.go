package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kode4food/timebox"
	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// HealthChecker monitors the health of registered step services
	HealthChecker struct {
		engine      *engine.Engine
		ctx         context.Context
		cancel      context.CancelFunc
		client      *http.Client
		consumer    *event.Consumer
		lastSuccess map[api.StepID]time.Time
		mu          sync.RWMutex
	}

	resolvedShardHealth struct {
		byNode      map[api.NodeID]map[api.StepID]*api.HealthState
		lastUpdated time.Time
	}
)

const (
	successWindow       = 60 * time.Second
	healthCheckTimeout  = 3 * time.Second
	healthCheckInterval = 30 * time.Second
	httpErrorThreshold  = 400
	roleUnknown         = "unknown"
)

// NewHealthChecker creates a health checker that periodically monitors HTTP
// step service availability and updates their health status
func NewHealthChecker(eng *engine.Engine, hub *event.Hub) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		engine: eng,
		ctx:    ctx,
		cancel: cancel,
		consumer: hub.NewTypeConsumer(
			timebox.EventType(api.EventTypeStepCompleted),
		),
		lastSuccess: map[api.StepID]time.Time{},
		client: &http.Client{
			Timeout: healthCheckTimeout,
		},
	}
}

// Start begins the health check loop and event processing
func (h *HealthChecker) Start() {
	go h.healthCheckLoop()
	go h.eventLoop()
}

// Stop gracefully shuts down the health checker
func (h *HealthChecker) Stop() {
	h.cancel()
	h.consumer.Close()
}

func (h *HealthChecker) eventLoop() {
	for {
		select {
		case <-h.ctx.Done():
			return

		case ev, ok := <-h.consumer.Receive():
			if !ok {
				return
			}
			sc, err := timebox.GetEventValue[api.StepCompletedEvent](ev)
			if err != nil {
				slog.Error("Failed to unmarshal step completed event",
					log.Error(err))
				continue
			}
			h.handleStepCompleted(sc)
		}
	}
}

func (h *HealthChecker) handleStepCompleted(sc api.StepCompletedEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastSuccess[sc.StepID] = time.Now()
}

func (h *HealthChecker) getLastSuccess(stepID api.StepID) (time.Time, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ts, ok := h.lastSuccess[stepID]
	return ts, ok
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
	cat, err := h.engine.GetCatalogState()
	if err != nil {
		slog.Error("Failed to get catalog state", log.Error(err))
		return
	}

	var httpSteps []*api.Step
	for _, step := range cat.Steps {
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
	lastSuccess, ok := h.getLastSuccess(step.ID)

	if ok && time.Since(lastSuccess) < successWindow {
		err := h.engine.UpdateStepHealth(step.ID, api.HealthHealthy, "")
		if err != nil {
			slog.Error("Failed to update step health",
				log.StepID(step.ID),
				log.Error(err))
		}
		return
	}

	status := api.HealthHealthy
	errorMsg := ""

	resp, err := h.client.Get(step.HTTP.HealthCheck)
	if err != nil {
		status = api.HealthUnhealthy
		errorMsg = err.Error()
		slog.Error("Health check failed",
			log.StepID(step.ID),
			log.Error(err))
		err := h.engine.UpdateStepHealth(step.ID, status, errorMsg)
		if err != nil {
			slog.Error("Failed to update step health",
				log.StepID(step.ID),
				log.Error(err))
		}
		return
	}

	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= httpErrorThreshold {
		status = api.HealthUnhealthy
		errorMsg = "HTTP " + resp.Status
		slog.Error("Health check failed",
			log.StepID(step.ID),
			log.Status(resp.Status))
	}

	if err := h.engine.UpdateStepHealth(step.ID, status, errorMsg); err != nil {
		slog.Error("Failed to update step health",
			log.StepID(step.ID),
			log.Error(err))
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	st := s.raftState()
	c.Header("X-Argyll-Raft-State", st)
	c.JSON(http.StatusOK, api.HealthResponse{
		Service: "argyll-engine",
		Details: s.statusDetails(),
		HealthState: api.HealthState{
			Status: api.HealthHealthy,
		},
	})
}

func (s *Server) handleEngineHealth(c *gin.Context) {
	res, err := resolveEngineHealth(s.engine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetNodeState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, api.HealthListResponse{
		Health: res.byNode,
	})
}

func (s *Server) handleEngineHealthByID(c *gin.Context) {
	stepID := api.StepID(c.Param("stepID"))

	res, err := resolveEngineHealth(s.engine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetNodeState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	merged := mergedShardHealth(res.byNode)
	health, ok := merged[stepID]
	if !ok {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("Health not found for step: %s", stepID),
			Status: http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, health)
}

func resolveEngineHealth(
	eng *engine.Engine,
) (*resolvedShardHealth, error) {
	cat, err := eng.GetCatalogState()
	if err != nil {
		return nil, err
	}
	nodeByID, err := eng.GetShardNodeStates()
	if err != nil {
		return nil, err
	}
	byNode := make(
		map[api.NodeID]map[api.StepID]*api.HealthState, len(nodeByID),
	)
	for id, node := range nodeByID {
		byNode[id] = engine.ResolveHealth(cat, node.Health)
	}
	return &resolvedShardHealth{
		byNode:      byNode,
		lastUpdated: latestNodeUpdate(nodeByID),
	}, nil
}

func mergedShardHealth(
	byNode map[api.NodeID]map[api.StepID]*api.HealthState,
) map[api.StepID]*api.HealthState {
	return engine.MergeNodeHealth(byNode)
}

func latestNodeUpdate(
	nodeByID map[api.NodeID]*api.NodeState,
) time.Time {
	var last time.Time
	for _, node := range nodeByID {
		if node != nil && node.LastSeen.After(last) {
			last = node.LastSeen
		}
	}
	return last
}

func (s *Server) raftState() string {
	details := s.statusDetails()
	if details == nil {
		return roleUnknown
	}
	backend, ok := details["backend"].(map[string]any)
	if !ok {
		return roleUnknown
	}
	st, ok := backend["state"].(raft.State)
	if !ok {
		return roleUnknown
	}
	return string(st)
}
