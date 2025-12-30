package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

func (s *Server) handleWebhook(c *gin.Context) {
	flowID := api.FlowID(c.Param("flowID"))
	stepID := api.StepID(c.Param("stepID"))
	token := api.Token(c.Param("token"))

	flow, err := s.engine.GetFlowState(c.Request.Context(), flowID)
	if err != nil {
		slog.Error("Flow not found",
			log.FlowID(flowID),
			log.Error(err))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("Flow not found: %v", err),
			Status: http.StatusBadRequest,
		})
		return
	}

	exec, ok := flow.Executions[stepID]
	if !ok {
		slog.Error("Execution not found",
			log.FlowID(flowID),
			log.StepID(stepID),
			log.Error(fmt.Errorf("execution not found")))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Step execution not found",
			Status: http.StatusBadRequest,
		})
		return
	}

	// Check if token matches a work item
	if exec.WorkItems == nil || exec.WorkItems[token] == nil {
		slog.Error("Work item not found",
			log.FlowID(flowID),
			log.StepID(stepID),
			log.Token(token),
			log.Error(fmt.Errorf("work item not found")))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Work item not found for token",
			Status: http.StatusBadRequest,
		})
		return
	}

	fs := engine.FlowStep{FlowID: flowID, StepID: stepID}
	s.handleWorkWebhook(c, fs, token)
}

func (s *Server) handleWorkWebhook(
	c *gin.Context, fs engine.FlowStep, token api.Token,
) {
	var result api.StepResult
	if err := c.ShouldBindJSON(&result); err != nil {
		slog.Error("Invalid JSON",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(token),
			log.Error(err))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  fmt.Sprintf("Invalid JSON: %v", err),
			Status: http.StatusBadRequest,
		})
		return
	}

	if !result.Success {
		slog.Error("Work failed",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(token),
			log.ErrorString(result.Error))
		if err := s.engine.FailWork(
			c.Request.Context(), fs, token, result.Error,
		); err != nil {
			slog.Error("Failed to record work failure",
				log.FlowID(fs.FlowID),
				log.StepID(fs.StepID),
				log.Token(token),
				log.Error(err))
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
		c.Request.Context(), fs, token, result.Outputs,
	); err != nil {
		slog.Error("Failed to complete work",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(token),
			log.Error(err))
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("Failed to complete work: %v", err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) handleWebSocket(c *gin.Context) {
	HandleWebSocket(s.eventHub, c.Writer, c.Request,
		func(id timebox.AggregateID, fromSeq int64) ([]*timebox.Event, error) {
			if len(id) == 0 {
				return nil, nil
			}
			ctx := c.Request.Context()
			switch id[0] {
			case "engine":
				return s.engine.GetEngineEvents(ctx, fromSeq)
			case "flow":
				if len(id) < 2 {
					return nil, errors.New("invalid aggregate_id")
				}
				flowID := api.FlowID(id[1])
				return s.engine.GetFlowEvents(ctx, flowID, fromSeq)
			default:
				return nil, errors.New("invalid aggregate_id")
			}
		},
	)
}
