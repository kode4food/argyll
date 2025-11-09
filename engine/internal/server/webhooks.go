package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

func (s *Server) handleWebhook(c *gin.Context) {
	flowID := timebox.ID(c.Param("flowID"))
	stepID := timebox.ID(c.Param("stepID"))
	token := api.Token(c.Param("token"))

	flow, err := s.engine.GetWorkflowState(c.Request.Context(), flowID)
	if err != nil {
		slog.Error("Workflow not found",
			slog.Any("flow_id", flowID),
			slog.Any("error", err))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("Workflow not found: %v", err),
			Status: http.StatusBadRequest,
		})
		return
	}

	exec, ok := flow.Executions[stepID]
	if !ok {
		slog.Error("Execution not found",
			slog.Any("flow_id", flowID),
			slog.Any("step_id", stepID))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Step execution not found",
			Status: http.StatusBadRequest,
		})
		return
	}

	// Check if token matches a work item
	if exec.WorkItems == nil || exec.WorkItems[token] == nil {
		slog.Error("Work item not found",
			slog.Any("flow_id", flowID),
			slog.Any("step_id", stepID),
			slog.Any("token", token))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Work item not found for token",
			Status: http.StatusBadRequest,
		})
		return
	}

	s.handleWorkWebhook(c, flowID, stepID, token)
}

func (s *Server) handleWorkWebhook(
	c *gin.Context, flowID, stepID timebox.ID, token api.Token,
) {
	var result api.StepResult
	if err := c.ShouldBindJSON(&result); err != nil {
		slog.Error("Invalid JSON",
			slog.Any("flow_id", flowID),
			slog.Any("step_id", stepID),
			slog.Any("token", token),
			slog.Any("error", err))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("Invalid JSON: %v", err),
			Status: http.StatusBadRequest,
		})
		return
	}

	if !result.Success {
		slog.Error("Work failed",
			slog.Any("flow_id", flowID),
			slog.Any("step_id", stepID),
			slog.Any("token", token),
			slog.String("error", result.Error))
		if err := s.engine.FailWork(
			c.Request.Context(), flowID, stepID, token, result.Error,
		); err != nil {
			slog.Error("Failed to record work failure",
				slog.Any("flow_id", flowID),
				slog.Any("step_id", stepID),
				slog.Any("token", token),
				slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, api.ErrorResponse{
				Error:  fmt.Sprintf("Failed to fail work: %v", err),
				Status: http.StatusInternalServerError,
			})
			return
		}
		c.Status(http.StatusOK)
		return
	}

	if err := s.engine.CompleteWork(
		c.Request.Context(), flowID, stepID, token, result.Outputs,
	); err != nil {
		slog.Error("Failed to complete work",
			slog.Any("flow_id", flowID),
			slog.Any("step_id", stepID),
			slog.Any("token", token),
			slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("Failed to complete work: %v", err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) handleWebSocket(c *gin.Context) {
	replayFunc := func(flowID timebox.ID, fromSeq int64) ([]*timebox.Event, error) {
		return s.engine.GetWorkflowEvents(c.Request.Context(), flowID, fromSeq)
	}
	HandleWebSocket(s.eventHub, c.Writer, c.Request, replayFunc)
}
