package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

var (
	ErrExecutionNotFound = errors.New("execution not found")
	ErrWorkItemNotFound  = errors.New("work item not found")
)

func (s *Server) handleWebhook(c *gin.Context) {
	flowID := api.FlowID(c.Param("flowID"))
	stepID := api.StepID(c.Param("stepID"))
	token := api.Token(c.Param("token"))

	flow, err := s.engine.GetFlowState(flowID)
	if err != nil {
		slog.Error("Flow not found",
			log.FlowID(flowID),
			log.Error(err))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Flow not found",
			Status: http.StatusBadRequest,
		})
		return
	}

	exec, ok := flow.Executions[stepID]
	if !ok {
		slog.Error("Execution not found",
			log.FlowID(flowID),
			log.StepID(stepID),
			log.Error(ErrExecutionNotFound))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Step execution not found",
			Status: http.StatusBadRequest,
		})
		return
	}

	// Check if token matches a work item
	if _, ok := exec.WorkItems[token]; !ok {
		slog.Error("Work item not found",
			log.FlowID(flowID),
			log.StepID(stepID),
			log.Token(token),
			log.Error(ErrWorkItemNotFound))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Work item not found for token",
			Status: http.StatusBadRequest,
		})
		return
	}

	fs := api.FlowStep{FlowID: flowID, StepID: stepID}
	s.handleWorkWebhook(c, fs, token)
}

func (s *Server) handleWorkWebhook(
	c *gin.Context, fs api.FlowStep, tkn api.Token,
) {
	var result api.StepResult
	if err := c.ShouldBindJSON(&result); err != nil {
		slog.Error("Invalid JSON",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(tkn),
			log.Error(err))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Invalid JSON",
			Status: http.StatusBadRequest,
		})
		return
	}

	if !result.Success {
		slog.Error("Work failed",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(tkn),
			log.ErrorString(result.Error))
		if err := s.engine.FailWork(fs, tkn, result.Error); err != nil {
			slog.Error("Failed to record work failure",
				log.FlowID(fs.FlowID),
				log.StepID(fs.StepID),
				log.Token(tkn),
				log.Error(err))
			if errors.Is(err, engine.ErrInvalidWorkTransition) {
				c.JSON(http.StatusBadRequest, api.ErrorResponse{
					Error:  "Work item already completed",
					Status: http.StatusBadRequest,
				})
				return
			}
			c.JSON(http.StatusInternalServerError, api.ErrorResponse{
				Error:  "Failed to record work failure",
				Status: http.StatusInternalServerError,
			})
			return
		}
		c.Status(http.StatusOK)
		return
	}

	if err := s.engine.CompleteWork(fs, tkn, result.Outputs); err != nil {
		slog.Error("Failed to complete work",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(tkn),
			log.Error(err))
		if errors.Is(err, engine.ErrInvalidWorkTransition) {
			c.JSON(http.StatusBadRequest, api.ErrorResponse{
				Error:  "Work item already completed",
				Status: http.StatusBadRequest,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  "Failed to complete work",
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.Status(http.StatusOK)
}
