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

	previewPlanArgs struct {
		Goals []string       `json:"goals"`
		Init  map[string]any `json:"init,omitempty"`
	}

	previewPlanInput struct {
		Goals []string        `json:"goals"`
		Init  *map[string]any `json:"init,omitempty"`
	}
)

var (
	ErrInvalidParams = errors.New("invalid params")
)

func (s *Server) registerTools(srv server.Server) {
	srv.Tool(
		"list_steps",
		"List registered steps in the engine",
		func(_ *server.Context, _ any) (any, error) {
			payload, err := s.httpGet("/engine/step")
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
		"engine_state",
		"Fetch the current engine state",
		func(_ *server.Context, _ any) (any, error) {
			payload, err := s.httpGet("/engine")
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
