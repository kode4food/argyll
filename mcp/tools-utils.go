package mcp

import (
	"encoding/json"
	"fmt"
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
		"content": []map[string]any{
			{
				"type": "text",
				"text": string(raw),
			},
		},
	}, nil
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
