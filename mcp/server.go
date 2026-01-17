package mcp

import (
	"net/http"

	"github.com/localrivet/gomcp/server"
)

type Server struct {
	baseURL string
	client  *http.Client
}

// DefaultBaseURL is used when no engine URL is provided
const DefaultBaseURL = "http://localhost:8080"

// NewServer constructs an MCP server with the provided base URL
func NewServer(baseURL string, client *http.Client) *Server {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Server{
		baseURL: baseURL,
		client:  client,
	}
}

// Run starts the MCP server over stdio and blocks until exit
func (s *Server) Run() error {
	return s.MCPServer().AsStdio().Run()
}

// MCPServer builds a configured gomcp server for advanced usage and tests
func (s *Server) MCPServer() server.Server {
	srv := server.NewServer("argyll-mcp")
	s.registerTools(srv)
	return srv
}
