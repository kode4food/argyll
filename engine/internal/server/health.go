package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

// HealthChecker monitors the health of registered step services
type HealthChecker struct {
	engine *engine.Engine
	ctx    context.Context
	cancel context.CancelFunc
	client *http.Client
}

const (
	healthCheckTimeout  = 3 * time.Second
	healthCheckInterval = 30 * time.Second
	httpErrorThreshold  = 400
	roleUnknown         = "unknown"
)

// NewHealthChecker creates a health checker that periodically monitors HTTP
// step service availability and updates their health status
func NewHealthChecker(eng *engine.Engine) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		engine: eng,
		ctx:    ctx,
		cancel: cancel,
		client: &http.Client{
			Timeout: healthCheckTimeout,
		},
	}
}

// Start begins the health check loop and event processing
func (h *HealthChecker) Start() {
	go h.healthCheckLoop()
}

// Stop gracefully shuts down the health checker
func (h *HealthChecker) Stop() {
	h.cancel()
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

	health := make(map[api.StepID]api.HealthState, len(cat.Steps))

	var httpSteps []*api.Step
	for _, step := range cat.Steps {
		switch step.Type {
		case api.StepTypeScript:
			h.updateScriptHealth(step, health)
			h.updateFlowSteps(cat, health)
		case api.StepTypeSync, api.StepTypeAsync:
			if step.HTTP != nil && step.HTTP.HealthCheck != "" {
				httpSteps = append(httpSteps, step)
			}
		}
	}

	httpCount := len(httpSteps)
	var delay time.Duration
	if httpCount > 1 {
		delay = healthCheckInterval / time.Duration(httpCount)
	}
	for _, step := range httpSteps {
		h.updateStepHealth(step, health)
		h.updateFlowSteps(cat, health)
		if delay > 0 {
			time.Sleep(delay)
		}
	}
}

func (h *HealthChecker) updateScriptHealth(
	step *api.Step, health map[api.StepID]api.HealthState,
) {
	status := api.HealthHealthy
	errMsg := ""
	if err := h.engine.VerifyScript(step); err != nil {
		status = api.HealthUnhealthy
		errMsg = err.Error()
	}
	health[step.ID] = api.HealthState{
		Status: status,
		Error:  errMsg,
	}
	if err := h.engine.UpdateStepHealth(step.ID, status, errMsg); err != nil {
		slog.Error("Failed to update script health",
			log.StepID(step.ID), log.Error(err))
	}
}

func (h *HealthChecker) updateFlowSteps(
	cat api.CatalogState, health map[api.StepID]api.HealthState,
) {
	resolved := engine.ResolveHealth(cat, health)
	for stepID, step := range cat.Steps {
		if step.Type != api.StepTypeFlow {
			continue
		}
		stepHealth, ok := resolved[stepID]
		if !ok {
			continue
		}
		health[stepID] = stepHealth
		if err := h.engine.UpdateStepHealth(
			stepID, stepHealth.Status, stepHealth.Error,
		); err != nil {
			slog.Error("Failed to update flow step health",
				log.StepID(stepID), log.Error(err))
		}
	}
}

func (h *HealthChecker) updateStepHealth(
	step *api.Step, health map[api.StepID]api.HealthState,
) {
	status := api.HealthHealthy
	errorMsg := ""

	resp, err := h.client.Get(step.HTTP.HealthCheck)
	if err != nil {
		status = api.HealthUnhealthy
		errorMsg = err.Error()
		slog.Error("Health check failed",
			log.StepID(step.ID),
			log.Error(err))
		health[step.ID] = api.HealthState{
			Status: status,
			Error:  errorMsg,
		}
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

	health[step.ID] = api.HealthState{
		Status: status,
		Error:  errorMsg,
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
	cat, err := s.engine.GetCatalogState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetCatalogState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}
	cluster, err := s.engine.GetClusterState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetNodeState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, completeClusterHealth(cat, cluster))
}

func (s *Server) handleEngineHealthByID(c *gin.Context) {
	stepID := api.StepID(c.Param("stepID"))

	cat, err := s.engine.GetCatalogState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetCatalogState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}
	cluster, err := s.engine.GetClusterState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetNodeState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	merged := engine.ResolveHealth(cat, engine.MergeNodeHealth(cluster))
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
