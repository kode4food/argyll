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
		Limit  int    `json:"limit,omitempty" jsonschema:"Maximum Flows to return"`
		Cursor string `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page"`
	}

	queryFlowsArgs struct {
		Tags     api.Tags         `json:"tags,omitempty" jsonschema:"Tags a Flow must all carry"`
		IDPrefix string           `json:"id_prefix,omitempty" jsonschema:"Flow ID prefix"`
		Cursor   string           `json:"cursor,omitempty" jsonschema:"Previous page's next_cursor"`
		Statuses []api.FlowStatus `json:"statuses,omitempty" jsonschema:"active, completed, failed"`
		Limit    int              `json:"limit,omitempty" jsonschema:"Maximum Flows to return"`
	}

	registerStepInput struct {
		Step map[string]any `json:"step" jsonschema:"Step registration"`
	}

	updateStepArgs struct {
		Step map[string]any `json:"step" jsonschema:"Whole Step registration"`
		ID   api.StepID     `json:"id" jsonschema:"Step ID"`
	}

	diffProposedStepsArgs struct {
		Steps []map[string]any `json:"steps,omitempty" jsonschema:"Proposed Steps"`
	}

	applyProposedStepsArgs struct {
		Steps       []map[string]any `json:"steps,omitempty" jsonschema:"Proposed Steps"`
		SkipUpdates bool             `json:"skip_updates,omitempty" jsonschema:"Skip changed Steps"`
	}

	previewPlanInput struct {
		Init    api.InitArgs `json:"init,omitempty" jsonschema:"Initial Attribute values"`
		SpaceID api.SpaceID  `json:"space_id,omitempty" jsonschema:"Omit to plan with all Steps"`
		Goals   api.Goals    `json:"goals" jsonschema:"Goal sets to reach"`
	}

	startFlowInput struct {
		Init    api.InitArgs `json:"init,omitempty" jsonschema:"Initial Attribute values"`
		Tags    api.Tags     `json:"tags,omitempty" jsonschema:"Tags to find the Flow by"`
		SpaceID api.SpaceID  `json:"space_id,omitempty" jsonschema:"Omit to plan with all Steps"`
		Goals   api.Goals    `json:"goals" jsonschema:"Goal sets to reach"`
		ID      api.FlowID   `json:"id,omitempty" jsonschema:"Leave out for a fresh ID"`
	}
)

// Argument shapes the schema cannot express, spelled out in tool descriptions
const (
	goalsShape = `goals is {"steps": ["step-id"]}, with an optional "else" goal set tried ` +
		"when the first becomes impossible. Any Step can be a Goal, including one without outputs"
	initShape = "init maps each Attribute name to an array of values, such as " +
		`{"user_id": ["user-123"]}. A present key freezes those values and prunes providers. ` +
		"An empty array declares zero resolved values; it does not request a default. Omit a " +
		"key to allow providers or optional fallback. See /engine/semantics for collect policies"
	stepShape = "A Step has id, name, description, type (service, script, flow, or agent), " +
		"attributes, and the config object named by its type. attributes maps each Attribute " +
		`name to {"role": ...}, where role is required, optional, const, meta, feed, output, or ` +
		"metric. Copy the shape from get_step"
	registeredShape = "Registered Steps join existing_steps unless include_registered is false"
)

var (
	ErrInvalidParams   = errors.New("invalid params")
	ErrToolSchema      = errors.New("could not build tool schema")
	ErrDuplicateStepID = errors.New("duplicate proposed step ID")
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
		Description: "Fetch registered Steps, Spaces, Attributes, and health. Use query_flows " +
			"with status active to find active Flows",
	},
		pathTool(s.httpGet, "/engine"),
	)
	addTool(reg, &mcp.Tool{
		Name: "agent_capabilities",
		Description: "Discover available agent providers, models, efforts, and MCP servers/tools " +
			"before configuring an agent Step",
	}, pathTool(s.httpGet, "/engine/agents/capabilities"))
}

func (s *Server) registerAnalysisTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name:        "analyze_openapi_contract",
		Description: "Extract neutral REST contract facts from an OpenAPI spec. " + registeredShape,
	},
		s.analyzeOpenAPIContract,
	)
	addTool(reg, &mcp.Tool{
		Name: "analyze_service_spec",
		Description: "Analyze one external REST/JSON service spec for planner-oriented use. " +
			registeredShape,
	},
		s.analyzeServiceSpec,
	)
	addTool(reg, &mcp.Tool{
		Name: "analyze_service_landscape",
		Description: "Analyze multiple service specs and describe cross-service links. " +
			registeredShape,
	},
		s.analyzeServiceLandscape,
	)
	addTool(reg, &mcp.Tool{
		Name: "generate_step_impl",
		Description: "Generate Go or Python registration code preserving a proposed Step. " +
			"Service endpoints must already be implemented; use sdk_step_template for a handler " +
			"scaffold. Includes the unchanged registration and implementation notes",
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
		Description: "Fetch one Step's registration: Attributes with roles, predicate, and type " +
			"configuration",
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
		Description: "Replace an existing Step registration. Fetch it with get_step, change it, " +
			"and send the whole Step",
	},
		func(ctx context.Context, args updateStepArgs) (any, error) {
			if args.ID == "" || len(args.Step) == 0 {
				return nil, errInvalidParams("id and step body are required")
			}
			return s.httpPut(ctx, "/engine/steps/"+string(args.ID), args.Step)
		},
	)
	addTool(reg, &mcp.Tool{
		Name: "diff_proposed_steps",
		Description: "Compare proposals with the catalog, normalizing tag order and duplicates. " +
			"Work config is compared as supplied; omitted defaults can appear as changes. " +
			"Requires canonical IDs. Does not validate registration or execution",
	},
		s.diffProposedSteps,
	)
	addTool(reg, &mcp.Tool{
		Name: "apply_proposed_steps",
		Description: "Apply proposed step registrations using existing register/update " +
			"operations. Set skip_updates to true to register new Steps only. Not atomic: errors " +
			"report applied, skipped, failed, and pending operations. Read the catalog before " +
			"retrying; a failed request may have committed. Verification covers mappings, " +
			"matches, and metric selectors, not runtime behavior",
	},
		s.applyProposedSteps,
	)
}

func (s *Server) registerFlowReadTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name: "list_flows",
		Description: "List one page of Flows, newest first. Pass next_cursor as cursor for the " +
			"next page",
	},
		func(ctx context.Context, args listFlowsArgs) (any, error) {
			q := url.Values{}
			if args.Limit != 0 {
				q.Set("limit", strconv.Itoa(args.Limit))
			}
			if args.Cursor != "" {
				q.Set("cursor", args.Cursor)
			}
			return s.httpGet(ctx, "/engine/flows?"+q.Encode())
		},
	)
	addTool(reg, &mcp.Tool{
		Name: "get_flow",
		Description: "Fetch a Flow's full state: plan, Attribute values, and each Step's " +
			"executions with inputs, outputs, and errors. Use it to see why a Flow stalled or " +
			"failed",
	},
		idTool(s.httpGet, "/engine/flows/%s"),
	)
	addTool(reg, &mcp.Tool{
		Name: "get_flow_status",
		Description: "Fetch a Flow's outcome: active, completed, or failed. Terminal status does " +
			"not mean work and compensation have settled; inspect get_flow deactivated_at",
	},
		idTool(s.httpGet, "/engine/flows/%s/status"),
	)
	addTool(reg, &mcp.Tool{
		Name: "get_flow_events",
		Description: "Fetch a Flow's event history in order, including Step starts, skips, " +
			"retries, completions, and failures",
	},
		idTool(s.httpGet, "/engine/flows/%s/events"),
	)
	addTool(reg, &mcp.Tool{
		Name:        "query_flows",
		Description: "Query Flows by status, ID prefix, and tags, newest first",
	},
		func(ctx context.Context, args queryFlowsArgs) (any, error) {
			return s.httpPost(ctx, "/engine/flows/query", api.QueryFlowsRequest(args))
		},
	)
}

func (s *Server) registerFlowWriteTools(reg *toolRegistry) {
	addTool(reg, &mcp.Tool{
		Name: "start_flow",
		Description: "Start a Flow that runs its Goal Steps. The plan includes every Step needed " +
			"to produce the Goals' inputs. The response names the Flow ID. Never reuse an " +
			"earlier ID. Acceptance does not guarantee runtime collection will succeed. " +
			goalsShape + ". " + initShape,
	},
		func(ctx context.Context, args startFlowInput) (any, error) {
			if !args.Goals.Valid() {
				return nil, errInvalidParams("goals requires at least one non-empty goal set")
			}
			id := args.ID
			if id == "" {
				id = newFlowID(args.Goals)
			}
			if args.Init == nil {
				args.Init = api.InitArgs{}
			}
			return s.httpPost(ctx, "/engine/flows", api.CreateFlowRequest{
				Init: args.Init, SpaceID: args.SpaceID, Goals: args.Goals,
				ID:   id,
				Tags: args.Tags,
			})
		},
	)
	addTool(reg, &mcp.Tool{
		Name: "preview_plan",
		Description: "Preview the plan for Goal Steps and initial Attribute values. Shows the " +
			"Steps selected for execution. The plan's required lists external input names, " +
			"not a guarantee of runtime success. An optional input can use its default or omit a " +
			"value. " + goalsShape + ". " + initShape,
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
