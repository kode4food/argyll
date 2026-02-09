package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	glog "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// Server implements the HTTP API server for the orchestrator
type Server struct {
	engine   *engine.Engine
	eventHub *timebox.EventHub
	sockets  util.Set[*Client]
	mu       sync.Mutex
}

var (
	// ErrGetEngineState is returned when the engine state cannot be retrieved
	ErrGetEngineState = errors.New("failed to get engine state")
)

// NewServer creates a new HTTP API server
func NewServer(eng *engine.Engine, hub *timebox.EventHub) *Server {
	return &Server{
		engine:   eng,
		eventHub: hub,
		sockets:  util.Set[*Client]{},
	}
}

// SetupRoutes configures and returns the HTTP router with all API endpoints
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

		// Flow endpoints
		eng.GET("/flow", s.listFlows)
		eng.POST("/flow", s.startFlow)
		eng.POST("/flow/query", s.queryFlows)
		eng.GET("/flow/:flowID", s.getFlow)

		// WebSocket
		eng.GET("/ws", s.handleWebSocket)
	}

	return router
}

func (s *Server) handleEngine(c *gin.Context) {
	engState, err := s.engine.GetEngineState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Error:  fmt.Sprintf("%s: %v", ErrGetEngineState, err),
			Status: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, engState)
}

func (s *Server) registerWebSocket(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sockets.Add(c)
}

func (s *Server) unregisterWebSocket(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sockets.Remove(c)
}

// CloseWebSockets closes all active WebSocket connections.
func (s *Server) CloseWebSockets() {
	s.mu.Lock()
	conns := make([]*Client, 0, len(s.sockets))
	for c := range s.sockets {
		conns = append(conns, c)
	}
	s.mu.Unlock()

	for _, c := range conns {
		c.Close()
	}
}
