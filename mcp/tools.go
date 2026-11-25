package mcp

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	listFlowsArgs struct {
		Limit  *int    `json:"limit,omitempty" jsonschema:"Maximum Flows to return"`
		Cursor *string `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page"`
	}

	queryFlowsArgs struct {
		IDPrefix *string   `json:"id_prefix,omitempty" jsonschema:"Flow ID prefix"`
		Tags     *[]string `json:"tags,omitempty" jsonschema:"Tags a Flow must all carry"`
		Statuses *[]string `json:"statuses,omitempty" jsonschema:"active, completed, or failed"`
		Limit    *int      `json:"limit,omitempty" jsonschema:"Maximum Flows to return"`
		Cursor   *string   `json:"cursor,omitempty" jsonschema:"next_cursor from the prior page"`
	}

	registerStepInput struct {
		Step map[string]any `json:"step"`
	}

	updateStepArgs struct {
		Step map[string]any `json:"step"`
		ID   string         `json:"id" jsonschema:"Step ID"`
	}

	diffProposedStepsArgs struct {
		Steps *[]map[string]any `json:"steps,omitempty"`
	}

	applyProposedStepsArgs struct {
		Steps        *[]map[string]any `json:"steps,omitempty"`
		ApplyUpdates *bool             `json:"apply_updates,omitempty"`
	}

	previewPlanInput struct {
		Init    api.InitArgs `json:"init,omitempty"`
		SpaceID api.SpaceID  `json:"space_id,omitempty" jsonschema:"Space to plan in"`
		Goals   api.Goals    `json:"goals"`
	}

	startFlowInput struct {
		Init    *map[string]any `json:"init,omitempty"`
		Tags    *[]string       `json:"tags,omitempty"`
		SpaceID api.SpaceID     `json:"space_id,omitempty" jsonschema:"Space to plan in"`
		Goals   api.Goals       `json:"goals"`
		ID      *string         `json:"id,omitempty" jsonschema:"Leave out for a fresh ID"`
	}
)

// Argument shapes the schema cannot express, spelled out in tool descriptions
const (
	goalsShape = `goals is {"steps": ["step-id"]}, with an optional "else" goal set tried ` +
		"when the first becomes impossible. Any Step can be a Goal, including one " +
		"without outputs"
	initShape = `init maps each Attribute name to an array of values, such as ` +
		`{"user_id": ["user-123"]}. Inputs left out use their defaults or come from ` +
		"other Steps"
	stepShape = "A Step has id, name, description, type (service, script, flow, or " +
		"agent), attributes, and the config object named by its type. attributes maps " +
		`each Attribute name to {"role": ...}, where role is required, optional, const, ` +
		"meta, feed, output, or metric. Copy the shape from get_step"
)

var (
	ErrInvalidParams = errors.New("invalid params")
	ErrToolSchema    = errors.New("could not build tool schema")
)

func (s *Server) registerTools(reg *toolRegistry) {
	s.registerEngineTools(reg)
	s.registerAnalysisTools(reg)
	s.registerStepTools(reg)
	s.registerFlowReadTools(reg)
	s.registerFlowWriteTools(reg)
	s.registerHealthTools(reg)
	s.registerSpaceTools(reg)
	s.registerFeedTools(reg)
}

func (s *Server) registerEngineTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name: "engine_state",
		Description: "Fetch the engine overview: registered Steps, Attributes, Step health, " +
			"and active Flows",
	},
		pathTool(s.httpGet, "/engine"),
	)
}

func (s *Server) registerAnalysisTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name:        "analyze_openapi_contract",
		Description: "Extract neutral REST contract facts from an OpenAPI spec",
	},
		s.analyzeOpenAPIContract,
	)
	addTool(reg, &mcp.Tool{
		Name:        "analyze_service_spec",
		Description: "Analyze one external REST/JSON service spec for planner-oriented use",
	},
		s.analyzeServiceSpec,
	)
	addTool(reg, &mcp.Tool{
		Name:        "analyze_service_landscape",
		Description: "Analyze multiple service specs and describe cross-service links",
	},
		s.analyzeServiceLandscape,
	)
	addTool(reg, &mcp.Tool{
		Name:        "generate_step_impl",
		Description: "Generate an SDK implementation draft for a proposed step",
	},
		s.generateStepImpl,
	)
}

func (s *Server) registerStepTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name:        "list_steps",
		Description: "List every registered Step with its Attributes and configuration",
	},
		pathTool(s.httpGet, "/engine/steps"),
	)
	addTool(reg, &mcp.Tool{
		Name: "get_step",
		Description: "Fetch one Step's registration: Attributes with roles, predicate, and " +
			"type configuration",
	},
		idTool(s.httpGet, "/engine/steps/%s"),
	)
	addTool(reg, &mcp.Tool{
		Name:        "unregister_step",
		Description: "Remove a Step from the catalog",
	},
		idTool(s.httpDelete, "/engine/steps/%s"),
	)
	addTool(reg, &mcp.Tool{
		Name:        "register_step",
		Description: "Register a new Step in the catalog. " + stepShape,
	},
		func(ctx context.Context, args registerStepInput) (any, error) {
			if len(args.Step) == 0 {
				return nil, errInvalidParams("step body is required")
			}
			return s.httpPost(ctx, "/engine/steps", args.Step)
		},
	)
	addTool(reg, &mcp.Tool{
		Name: "update_step",
		Description: "Replace an existing Step registration. Fetch it with get_step, change " +
			"it, and send the whole Step",
	},
		func(ctx context.Context, args updateStepArgs) (any, error) {
			if args.ID == "" || len(args.Step) == 0 {
				return nil, errInvalidParams("id and step body are required")
			}
			return s.httpPut(ctx, "/engine/steps/"+args.ID, args.Step)
		},
	)
	addTool(reg, &mcp.Tool{
		Name:        "diff_proposed_steps",
		Description: "Dry-run proposed step registrations against the live catalog",
	},
		s.diffProposedSteps,
	)
	addTool(reg, &mcp.Tool{
		Name: "apply_proposed_steps",
		Description: "Apply proposed step registrations using existing register/update " +
			"operations",
	},
		s.applyProposedSteps,
	)
}

func (s *Server) registerFlowReadTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name: "list_flows",
		Description: "List one page of Flows, newest first. Pass next_cursor as cursor for " +
			"the next page",
	},
		func(ctx context.Context, args listFlowsArgs) (any, error) {
			q := url.Values{}
			if args.Limit != nil {
				q.Set("limit", strconv.Itoa(*args.Limit))
			}
			if args.Cursor != nil {
				q.Set("cursor", *args.Cursor)
			}
			return s.httpGet(ctx, "/engine/flows?"+q.Encode())
		},
	)
	addTool(reg, &mcp.Tool{
		Name: "get_flow",
		Description: "Fetch a Flow's full state: plan, Attribute values, and each Step's " +
			"executions with inputs, outputs, and errors. Use it to see why a " +
			"Flow stalled or failed",
	},
		idTool(s.httpGet, "/engine/flows/%s"),
	)
	addTool(reg, &mcp.Tool{
		Name:        "get_flow_status",
		Description: "Fetch a Flow's status: active, completed, or failed",
	},
		idTool(s.httpGet, "/engine/flows/%s/status"),
	)
	addTool(reg, &mcp.Tool{
		Name: "get_flow_events",
		Description: "Fetch a Flow's event history in order, including Step starts, " +
			"skips, retries, completions, and failures",
	},
		idTool(s.httpGet, "/engine/flows/%s/events"),
	)
	addTool(reg, &mcp.Tool{
		Name:        "query_flows",
		Description: "Query Flows by status, ID prefix, and tags, newest first",
	},
		func(ctx context.Context, args queryFlowsArgs) (any, error) {
			return s.httpPost(ctx, "/engine/flows/query", args)
		},
	)
}

func (s *Server) registerFlowWriteTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name: "start_flow",
		Description: "Start a Flow that runs its Goal Steps. The plan includes every Step " +
			"needed to produce the Goals' inputs. The response names the Flow ID. " +
			"Never reuse an earlier ID. A started Flow has every input it needs. " +
			goalsShape + ". " + initShape,
	},
		func(ctx context.Context, args startFlowInput) (any, error) {
			if !args.Goals.Valid() {
				return nil, errInvalidParams("goals requires at least one non-empty goal set")
			}
			id := newFlowID(args.Goals)
			if args.ID != nil && *args.ID != "" {
				id = api.FlowID(*args.ID)
			}
			body := map[string]any{
				"id":    id,
				"goals": args.Goals,
				"init":  map[string]any{},
			}
			if args.Init != nil {
				body["init"] = *args.Init
			}
			if args.Tags != nil {
				body["tags"] = *args.Tags
			}
			if args.SpaceID != "" {
				body["space_id"] = args.SpaceID
			}
			return s.httpPost(ctx, "/engine/flows", body)
		},
	)
	addTool(reg, &mcp.Tool{
		Name: "preview_plan",
		Description: "Preview the plan for Goal Steps and initial Attribute values. Shows " +
			"the Steps that would run. The plan's required lists the only init values " +
			"the Flow needs. An optional input with no provider runs on its default, or " +
			"without the value. " + goalsShape + ". " + initShape,
	},
		func(ctx context.Context, args previewPlanInput) (any, error) {
			if !args.Goals.Valid() {
				return nil, errInvalidParams("goals requires at least one non-empty goal set")
			}
			init := args.Init
			if init == nil {
				init = api.InitArgs{}
			}
			return s.httpPost(ctx, "/engine/plan", api.ExecutionPlanRequest{
				Goals:   args.Goals,
				Init:    init,
				SpaceID: args.SpaceID,
			})
		},
	)
}

func (s *Server) registerHealthTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name:        "list_step_health",
		Description: "List health for every registered Step",
	},
		pathTool(s.httpGet, "/engine/health"),
	)
	addTool(reg, &mcp.Tool{
		Name:        "get_step_health",
		Description: "Fetch one Step's health",
	},
		idTool(s.httpGet, "/engine/health/%s"),
	)
}
