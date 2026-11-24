package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"

	glog "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"

	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/internal/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/auth"
	"github.com/kode4food/argyll/engine/pkg/util"
)

// Server implements the HTTP API server for the orchestrator
type Server struct {
	engine        *engine.Engine
	eventHub      *event.Hub
	sockets       util.Set[*Client]
	status        []StatusProvider
	callbackLimit int64
	authenticator auth.Authenticator
	mu            sync.Mutex
}

// NewServer creates a new HTTP API server
func NewServer(eng *engine.Engine, hub *event.Hub, status ...StatusProvider) *Server {
	s := &Server{
		engine:   eng,
		eventHub: hub,
		sockets:  util.Set[*Client]{},
	}
	s.status = append([]StatusProvider{NewWebSocketStatusProvider(s)}, status...)
	return s
}

// WithAuthenticator protects engine routes with the configured authenticator
func (s *Server) WithAuthenticator(a auth.Authenticator) *Server {
	s.authenticator = a
	return s
}

// SetupRoutes configures and returns the HTTP router with all API endpoints
func (s *Server) SetupRoutes() *gin.Engine {
	router := gin.New()
	// Gin's request dump leaves callback paths and WebSocket credentials intact
	router.Use(gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		slog.Error("HTTP handler panic", slog.String("stack", string(debug.Stack())))
		c.AbortWithStatus(http.StatusInternalServerError)
	}))
	router.Use(glog.SetLogger(
		glog.WithLogger(func(c *gin.Context, l *slog.Logger) *slog.Logger {
			return slog.Default()
		}),
		// Callback paths carry a secret, and the callback handler logs its own failures
		glog.WithSkipper(func(c *gin.Context) bool {
			return strings.HasPrefix(c.Request.URL.Path, "/callbacks/")
		}),
	))

	// Health check
	router.GET("/health", s.handleHealth)

	// Callback endpoint
	router.POST(api.CallbackRoute, s.handleWebhook)

	// Engine endpoints
	reader := router.Group("/engine", requireRole(auth.RoleReader, s.authenticator))
	operator := router.Group("/engine", requireRole(auth.RoleOperator, s.authenticator))
	admin := router.Group("/engine", requireRole(auth.RoleAdmin, s.authenticator))
	{
		reader.GET("", s.handleEngine)

		// Step endpoints
		reader.GET("/steps", s.listSteps)
		admin.POST("/steps", s.createStep)
		reader.GET("/steps/:step_id", s.getStep)
		admin.PUT("/steps/:step_id", s.updateStep)
		admin.DELETE("/steps/:step_id", s.deleteStep)

		// Space endpoints
		reader.GET("/spaces", s.listSpaces)
		admin.POST("/spaces", s.createSpace)
		reader.POST("/spaces/preview", s.previewSpace)
		reader.GET("/spaces/:space_id", s.getSpace)
		reader.GET("/spaces/:space_id/steps", s.listSpaceSteps)
		admin.PUT("/spaces/:space_id", s.updateSpace)
		admin.DELETE("/spaces/:space_id", s.deleteSpace)

		// Feed endpoints
		reader.GET("/feeds", s.listFeeds)
		admin.POST("/feeds", s.createFeed)
		reader.GET("/feeds/:feed_id", s.getFeed)
		admin.PUT("/feeds/:feed_id", s.updateFeed)
		admin.DELETE("/feeds/:feed_id", s.deleteFeed)

		// Health endpoints
		reader.GET("/health", s.handleEngineHealth)
		reader.GET("/health/:step_id", s.handleEngineHealthByID)

		// Plan preview
		reader.POST("/plan", s.handlePlanPreview)

		// Catalog endpoints
		reader.GET("/catalog", s.getCatalog)
		reader.GET("/catalog/events", s.getCatalogEvents)

		// Cluster endpoints
		reader.GET("/cluster", s.getCluster)
		reader.GET("/cluster/events", s.getClusterEvents)

		// Flow endpoints
		reader.GET("/flows", s.listFlows)
		operator.POST("/flows", s.startFlow)
		reader.POST("/flows/query", s.queryFlows)
		reader.GET("/flows/:flow_id/status", s.getFlowStatus)
		reader.GET("/flows/:flow_id/events", s.getFlowEvents)
		reader.GET("/flows/:flow_id", s.getFlow)

		// WebSocket
		reader.GET("/ws", s.handleWebSocket)
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
func requireRole(role auth.Role, a auth.Authenticator) gin.HandlerFunc {
	if a == nil {
		return func(*gin.Context) {}
	}
	return func(c *gin.Context) {
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})
		auth.Handler(next, a, role).ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
