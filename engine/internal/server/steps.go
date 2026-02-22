package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

const MaxStepBodyBytes = 512 * 1024 // 512 KB

var (
	ErrInvalidJSON    = errors.New("invalid JSON request")
	ErrListSteps      = errors.New("failed to list steps")
	ErrRegisterStep   = errors.New("failed to register step")
	ErrUnregisterStep = errors.New("failed to unregister step")
)

func (s *Server) listSteps(c *gin.Context) {
	steps, err := s.engine.ListSteps()
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
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxStepBodyBytes)
	var step api.Step
	if err := c.ShouldBindJSON(&step); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrInvalidJSON, err),
			Status: http.StatusBadRequest,
		})
		return
	}

	err := s.engine.RegisterStep(&step)
	if err == nil {
		c.JSON(http.StatusCreated, api.StepRegisteredResponse{
			Message: "Step registered",
			Step:    &step,
		})
		return
	}

	if errors.Is(err, engine.ErrStepExists) {
		c.JSON(http.StatusConflict, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusConflict,
		})
		return
	}
	if errors.Is(err, engine.ErrInvalidStep) {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
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

	catState, err := s.engine.GetCatalogState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetCatalogState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	if step, ok := catState.Steps[stepID]; ok {
		c.JSON(http.StatusOK, step)
		return
	}

	c.JSON(http.StatusNotFound, api.ErrorResponse{
		Error:  fmt.Sprintf("%s: %s", engine.ErrStepNotFound, stepID),
		Status: http.StatusNotFound,
	})
}

func (s *Server) updateStep(c *gin.Context) {
	stepID := api.StepID(c.Param("stepID"))

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxStepBodyBytes)
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

	err := s.engine.UpdateStep(&step)
	if err == nil {
		c.JSON(http.StatusOK, api.StepRegisteredResponse{
			Message: "Step updated",
			Step:    &step,
		})
		return
	}

	if errors.Is(err, engine.ErrStepNotFound) {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusNotFound,
		})
		return
	}
	if errors.Is(err, engine.ErrInvalidStep) {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  err.Error(),
			Status: http.StatusBadRequest,
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

	catState, err := s.engine.GetCatalogState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetCatalogState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	if _, ok := catState.Steps[stepID]; !ok {
		c.JSON(http.StatusNotFound, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %s", engine.ErrStepNotFound, stepID),
			Status: http.StatusNotFound,
		})
		return
	}

	err = s.engine.UnregisterStep(stepID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrUnregisterStep, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, api.MessageResponse{
		Message: "Step unregistered",
	})
}
