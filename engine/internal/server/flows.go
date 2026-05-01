package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const (
	MaxFlowBodyBytes = 1 * 1024 * 1024 // 1 MB
	MaxQueryLimit    = 1000
)

const errLimitTooHigh = "Limit must be between 0 and %d"

var (
	ErrQueryFlows          = errors.New("failed to query flows")
	ErrGetFlow             = errors.New("failed to get flow")
	ErrGetFlowStatus       = errors.New("failed to get flow status")
	ErrCreateExecutionPlan = errors.New("failed to create execution plan")
	ErrStartFlow           = errors.New("failed to start flow")
)

func (s *Server) queryFlows(c *gin.Context) {
	var req api.QueryFlowsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return
	}

	if req.IDPrefix != "" && api.InvalidIDChars.MatchString(req.IDPrefix) {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Invalid ID prefix",
			Status: http.StatusBadRequest,
		})
		return
	}

	if req.Limit < 0 || req.Limit > MaxQueryLimit {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf(errLimitTooHigh, MaxQueryLimit),
			Status: http.StatusBadRequest,
		})
		return
	}

	if req.Sort != "" && req.Sort != api.FlowSortRecentDesc &&
		req.Sort != api.FlowSortRecentAsc {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Invalid sort",
			Status: http.StatusBadRequest,
		})
		return
	}

	for key, value := range req.Labels {
		if key == "" || value == "" {
			c.JSON(http.StatusBadRequest, api.ErrorResponse{
				Error:  "Invalid labels",
				Status: http.StatusBadRequest,
			})
			return
		}
	}

	if !validFlowStatuses(req.Statuses) {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Invalid statuses",
			Status: http.StatusBadRequest,
		})
		return
	}

	resp, err := s.engine.QueryFlows(&req)
	if err != nil {
		if errors.Is(err, engine.ErrInvalidFlowCursor) {
			c.JSON(http.StatusBadRequest, api.ErrorResponse{
				Error:  err.Error(),
				Status: http.StatusBadRequest,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrQueryFlows, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) listFlows(c *gin.Context) {
	resp, err := s.engine.ListFlows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrQueryFlows, err),
			Status: http.StatusInternalServerError,
		})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func validFlowStatuses(statuses []api.FlowStatus) bool {
	if len(statuses) == 0 {
		return true
	}
	for _, status := range statuses {
		if status != api.FlowActive &&
			status != api.FlowCompleted &&
			status != api.FlowFailed {
			return false
		}
	}
	return true
}

func (s *Server) startFlow(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(
		c.Writer, c.Request.Body, MaxFlowBodyBytes,
	)
	var req api.CreateFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return
	}

	req.ID = api.SanitizeID(req.ID)
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		})
		return
	}

	pl, ok := s.createExecutionPlan(c, req.Goals, req.Init)
	if !ok {
		return
	}

	var apps []flow.Applier
	if req.Init != nil {
		apps = append(apps, flow.WithInit(req.Init))
	}
	if len(req.Labels) > 0 {
		apps = append(apps, flow.WithLabels(req.Labels))
	}
	err := s.engine.StartFlow(req.ID, pl, apps...)
	if err == nil {
		c.JSON(http.StatusCreated, api.FlowStartedResponse{
			FlowID: req.ID,
		})
		return
	}

	if errors.Is(err, engine.ErrFlowExists) {
		c.JSON(http.StatusConflict, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", err.Error(), req.ID),
			Status: http.StatusConflict,
		})
		return
	}
	if errors.Is(err, api.ErrRequiredInputs) {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrStartFlow, err),
		Status: http.StatusInternalServerError,
	})
}

func (s *Server) getFlow(c *gin.Context) {
	id := api.FlowID(c.Param("flowID"))

	fl, err := s.engine.GetFlowState(id)
	if err == nil {
		c.JSON(http.StatusOK, fl)
		return
	}

	if errors.Is(err, engine.ErrFlowNotFound) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", err.Error(), id),
			Status: http.StatusNotFound,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrGetFlow, err),
		Status: http.StatusInternalServerError,
	})
}

func (s *Server) getFlowStatus(c *gin.Context) {
	id := api.FlowID(c.Param("flowID"))

	status, err := s.engine.GetFlowStatus(id)
	if err == nil {
		c.JSON(http.StatusOK, api.FlowStatusResponse{
			ID:     id,
			Status: status,
		})
		return
	}

	if errors.Is(err, engine.ErrFlowNotFound) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", err.Error(), id),
			Status: http.StatusNotFound,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrGetFlowStatus, err),
		Status: http.StatusInternalServerError,
	})
}

func (s *Server) createExecutionPlan(
	c *gin.Context, goals []api.StepID, init api.InitArgs,
) (*api.ExecutionPlan, bool) {
	return s.createPlan(c, goals, init, plan.Create)
}

func (s *Server) createPreviewPlan(
	c *gin.Context, goals []api.StepID, init api.InitArgs,
) (*api.ExecutionPlan, bool) {
	return s.createPlan(c, goals, init, plan.Preview)
}

func (s *Server) createPlan(
	c *gin.Context, goals []api.StepID, init api.InitArgs, planner plan.Planner,
) (*api.ExecutionPlan, bool) {
	cat, err := s.engine.GetCatalogState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetCatalogState, err),
			Status: http.StatusInternalServerError,
		})
		return nil, false
	}

	pl, err := planner(cat, goals, init)
	if err == nil {
		return pl, true
	}

	if errors.Is(err, plan.ErrStepNotFound) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", err.Error(), goals),
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

	if pl, ok := s.createPreviewPlan(c, req.Goals, req.Init); ok {
		c.JSON(http.StatusOK, pl)
	}
}
