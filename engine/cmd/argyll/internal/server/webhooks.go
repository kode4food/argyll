package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

var (
	ErrInvalidAgentOutcome = errors.New("agent outcome requires either outputs or a problem")
)

func (s *Server) handleWebhook(c *gin.Context) {
	req, err := callbackRequest(c)
	if err == nil {
		err = builtins.Callback(s.engine, &req)
	}
	if err == nil {
		c.Status(http.StatusOK)
		return
	}

	slog.Error("Callback failed",
		log.FlowID(req.FlowStep.FlowID),
		log.StepID(req.FlowStep.StepID),
		log.Token(req.Token),
		log.Error(err))
	status := callbackStatus(err)
	c.JSON(status, api.ErrorResponse{
		Error:  err.Error(),
		Status: status,
	})
}

// callbackRequest reads an Agent outcome or an HTTP step's callback
func callbackRequest(c *gin.Context) (builtins.CallbackRequest, error) {
	req := builtins.CallbackRequest{
		FlowStep: api.FlowStep{
			FlowID: api.FlowID(c.Param(api.ParamFlowID)),
			StepID: api.StepID(c.Param(api.ParamStepID)),
		},
		Token:  api.Token(c.Param(api.ParamToken)),
		Action: api.CallbackAction(c.Param(api.ParamAction)),
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return req, invalidCallback(err)
	}
	if req.Action == api.ActionAgent {
		err := decodeAgentCallback(&req, body)
		return req, err
	}
	if api.IsProblemJSON(c.GetHeader("Content-Type")) {
		var problem api.ProblemDetails
		if err := json.Unmarshal(body, &problem); err != nil {
			return req, invalidCallback(err)
		}
		req.Problem = &problem
		return req, nil
	}
	if req.Action != api.ActionInvoke {
		return req, nil
	}
	if err := json.Unmarshal(body, &req.Outputs); err != nil {
		return req, invalidCallback(err)
	}
	return req, nil
}

func decodeAgentCallback(req *builtins.CallbackRequest, body []byte) error {
	var out api.AgentOutcome
	if err := json.Unmarshal(body, &out); err != nil {
		return invalidCallback(err)
	}
	if (out.Outputs == nil) == (out.Problem == nil) {
		return ErrInvalidAgentOutcome
	}
	req.Outputs = out.Outputs
	req.Problem = out.Problem
	req.Usage = out.Usage
	return nil
}

func invalidCallback(err error) error {
	return fmt.Errorf("%w: %v", builtins.ErrInvalidCallback, err)
}

func callbackStatus(err error) int {
	switch {
	case errors.Is(err, builtins.ErrInvalidCallback),
		errors.Is(err, ErrInvalidAgentOutcome),
		errors.Is(err, builtins.ErrInvalidCallbackAction),
		errors.Is(err, builtins.ErrCallbackExecutionNotFound),
		errors.Is(err, api.ErrWorkItemNotFound),
		errors.Is(err, builtins.ErrCallbackNotCompensating):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
