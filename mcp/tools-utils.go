package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// toolHandler answers one tool call, with the caller's Authorization carried in ctx
	toolHandler[In any] func(context.Context, In) (any, error)

	// engineCall reaches one engine path for the caller in ctx
	engineCall func(ctx context.Context, path string) (any, error)

	toolRegistry struct {
		server *mcp.Server
		err    error
	}

	noInput struct{}

	idInput struct {
		ID string `json:"id" jsonschema:"Step, Flow, Space, or Feed ID"`
	}
)

// schemaOptions supplies an immutable override for the recursive Goals schema
var schemaOptions = &jsonschema.ForOptions{
	TypeSchemas: map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[api.Goals](): {
			Type:     "object",
			Required: []string{"steps"},
			Properties: map[string]*jsonschema.Schema{
				"steps": {
					Type:        "array",
					Items:       &jsonschema.Schema{Type: "string"},
					Description: "Goal Step IDs",
				},
				"else": {
					Type:        "object",
					Description: "Fallback goal set of the same shape",
				},
			},
		},
	},
}

// addTool registers a tool whose engine requests act as the MCP caller
func addTool[In any](reg *toolRegistry, tool *mcp.Tool, h toolHandler[In]) {
	if reg.err != nil {
		return
	}
	schema, err := jsonschema.For[In](schemaOptions)
	if err != nil {
		reg.err = fmt.Errorf("%w: %s: %w", ErrToolSchema, tool.Name, err)
		return
	}
	tool.InputSchema = schema
	mcp.AddTool(reg.server, tool, func(
		ctx context.Context, req *mcp.CallToolRequest, in In,
	) (*mcp.CallToolResult, any, error) {
		if req.Extra != nil && req.Extra.Header != nil {
			ctx = context.WithValue(ctx, authorizationKey{}, req.Extra.Header.Get("Authorization"))
		}
		payload, err := h(ctx, in)
		if err != nil {
			return nil, nil, err
		}
		res, err := toolResult(payload)
		return res, nil, err
	})
}

func toolResult(payload any) (*mcp.CallToolResult, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}},
	}, nil
}

// pathTool serves a tool without arguments that calls one path
func pathTool(call engineCall, path string) toolHandler[noInput] {
	return func(ctx context.Context, _ noInput) (any, error) {
		return call(ctx, path)
	}
}

// idTool formats the engine path with a single escaped ID
func idTool(call engineCall, pathFormat string) toolHandler[idInput] {
	return func(ctx context.Context, args idInput) (any, error) {
		if args.ID == "" {
			return nil, errInvalidParams("id is required")
		}
		return call(ctx, fmt.Sprintf(pathFormat, url.PathEscape(args.ID)))
	}
}

func errInvalidParams(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidParams, message)
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func coalesceType(s string) string {
	if s == "" {
		return "any"
	}
	return s
}

func asMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		res := map[string]any{}
		for k, v := range m {
			ks, ok := k.(string)
			if !ok {
				continue
			}
			res[ks] = v
		}
		return res, true
	default:
		return nil, false
	}
}
