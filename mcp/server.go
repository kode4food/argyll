package mcp

import (
	"context"
	"io"
	"net/http"

	"github.com/deinstapel/go-jsonrpc"
)

type Server struct {
	baseURL string
	client  *http.Client
}

const defaultBaseURL = "http://localhost:8080"

// NewServer constructs an MCP server with the provided base URL.
func NewServer(baseURL string, client *http.Client) *Server {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Server{
		baseURL: baseURL,
		client:  client,
	}
}

// Run starts the MCP server over the provided streams.
func Run(in io.Reader, out io.Writer) {
	NewServer("", nil).ServeContext(context.Background(), in, out)
}

// Serve starts the MCP server over the provided streams.
func (s *Server) Serve(in io.Reader, out io.Writer) {
	s.ServeContext(context.Background(), in, out)
}

// ServeContext starts the MCP server over the provided streams.
func (s *Server) ServeContext(
	ctx context.Context, in io.Reader, out io.Writer,
) {
	tr := newStdioTransport(in, out)
	peer := jsonrpc.NewPeer(ctx, tr)

	_ = peer.RegisterRPC("initialize", s.handleInitialize)
	_ = peer.RegisterRPC("tools/list", s.handleToolsList)
	_ = peer.RegisterRPC("tools/call", s.handleToolsCall)

	select {
	case <-tr.done:
	case <-ctx.Done():
		tr.Close()
		peer.Close()
	}
}
