package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	glog "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/internal/client"
	"github.com/kode4food/spuds/engine/internal/config"
	"github.com/kode4food/spuds/engine/internal/engine"
	"github.com/kode4food/spuds/engine/pkg/api"
)

type Server struct {
	engine     *engine.Engine
	config     *config.Config
	eventHub   timebox.EventHub
	stepClient client.Client
}

var ErrGetEngineState = errors.New("failed to get engine state")

func NewServer(
	eng *engine.Engine, cfg *config.Config, hub timebox.EventHub,
	client client.Client,
) *Server {
	return &Server{
		engine:     eng,
		config:     cfg,
		eventHub:   hub,
		stepClient: client,
	}
}

func (s *Server) SetupRoutes() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(glog.SetLogger(
		glog.WithLogger(func(c *gin.Context, l *slog.Logger) *slog.Logger {
			return slog.Default()
		}),
	))

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, POST, PUT, DELETE, OPTIONS",
		)
		c.Writer.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, Authorization",
		)

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})

	// Health check
	router.GET("/health", s.handleHealth)

	// Webhook endpoint
	router.POST("/webhook/:flowID/:stepID/:token", s.handleWebhook)

	// Engine endpoints
	eng := router.Group("/engine")
	{
		eng.GET("", s.handleEngine)
		eng.GET("/", s.handleEngine)

		// Step endpoints
		eng.GET("/step", s.listSteps)
		eng.POST("/step", s.createStep)
		eng.GET("/step/:stepID", s.getStep)
		eng.PUT("/step/:stepID", s.updateStep)
		eng.DELETE("/step/:stepID", s.deleteStep)

		// Health endpoints
		eng.GET("/health", s.handleEngineHealth)
		eng.GET("/health/:stepID", s.handleEngineHealthByID)

		// Plan preview
		eng.POST("/plan", s.handlePlanPreview)

		// Workflow endpoints
		eng.GET("/workflow", s.listWorkflows)
		eng.POST("/workflow", s.startWorkflow)
		eng.GET("/workflow/:flowID", s.getWorkflow)

		// WebSocket
		eng.GET("/ws", s.handleWebSocket)
	}

	return router
}

func (s *Server) handleEngine(c *gin.Context) {
	engState, err := s.engine.GetEngineState(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetEngineState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, engState)
}

func isNotFoundError(err error) bool {
	return errors.Is(err, engine.ErrStepDoesNotExist) ||
		errors.Is(err, engine.ErrWorkflowNotFound) ||
		errors.Is(err, engine.ErrStepNotFound)
}

func existsError(err error) bool {
	return errors.Is(err, engine.ErrStepAlreadyExists) ||
		errors.Is(err, engine.ErrWorkflowExists) ||
		errors.Is(err, engine.ErrStepExists)
}
