package server

import (
	"log/slog"
	"net/http"
	"sync"

	glog "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// Server implements the HTTP API server for the orchestrator
type Server struct {
	engine   *engine.Engine
	eventHub *event.Hub
	sockets  util.Set[*Client]
	status   []StatusProvider
	mu       sync.Mutex
}

// NewServer creates a new HTTP API server
func NewServer(
	eng *engine.Engine, hub *event.Hub, status ...StatusProvider,
) *Server {
	s := &Server{
		engine:   eng,
		eventHub: hub,
		sockets:  util.Set[*Client]{},
	}
	s.status = append([]StatusProvider{
		NewWebSocketStatusProvider(s),
	}, status...)
	return s
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

		// Catalog endpoints
		eng.GET("/catalog", s.getCatalog)
		eng.GET("/catalog/events", s.getCatalogEvents)

		// Cluster endpoints
		eng.GET("/cluster", s.getCluster)
		eng.GET("/cluster/events", s.getClusterEvents)

		// Flow endpoints
		eng.GET("/flow", s.listFlows)
		eng.POST("/flow", s.startFlow)
		eng.POST("/flow/query", s.queryFlows)
		eng.GET("/flow/:flowID/status", s.getFlowStatus)
		eng.GET("/flow/:flowID/events", s.getFlowEvents)
		eng.GET("/flow/:flowID", s.getFlow)

		// WebSocket
		eng.GET("/ws", s.handleWebSocket)
	}

	return router
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

func (s *Server) webSocketCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sockets)
}

// CloseWebSockets closes all active WebSocket connections
func (s *Server) CloseWebSockets() {
	for _, c := range s.webSockets() {
		c.Close()
	}
}

func (s *Server) webSockets() []*Client {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := make([]*Client, 0, len(s.sockets))
	for c := range s.sockets {
		res = append(res, c)
	}
	return res
}
