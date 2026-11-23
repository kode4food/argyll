package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/localrivet/gomcp/server"
)

func toolResult(payload any, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(raw)}},
	}, nil
}

// pathTool serves a tool without arguments that calls one path
func pathTool(call func(string) (any, error), path string) func(*server.Context, any) (any, error) {
	return func(*server.Context, any) (any, error) {
		return toolResult(call(path))
	}
}

type idInput struct {
	ID string `json:"id"`
}

// idTool serves a tool that calls pfx + id + sfx
func idTool(
	call func(string) (any, error), pfx, sfx string,
) func(*server.Context, idInput) (any, error) {
	return func(_ *server.Context, args idInput) (any, error) {
		if args.ID == "" {
			return nil, errInvalidParams("id is required")
		}
		return toolResult(call(pfx + args.ID + sfx))
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
