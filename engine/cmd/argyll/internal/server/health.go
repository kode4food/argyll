package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kode4food/timebox/raft"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

// HealthChecker monitors the health of registered step services
type HealthChecker struct {
	engine  *engine.Engine
	workers builtins.AgentWorkers
	ctx     context.Context
	cancel  context.CancelFunc
	client  *http.Client
}

const (
	healthCheckTimeout  = 3 * time.Second
	healthCheckInterval = 30 * time.Second
	httpErrorThreshold  = 400
	roleUnknown         = "unknown"
)

var (
	ErrGetStepHealth = errors.New("failed to get step health")
)

// NewHealthChecker creates a health checker that periodically monitors HTTP step service and agent
// worker availability and updates their health status
func NewHealthChecker(eng *engine.Engine, workers builtins.AgentWorkers) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		engine:  eng,
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
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

	probed := map[api.StepID]api.HealthState{}
	endpoints := map[string][]api.StepID{}
	for _, st := range cat.Steps {
		if !probesHealth(st) {
			continue
		}
		url, err := h.healthURL(st)
		if err != nil {
			probed[st.ID] = api.HealthState{
				Status: api.HealthUnhealthy,
				Error:  err.Error(),
			}
			continue
		}
		endpoints[url] = append(endpoints[url], st.ID)
	}

	// The engine derives script and flow step health on its own, so a pass with nothing probed
	// still refreshes those
	h.refresh(probed)

	var delay time.Duration
	if len(endpoints) > 1 {
		delay = healthCheckInterval / time.Duration(len(endpoints))
	}
	for url, ids := range endpoints {
		health := h.probe(ids[0], url)
		for _, sid := range ids {
			probed[sid] = health
		}
		h.refresh(probed)
		if delay > 0 {
			time.Sleep(delay)
		}
	}
}

func (h *HealthChecker) refresh(probed map[api.StepID]api.HealthState) {
	if err := h.engine.RefreshStepHealth(probed); err != nil {
		slog.Error("Failed to refresh step health", log.Error(err))
	}
}

func (h *HealthChecker) probe(sid api.StepID, url string) api.HealthState {
	resp, err := h.client.Get(url)
	if err != nil {
		slog.Error("Health check failed", log.StepID(sid), log.Error(err))
		return api.HealthState{
			Status: api.HealthUnhealthy,
			Error:  err.Error(),
		}
	}

	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= httpErrorThreshold {
		slog.Error("Health check failed", log.StepID(sid), log.Status(resp.Status))
		return api.HealthState{
			Status: api.HealthUnhealthy,
			Error:  "HTTP " + resp.Status,
		}
	}
	return api.HealthState{Status: api.HealthHealthy}
}

// healthURL returns where a step's health is probed: its service's health endpoint, or the health
// endpoint of the worker serving its agent provider
func (h *HealthChecker) healthURL(st *api.Step) (string, error) {
	if st.Type != api.StepTypeAgent {
		return st.HTTP.Health, nil
	}
	base, ok := h.workers.URL(st.Agent.Provider)
	if !ok {
		return "", fmt.Errorf("%w: %s", builtins.ErrNoAgentWorker, st.Agent.Provider)
	}
	return base + builtins.AgentHealthPath, nil
}

func probesHealth(st *api.Step) bool {
	switch st.Type {
	case api.StepTypeService:
		return st.HTTP != nil && st.HTTP.Health != ""
	case api.StepTypeAgent:
		return st.Agent != nil
	default:
		return false
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	st := s.raftState()
	c.Header("X-Argyll-Raft-State", st)
	c.JSON(http.StatusOK, api.HealthResponse{
		Service: "argyll-engine",
		Details: s.statusDetails(),
		Status:  api.HealthHealthy,
	})
}

func (s *Server) handleEngineHealth(c *gin.Context) {
	cat, err := s.engine.GetCatalogState()
	if writeError(c, writeErrorArgs{
		sentinel: ErrGetCatalogState,
		err:      err,
	}) {
		return
	}
	cluster, err := s.engine.GetClusterState()
	if writeError(c, writeErrorArgs{
		sentinel: ErrGetClusterState,
		err:      err,
	}) {
		return
	}
	c.JSON(http.StatusOK, completeClusterHealth(cat, cluster))
}

func (s *Server) handleEngineHealthByID(c *gin.Context) {
	sid := sanitizedParam[api.StepID](c, "step_id")

	health, err := s.engine.GetStepHealth(sid)
	if errors.Is(err, api.ErrStepNotFound) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusNotFound,
		})
		return
	}
	if writeError(c, writeErrorArgs{
		sentinel: ErrGetStepHealth,
		err:      err,
	}) {
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
