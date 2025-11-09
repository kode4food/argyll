package server

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

var (
	ErrListWorkflows       = errors.New("failed to list workflows")
	ErrGetWorkflow         = errors.New("failed to get workflow")
	ErrCreateExecutionPlan = errors.New("failed to create execution plan")
)

var invalidWorkflowIDChars = regexp.MustCompile(`[^a-zA-Z0-9_.\-+ ]`)

func (s *Server) listWorkflows(c *gin.Context) {
	flows, err := s.engine.ListWorkflows(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrListWorkflows, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, api.WorkflowsListResponse{
		Workflows: flows,
		Count:     len(flows),
	})
}

func (s *Server) startWorkflow(c *gin.Context) {
	var req api.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return
	}

	flowID := timebox.ID(sanitizeWorkflowID(string(req.ID)))
	if flowID == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Valid Workflow ID is required",
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
	err := s.engine.StartWorkflow(
		c.Request.Context(), flowID, plan, req.Init, meta,
	)
	if err == nil {
		c.JSON(http.StatusCreated, api.WorkflowStartedResponse{
			FlowID: flowID,
		})
		return
	}

	if existsError(err) {
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

func (s *Server) getWorkflow(c *gin.Context) {
	flowID := timebox.ID(c.Param("flowID"))

	flow, err := s.engine.GetWorkflowState(c.Request.Context(), flowID)
	if err == nil {
		c.JSON(http.StatusOK, flow)
		return
	}

	if isNotFoundError(err) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", err.Error(), flowID),
			Status: http.StatusNotFound,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrGetWorkflow, err),
		Status: http.StatusInternalServerError,
	})
}

func (s *Server) createPlan(
	c *gin.Context, goalStepIDs []timebox.ID, initialState api.Args,
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

	if isNotFoundError(err) {
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

func sanitizeWorkflowID(id string) string {
	id = strings.ToLower(id)
	sanitized := invalidWorkflowIDChars.ReplaceAllString(id, "")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	return strings.Trim(sanitized, "-")
}
