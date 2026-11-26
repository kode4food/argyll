package mcp

import (
	"context"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// Server exposes the engine API as MCP tools, resources, and prompts
type Server struct {
	client  *http.Client
	baseURL string
}

const (
	// DefaultBaseURL is used when no Argyll URL is provided
	DefaultBaseURL = "http://localhost:8080"

	// HTTPPath is where the Streamable HTTP transport serves MCP
	HTTPPath = "/mcp"

	serverName    = "argyll-mcp"
	serverVersion = "1.0.0"
)

// NewServer constructs an MCP server with the provided base URL
func NewServer(baseURL string, client *http.Client) *Server {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if client == nil {
		client = &http.Client{CheckRedirect: api.SameOriginRedirect}
	}
	return &Server{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

// Run serves MCP over stdio until the client disconnects
func (s *Server) Run(ctx context.Context) error {
	srv, err := s.MCPServer()
	if err != nil {
		return err
	}
	return srv.Run(ctx, &mcp.StdioTransport{})
}

// Handler serves Streamable HTTP and forwards each caller's Authorization header
func (s *Server) Handler() (http.Handler, error) {
	srv, err := s.MCPServer()
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle(HTTPPath, mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return srv
	}, nil))
	return mux, nil
}

// MCPServer builds the configured MCP server for transports and tests
func (s *Server) MCPServer() (*mcp.Server, error) {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)
	reg := &toolRegistry{server: srv}
	s.registerTools(reg)
	s.registerGuidance(reg)
	if reg.err != nil {
		return nil, reg.err
	}
	return srv, nil
}
