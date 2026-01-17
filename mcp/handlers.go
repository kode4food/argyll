package mcp

import (
	"context"
	"encoding/json"

	"github.com/deinstapel/go-jsonrpc"
)

type (
	rpcRequest struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}

	toolDef struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		InputSchema map[string]any `json:"inputSchema"`
	}

	toolsListResult struct {
		Tools []toolDef `json:"tools"`
	}

	toolCallResult struct {
		Content []toolContent `json:"content"`
	}

	toolContent struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
		JSON any    `json:"json,omitempty"`
	}

	previewPlanArgs struct {
		Goals []string       `json:"goals"`
		Init  map[string]any `json:"init,omitempty"`
	}

	getStepArgs struct {
		ID string `json:"id"`
	}
)

const (
	rpcErrInvalidParams  = -32602
	rpcErrMethodNotFound = -32601
)

func (s *Server) handleInitialize(
	_ context.Context, _ json.RawMessage,
) (json.RawMessage, error) {
	result := map[string]any{
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "argyll-mcp",
			"version": "0.1.0",
		},
	}
	return marshalResult(result)
}

func (s *Server) handleToolsList(
	_ context.Context, _ json.RawMessage,
) (json.RawMessage, error) {
	tools := []toolDef{
		{
			Name:        "list_steps",
			Description: "List registered steps in the engine",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "get_step",
			Description: "Fetch a single step by ID",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type": "string",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "preview_plan",
			Description: "Preview execution plan for goal steps and init state",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"goals": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"init": map[string]any{
						"type": "object",
					},
				},
				"required": []string{"goals"},
			},
		},
		{
			Name:        "engine_state",
			Description: "Fetch the current engine state",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}
	return marshalResult(toolsListResult{Tools: tools})
}

func (s *Server) handleToolsCall(
	_ context.Context, params json.RawMessage,
) (json.RawMessage, error) {
	var call rpcRequest
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, jsonrpc.NewRPCErr(rpcErrInvalidParams, "Invalid params")
	}

	switch call.Name {
	case "list_steps":
		payload, err := s.httpGet("/engine/step")
		return toolResult(payload, err)
	case "get_step":
		var args getStepArgs
		if err := json.Unmarshal(call.Arguments, &args); err != nil ||
			args.ID == "" {
			return nil, jsonrpc.NewRPCErr(
				rpcErrInvalidParams, "Invalid params",
			)
		}
		payload, err := s.httpGet("/engine/step/" + args.ID)
		return toolResult(payload, err)
	case "preview_plan":
		var args previewPlanArgs
		if err := json.Unmarshal(call.Arguments, &args); err != nil ||
			len(args.Goals) == 0 {
			return nil, jsonrpc.NewRPCErr(
				rpcErrInvalidParams, "Invalid params",
			)
		}
		if args.Init == nil {
			args.Init = map[string]any{}
		}
		payload, err := s.httpPost("/engine/plan", args)
		return toolResult(payload, err)
	case "engine_state":
		payload, err := s.httpGet("/engine")
		return toolResult(payload, err)
	default:
		return nil, jsonrpc.NewRPCErr(
			rpcErrMethodNotFound, "Method not found",
		)
	}
}
