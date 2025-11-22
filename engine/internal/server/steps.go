package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

var (
	ErrInvalidJSON    = errors.New("invalid JSON request")
	ErrListSteps      = errors.New("failed to list steps")
	ErrRegisterStep   = errors.New("failed to register step")
	ErrUnregisterStep = errors.New("failed to unregister step")
)

func (s *Server) listSteps(c *gin.Context) {
	steps, err := s.engine.ListSteps(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrListSteps, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, api.StepsListResponse{
		Steps: steps,
		Count: len(steps),
	})
}

func (s *Server) createStep(c *gin.Context) {
	var step api.Step
	if err := c.ShouldBindJSON(&step); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return
	}

	if err := step.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		})
		return
	}

	err := s.engine.RegisterStep(c.Request.Context(), &step)
	if err == nil {
		c.JSON(http.StatusCreated, api.StepRegisteredResponse{
			Message: "Step registered",
			Step:    &step,
		})
		return
	}

	if existsError(err) {
		c.JSON(http.StatusConflict, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusConflict,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrRegisterStep, err),
		Status: http.StatusInternalServerError,
	})
}

func (s *Server) getStep(c *gin.Context) {
	stepID := api.StepID(c.Param("stepID"))

	engState, err := s.engine.GetEngineState(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetEngineState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	step, ok := engState.Steps[stepID]
	if !ok {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", engine.ErrStepNotFound, stepID),
			Status: http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, step)
}

func (s *Server) updateStep(c *gin.Context) {
	stepID := api.StepID(c.Param("stepID"))

	var step api.Step
	if err := c.ShouldBindJSON(&step); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return
	}

	if step.ID != stepID {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Step ID in URL does not match step ID in body",
			Status: http.StatusBadRequest,
		})
		return
	}

	if err := step.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
		})
		return
	}

	err := s.engine.UpdateStep(c.Request.Context(), &step)
	if err == nil {
		c.JSON(http.StatusOK, api.StepRegisteredResponse{
			Message: "Step updated",
			Step:    &step,
		})
		return
	}

	if isNotFoundError(err) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusNotFound,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("failed to update step: %v", err),
		Status: http.StatusInternalServerError,
	})
}

func (s *Server) deleteStep(c *gin.Context) {
	stepID := api.StepID(c.Param("stepID"))

	err := s.engine.UnregisterStep(c.Request.Context(), stepID)
	if err == nil {
		c.JSON(http.StatusOK, api.MessageResponse{
			Message: "Step unregistered",
		})
		return
	}

	if isNotFoundError(err) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", engine.ErrStepNotFound, stepID),
			Status: http.StatusNotFound,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %v", ErrUnregisterStep, err),
		Status: http.StatusInternalServerError,
	})
}
