package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type (
	// toolbox holds the allowed tools of an attempt's MCP servers, named for the model
	toolbox struct {
		sessions []*mcp.ClientSession
		routes   map[string]route
		specs    []ToolSpec
	}

	route struct {
		session *mcp.ClientSession
		tool    string
	}

	// headerTransport adds an MCP server's configured headers to every request
	headerTransport struct {
		headers map[string]string
	}
)

const (
	clientName    = "argyll-agent-worker"
	clientVersion = "1"
	toolSeparator = "__"
)

var (
	ErrUnknownServer = errors.New("unknown MCP server")
	ErrDuplicateTool = errors.New("duplicate MCP tool name")
)

var _ http.RoundTripper = (*headerTransport)(nil)

// RoundTrip adds the server's configured headers to the request
func (t *headerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	req := r.Clone(r.Context())
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// addServer connects to a selected server and routes its allowed tools
func (b *toolbox) addServer(
	ctx context.Context, sel api.MCPServer, servers map[string]MCPServer,
) error {
	s, ok := servers[sel.Server]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownServer, sel.Server)
	}
	cs, err := connect(ctx, s)
	if err != nil {
		return err
	}
	b.sessions = append(b.sessions, cs)
	tools, err := listTools(ctx, cs)
	if err != nil {
		return err
	}
	for _, t := range tools {
		if sel.Tools != nil && !slices.Contains(sel.Tools, t.Name) {
			continue
		}
		name := sel.Server + toolSeparator + t.Name
		if b.has(name) {
			return fmt.Errorf("%w: %q", ErrDuplicateTool, name)
		}
		b.routes[name] = route{
			session: cs,
			tool:    t.Name,
		}
		b.specs = append(b.specs, ToolSpec{
			Name:        name,
			Description: t.Description,
			Schema:      schemaOf(t.InputSchema),
		})
	}
	return nil
}

func (b *toolbox) has(name string) bool {
	_, ok := b.routes[name]
	return ok
}

// call runs a tool and renders its result as text for the model
func (b *toolbox) call(ctx context.Context, name string, args api.Args) (string, error) {
	r := b.routes[name]
	res, err := r.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      r.tool,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(res.Content))
	for _, c := range res.Content {
		parts = append(parts, contentText(c))
	}
	text := strings.Join(parts, "\n")
	if res.IsError {
		return "Error: " + text, nil
	}
	return text, nil
}

func (b *toolbox) close() {
	for _, cs := range b.sessions {
		_ = cs.Close()
	}
}

func openToolbox(
	ctx context.Context, selections []api.MCPServer, servers map[string]MCPServer,
) (*toolbox, error) {
	box := &toolbox{routes: map[string]route{}}
	for _, sel := range selections {
		if err := box.addServer(ctx, sel, servers); err != nil {
			box.close()
			return nil, err
		}
	}
	// A stable tool order keeps prompts identical across attempts, so providers can cache them
	slices.SortFunc(box.specs, func(a, b ToolSpec) int {
		return strings.Compare(a.Name, b.Name)
	})
	return box, nil
}

// listServers reports every configured MCP server with its tools, leaving out the tools of a
// server that cannot be reached
func listServers(
	ctx context.Context, servers map[string]MCPServer,
) ([]api.MCPServer, []api.ProblemDetails) {
	res := make([]api.MCPServer, 0, len(servers))
	var problems []api.ProblemDetails
	for _, name := range slices.Sorted(maps.Keys(servers)) {
		tools, err := toolNames(ctx, servers[name])
		if err != nil {
			slog.Warn("MCP discovery failed", slog.String("server", name), log.Error(err))
			p := *problemOf(err)
			p.Instance = "mcp/" + name
			problems = append(problems, p)
		}
		res = append(res, api.MCPServer{
			Server: name,
			Tools:  tools,
		})
	}
	return res, problems
}

func toolNames(ctx context.Context, s MCPServer) ([]string, error) {
	cs, err := connect(ctx, s)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cs.Close() }()
	tools, err := listTools(ctx, cs)
	if err != nil {
		return nil, err
	}
	res := make([]string, 0, len(tools))
	for _, t := range tools {
		res = append(res, t.Name)
	}
	slices.Sort(res)
	return res, nil
}

func connect(ctx context.Context, s MCPServer) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    clientName,
		Version: clientVersion,
	}, nil)
	var rt http.RoundTripper = &headerTransport{
		headers: s.Headers,
	}
	if s.Tokens != nil {
		rt = &oauth2.Transport{
			Source: s.Tokens,
			Base:   rt,
		}
	}
	transport := &mcp.StreamableClientTransport{
		Endpoint: s.URL,
		HTTPClient: &http.Client{
			Transport:     rt,
			CheckRedirect: api.SameOriginRedirect,
		},
		DisableStandaloneSSE: true,
	}
	return client.Connect(ctx, transport, nil)
}

func listTools(ctx context.Context, cs *mcp.ClientSession) ([]*mcp.Tool, error) {
	var res []*mcp.Tool
	for t, err := range cs.Tools(ctx, nil) {
		if err != nil {
			return nil, err
		}
		res = append(res, t)
	}
	return res, nil
}

// schemaOf reads a listed tool's input schema, which a client decodes as a JSON object
func schemaOf(schema any) map[string]any {
	if m, ok := schema.(map[string]any); ok {
		return m
	}
	return map[string]any{"type": "object"}
}

func contentText(c mcp.Content) string {
	if t, ok := c.(*mcp.TextContent); ok {
		return t.Text
	}
	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(data)
}
