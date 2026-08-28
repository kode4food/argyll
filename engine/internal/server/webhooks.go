package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

var (
	ErrExecutionNotFound = errors.New("execution not found")
	ErrWorkItemNotFound  = errors.New("work item not found")
)

func (s *Server) handleWebhook(c *gin.Context) {
	fid := api.FlowID(c.Param(api.ParamFlowID))
	sid := api.StepID(c.Param(api.ParamStepID))
	tkn := api.Token(c.Param(api.ParamToken))

	fl, err := s.engine.GetFlowState(fid)
	if err != nil {
		slog.Error("Flow not found",
			log.FlowID(fid),
			log.Error(err))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Flow not found",
			Status: http.StatusBadRequest,
		})
		return
	}

	ex, ok := fl.Executions[sid]
	if !ok {
		slog.Error("Execution not found",
			log.FlowID(fid),
			log.StepID(sid),
			log.Error(ErrExecutionNotFound))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Step execution not found",
			Status: http.StatusBadRequest,
		})
		return
	}

	// Check if token matches a work item
	work, ok := ex.WorkItems[tkn]
	if !ok {
		slog.Error("Work item not found",
			log.FlowID(fid),
			log.StepID(sid),
			log.Token(tkn),
			log.Error(ErrWorkItemNotFound))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Work item not found for token",
			Status: http.StatusBadRequest,
		})
		return
	}

	fs := api.FlowStep{FlowID: fid, StepID: sid}
	switch api.CallbackAction(c.Param(api.ParamAction)) {
	case api.ActionInvoke:
		s.handleInvokeWebhook(c, fs, tkn)
	case api.ActionCompensate:
		s.handleCompensateWebhook(c, fs, tkn, work)
	default:
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Invalid callback action",
			Status: http.StatusBadRequest,
		})
	}
}

func (s *Server) handleCompensateWebhook(
	c *gin.Context, fs api.FlowStep, tkn api.Token, work api.WorkState,
) {
	if !policy.WorkCompActive(work.Status) &&
		work.Status != api.WorkCompensated &&
		work.Status != api.WorkCompFailed {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Work item is not compensating",
			Status: http.StatusBadRequest,
		})
		return
	}

	var err error
	if api.IsProblemJSON(c.GetHeader("Content-Type")) {
		var problem api.ProblemDetails
		if err = c.ShouldBindJSON(&problem); err != nil {
			slog.Error("Invalid problem JSON",
				log.FlowID(fs.FlowID),
				log.StepID(fs.StepID),
				log.Token(tkn),
				log.Error(err))
			c.JSON(http.StatusBadRequest, api.ErrorResponse{
				Error:  "Invalid problem JSON",
				Status: http.StatusBadRequest,
			})
			return
		}
		slog.Error("Compensation failed",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(tkn),
			log.ErrorString(problem.Error()))
		err = s.engine.FailCompensation(fs, tkn, problem.Error())
	} else {
		err = s.engine.CompleteCompensation(fs, tkn)
	}
	if err != nil {
		slog.Error("Failed to record compensation outcome",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(tkn),
			log.Error(err))
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  "Failed to record compensation outcome",
			Status: http.StatusInternalServerError,
		})
		return
	}
	c.Status(http.StatusOK)
}

func (s *Server) handleInvokeWebhook(
	c *gin.Context, fs api.FlowStep, tkn api.Token,
) {
	contentType := c.GetHeader("Content-Type")
	if api.IsProblemJSON(contentType) {
		s.handleInvokeProblemWebhook(c, fs, tkn)
		return
	}

	var outputs api.Args
	if err := c.ShouldBindJSON(&outputs); err != nil {
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

	if err := s.engine.CompleteWork(fs, tkn, outputs); err != nil {
		if errors.Is(err, engine.ErrInvalidWorkTransition) {
			slog.Info("Ignoring duplicate work completion",
				log.FlowID(fs.FlowID),
				log.StepID(fs.StepID),
				log.Token(tkn))
			c.Status(http.StatusOK)
			return
		}
		slog.Error("Failed to complete work",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(tkn),
			log.Error(err))
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  "Failed to complete work",
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.Status(http.StatusOK)
}

func (s *Server) handleInvokeProblemWebhook(
	c *gin.Context, fs api.FlowStep, tkn api.Token,
) {
	var problem api.ProblemDetails
	if err := c.ShouldBindJSON(&problem); err != nil {
		slog.Error("Invalid problem JSON",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(tkn),
			log.Error(err))
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error:  "Invalid problem JSON",
			Status: http.StatusBadRequest,
		})
		return
	}

	errMsg := problem.Error()
	slog.Error("Work failed",
		log.FlowID(fs.FlowID),
		log.StepID(fs.StepID),
		log.Token(tkn),
		log.ErrorString(errMsg))
	if err := s.engine.FailWork(fs, tkn, errMsg); err != nil {
		if errors.Is(err, engine.ErrInvalidWorkTransition) {
			slog.Info("Ignoring duplicate work failure",
				log.FlowID(fs.FlowID),
				log.StepID(fs.StepID),
				log.Token(tkn))
			c.Status(http.StatusOK)
			return
		}
		slog.Error("Failed to record work failure",
			log.FlowID(fs.FlowID),
			log.StepID(fs.StepID),
			log.Token(tkn),
			log.Error(err))
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  "Failed to record work failure",
			Status: http.StatusInternalServerError,
		})
		return
	}
	c.Status(http.StatusOK)
}
