package server

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

var (
	ErrListFlows           = errors.New("failed to list flows")
	ErrGetFlow             = errors.New("failed to get flow")
	ErrCreateExecutionPlan = errors.New("failed to create execution plan")
)

var invalidFlowIDChars = regexp.MustCompile(`[^a-zA-Z0-9_.\-+ ]`)

func (s *Server) listFlows(c *gin.Context) {
	flows, err := s.engine.ListFlows(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrListFlows, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, api.FlowsListResponse{
		Flows: flows,
		Count: len(flows),
	})
}

func (s *Server) startFlow(c *gin.Context) {
	var req api.CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return
	}

	flowID := sanitizeFlowID(string(req.ID))
	if flowID == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Valid Flow ID is required",
			Status: http.StatusBadRequest,
		})
		return
	}

	if len(req.Goals) == 0 {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "At least one goal step ID is required",
			Status: http.StatusBadRequest,
		})
		return
	}

	plan, ok := s.createPlan(c, req.Goals, req.Init)
	if !ok {
		return
	}

	meta := api.Metadata{}
	err := s.engine.StartFlow(
		c.Request.Context(), flowID, plan, req.Init, meta,
	)
	if err == nil {
		c.JSON(http.StatusCreated, api.FlowStartedResponse{
			FlowID: flowID,
		})
		return
	}

	if errors.Is(err, engine.ErrFlowExists) {
		c.JSON(http.StatusConflict, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", err.Error(), flowID),
			Status: http.StatusConflict,
		})
		return
	}
	c.JSON(http.StatusBadRequest, api.ErrorResponse{
		Error:  err.Error(),
		Status: http.StatusBadRequest,
	})
}

func (s *Server) getFlow(c *gin.Context) {
	flowID := api.FlowID(c.Param("flowID"))

	flow, err := s.engine.GetFlowState(c.Request.Context(), flowID)
	if err == nil {
		c.JSON(http.StatusOK, flow)
		return
	}

	if errors.Is(err, engine.ErrFlowNotFound) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", err.Error(), flowID),
			Status: http.StatusNotFound,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrGetFlow, err),
		Status: http.StatusInternalServerError,
	})
}

func (s *Server) createPlan(
	c *gin.Context, goalStepIDs []api.StepID, initialState api.Args,
) (*api.ExecutionPlan, bool) {
	engState, err := s.engine.GetEngineState(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetEngineState, err),
			Status: http.StatusInternalServerError,
		})
		return nil, false
	}

	plan, err := s.engine.CreateExecutionPlan(
		engState, goalStepIDs, initialState,
	)
	if err == nil {
		return plan, true
	}

	if errors.Is(err, engine.ErrStepNotFound) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", err.Error(), goalStepIDs),
			Status: http.StatusNotFound,
		})
		return nil, false
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrCreateExecutionPlan, err),
		Status: http.StatusInternalServerError,
	})
	return nil, false
}

func (s *Server) handlePlanPreview(c *gin.Context) {
	var req api.ExecutionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return
	}

	if len(req.Goals) == 0 {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "At least one goal step ID is required",
			Status: http.StatusBadRequest,
		})
		return
	}

	if plan, ok := s.createPlan(c, req.Goals, req.Init); ok {
		c.JSON(http.StatusOK, plan)
	}
}

func sanitizeFlowID(id string) api.FlowID {
	id = strings.ToLower(id)
	sanitized := invalidFlowIDChars.ReplaceAllString(id, "")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	return api.FlowID(strings.Trim(sanitized, "-"))
}
