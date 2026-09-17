package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/step/builtins"
)

func (s *Server) handleWebhook(c *gin.Context) {
	req := &builtins.CallbackRequest{
		FlowStep: api.FlowStep{
			FlowID: api.FlowID(c.Param(api.ParamFlowID)),
			StepID: api.StepID(c.Param(api.ParamStepID)),
		},
		Token:   api.Token(c.Param(api.ParamToken)),
		Action:  api.CallbackAction(c.Param(api.ParamAction)),
		Request: c.Request,
	}
	status, err := builtins.Callback(s.engine, req)
	if err == nil {
		c.Status(status)
		return
	}

	slog.Error("Callback failed",
		log.FlowID(req.FlowStep.FlowID),
		log.StepID(req.FlowStep.StepID),
		log.Token(req.Token),
		log.Error(err))
	c.JSON(status, api.ErrorResponse{
		Error:  err.Error(),
		Status: status,
	})
}
