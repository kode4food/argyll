package mcp

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/localrivet/gomcp/server"
)

type (
	getStepArgs struct {
		ID string `json:"id"`
	}

	getFlowArgs struct {
		ID string `json:"id"`
	}

	registerStepInput struct {
		Step map[string]any `json:"step"`
	}

	updateStepArgs struct {
		ID   string         `json:"id"`
		Step map[string]any `json:"step"`
	}

	previewPlanArgs struct {
		Goals []string       `json:"goals"`
		Init  map[string]any `json:"init,omitempty"`
	}

	previewPlanInput struct {
		Goals []string        `json:"goals"`
		Init  *map[string]any `json:"init,omitempty"`
	}

	startFlowInput struct {
		Flow map[string]any `json:"flow"`
	}
)

var (
	ErrInvalidParams = errors.New("invalid params")
)

func (s *Server) registerTools(srv server.Server) {
	srv.Tool(
		"openapi",
		"Return compact OpenAPI specs for client generation",
		func(_ *server.Context, _ struct{}) (any, error) {
			specs, err := loadAllSpecs()
			if err != nil {
				return nil, err
			}
			payload := make([]map[string]any, 0, len(specs))
			for _, spec := range specs {
				payload = append(payload, map[string]any{
					"name":    spec.Name,
					"title":   spec.Title,
					"version": spec.Version,
					"doc":     compactOpenAPI(spec.Doc),
				})
			}
			return toolResult(map[string]any{
				"specs": payload,
			}, nil)
		},
	)

	srv.Tool(
		"list_steps",
		"List registered steps in the engine",
		func(*server.Context, any) (any, error) {
			payload, err := s.httpGet("/engine/step")
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"register_step",
		"Register a new step with the engine",
		func(_ *server.Context, args registerStepInput) (any, error) {
			if len(args.Step) == 0 {
				return nil, errInvalidParams("step body is required")
			}
			payload, err := s.httpPost("/engine/step", args.Step)
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"get_step",
		"Fetch a single step by ID",
		func(_ *server.Context, args getStepArgs) (any, error) {
			if args.ID == "" {
				return nil, errInvalidParams("id is required")
			}
			payload, err := s.httpGet("/engine/step/" + args.ID)
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"update_step",
		"Update an existing step registration",
		func(_ *server.Context, args updateStepArgs) (any, error) {
			if args.ID == "" {
				return nil, errInvalidParams("id is required")
			}
			if len(args.Step) == 0 {
				return nil, errInvalidParams("step body is required")
			}
			payload, err := s.httpPut("/engine/step/"+args.ID, args.Step)
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"unregister_step",
		"Remove a step from the engine",
		func(_ *server.Context, args getStepArgs) (any, error) {
			if args.ID == "" {
				return nil, errInvalidParams("id is required")
			}
			payload, err := s.httpDelete("/engine/step/" + args.ID)
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"preview_plan",
		"Preview execution plan for goal steps and init state",
		func(_ *server.Context, args previewPlanInput) (any, error) {
			if len(args.Goals) == 0 {
				return nil, errInvalidParams("goals is required")
			}
			init := map[string]any{}
			if args.Init != nil {
				init = *args.Init
			}
			payload, err := s.httpPost(
				"/engine/plan",
				previewPlanArgs{
					Goals: args.Goals,
					Init:  init,
				},
			)
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"list_flows",
		"List all flows in the engine",
		func(*server.Context, any) (any, error) {
			payload, err := s.httpGet("/engine/flow")
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"get_flow",
		"Fetch a single flow by ID",
		func(_ *server.Context, args getFlowArgs) (any, error) {
			if args.ID == "" {
				return nil, errInvalidParams("id is required")
			}
			payload, err := s.httpGet("/engine/flow/" + args.ID)
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"start_flow",
		"Start a new flow execution",
		func(_ *server.Context, args startFlowInput) (any, error) {
			if len(args.Flow) == 0 {
				return nil, errInvalidParams("flow body is required")
			}
			payload, err := s.httpPost("/engine/flow", args.Flow)
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"engine_state",
		"Fetch the current engine state",
		func(*server.Context, any) (any, error) {
			payload, err := s.httpGet("/engine")
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"list_step_health",
		"List health status for all registered steps",
		func(*server.Context, any) (any, error) {
			payload, err := s.httpGet("/engine/health")
			return toolResult(payload, err)
		},
	)

	srv.Tool(
		"get_step_health",
		"Fetch health status for a single step",
		func(_ *server.Context, args getStepArgs) (any, error) {
			if args.ID == "" {
				return nil, errInvalidParams("id is required")
			}
			payload, err := s.httpGet("/engine/health/" + args.ID)
			return toolResult(payload, err)
		},
	)
}

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
