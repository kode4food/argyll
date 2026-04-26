package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/localrivet/gomcp/server"

	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/mcp/openapi"
)

type (
	getStepArgs struct {
		ID string `json:"id"`
	}

	getFlowArgs struct {
		ID string `json:"id"`
	}

	queryFlowsArgs struct {
		IDPrefix *string            `json:"id_prefix,omitempty"`
		Labels   *map[string]string `json:"labels,omitempty"`
		Statuses *[]string          `json:"statuses,omitempty"`
		Limit    *int               `json:"limit,omitempty"`
		Cursor   *string            `json:"cursor,omitempty"`
		Sort     *string            `json:"sort,omitempty"`
	}

	registerStepInput struct {
		Step map[string]any `json:"step"`
	}

	updateStepArgs struct {
		ID   string         `json:"id"`
		Step map[string]any `json:"step"`
	}

	diffProposedStepsArgs struct {
		Steps    *[]map[string]any `json:"steps,omitempty"`
		Proposal *map[string]any   `json:"proposal,omitempty"`
	}

	applyProposedStepsArgs struct {
		Steps        *[]map[string]any `json:"steps,omitempty"`
		Proposal     *map[string]any   `json:"proposal,omitempty"`
		ApplyUpdates *bool             `json:"apply_updates,omitempty"`
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
		"infer_openapi_steps",
		"Infer planner-oriented Argyll step drafts from an OpenAPI spec",
		func(_ *server.Context, args openapi.Args) (any, error) {
			return s.inferOpenAPI(args)
		},
	)

	srv.Tool(
		"analyze_service_spec",
		"Analyze one external REST/JSON service spec for planner-oriented use",
		func(_ *server.Context, args analyzeServiceSpecArgs) (any, error) {
			return s.analyzeServiceSpec(args)
		},
	)

	srv.Tool(
		"analyze_service_landscape",
		"Analyze multiple service specs and infer cross-service planning links",
		func(_ *server.Context, args analyzeServiceLandscapeArgs) (any, error) {
			return s.analyzeServiceLandscape(args)
		},
	)

	srv.Tool(
		"propose_bridge_steps",
		"Propose Lua bridge step drafts for missing cross-service planning "+
			"edges when declarative name mapping is not enough",
		func(_ *server.Context, args proposeBridgeStepsArgs) (any, error) {
			return s.proposeBridgeSteps(args)
		},
	)

	srv.Tool(
		"generate_step_impl",
		"Generate an SDK implementation draft for a proposed step, including "+
			"Lua script steps",
		func(_ *server.Context, args generateStepImplArgs) (any, error) {
			return s.generateStepImpl(args)
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
		"diff_proposed_steps",
		"Dry-run proposed step registrations against the live catalog",
		func(_ *server.Context, args diffProposedStepsArgs) (any, error) {
			return s.diffProposedSteps(args)
		},
	)

	srv.Tool(
		"apply_proposed_steps",
		"Apply proposed step registrations using existing register/update "+
			"operations",
		func(_ *server.Context, args applyProposedStepsArgs) (any, error) {
			return s.applyProposedSteps(args)
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
		"get_flow_status",
		"Fetch the current status for a single flow",
		func(_ *server.Context, args getFlowArgs) (any, error) {
			if args.ID == "" {
				return nil, errInvalidParams("id is required")
			}
			payload, err := s.httpGet("/engine/flow/" + args.ID + "/status")
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
		"query_flows",
		"Query flows by status, ID prefix, labels, and pagination",
		func(_ *server.Context, args queryFlowsArgs) (any, error) {
			payload, err := s.httpPost("/engine/flow/query", args)
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

func (s *Server) inferOpenAPI(args openapi.Args) (any, error) {
	existing, warnings := s.collectExistingSteps(args)
	payload, err := openapi.Analyze(args, existing, warnings)
	if err != nil {
		return nil, err
	}
	return toolResult(payload, nil)
}

func (s *Server) collectExistingSteps(
	args openapi.Args,
) ([]openapi.Step, []string) {
	var existing any
	if args.ExistingSteps != nil {
		existing = *args.ExistingSteps
	}
	res, warnings := normalizeExisting(existing)
	include := true
	if args.IncludeRegistered != nil {
		include = *args.IncludeRegistered
	}
	if !include {
		return res, warnings
	}

	payload, err := s.httpGet("/engine/step")
	if err != nil {
		warnings = append(
			warnings,
			"could not load registered steps from engine; inferred graph uses "+
				"provided steps only",
		)
		return res, warnings
	}
	loaded, moreWarnings := normalizeExisting(payload)
	warnings = append(warnings, moreWarnings...)
	if len(loaded) == 0 {
		return res, warnings
	}

	seen := util.Set[string]{}
	for _, st := range res {
		seen.Add(st.ID)
	}
	for _, st := range loaded {
		if seen.Contains(st.ID) {
			continue
		}
		res = append(res, st)
	}
	slices.SortFunc(res, func(a, b openapi.Step) int {
		return strings.Compare(a.ID, b.ID)
	})
	return res, warnings
}

func normalizeExisting(raw any) ([]openapi.Step, []string) {
	if raw == nil {
		return nil, nil
	}

	root, ok := asMap(raw)
	if !ok {
		return nil, []string{
			"existing_steps payload was not an object and was ignored",
		}
	}

	stepsRaw := root
	if nested, ok := asMap(root["steps"]); ok {
		stepsRaw = nested
	}

	var res []openapi.Step
	for id, item := range stepsRaw {
		st, ok := asMap(item)
		if !ok {
			continue
		}
		node := normalizeExistingStep(id, st)
		if node.ID == "" {
			continue
		}
		res = append(res, node)
	}

	slices.SortFunc(res, func(a, b openapi.Step) int {
		return strings.Compare(a.ID, b.ID)
	})
	return res, nil
}

func normalizeExistingStep(id string, st map[string]any) openapi.Step {
	attrs, _ := asMap(st["attributes"])
	res := openapi.Step{
		ID:            stringValue(st["id"]),
		Name:          stringValue(st["name"]),
		Source:        "existing",
		Required:      []string{},
		Optional:      []string{},
		Outputs:       []string{},
		InputsByType:  map[string]string{},
		OutputsByType: map[string]string{},
	}
	if res.ID == "" {
		res.ID = id
	}
	if httpCfg, ok := asMap(st["http"]); ok {
		res.Method = strings.ToUpper(stringValue(httpCfg["method"]))
		res.Path = stringValue(httpCfg["endpoint"])
	}

	for name, rawAttr := range attrs {
		attr, ok := asMap(rawAttr)
		if !ok {
			continue
		}
		role := stringValue(attr["role"])
		typ := coalesceType(stringValue(attr["type"]))
		switch role {
		case "required":
			res.Required = append(res.Required, name)
			res.InputsByType[name] = typ
		case "optional":
			res.Optional = append(res.Optional, name)
			res.InputsByType[name] = typ
		case "output":
			res.Outputs = append(res.Outputs, name)
			res.OutputsByType[name] = typ
		}
	}

	slices.Sort(res.Required)
	slices.Sort(res.Optional)
	slices.Sort(res.Outputs)
	return res
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
