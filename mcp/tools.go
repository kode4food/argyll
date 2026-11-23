package mcp

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/localrivet/gomcp/server"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/mcp/openapi"
)

type (
	listFlowsArgs struct {
		Limit  *int    `json:"limit,omitempty" description:"Maximum Flows to return"`
		Cursor *string `json:"cursor,omitempty" description:"next_cursor from the previous page"`
	}

	queryFlowsArgs struct {
		IDPrefix *string   `json:"id_prefix,omitempty" description:"Flow ID prefix"`
		Tags     *[]string `json:"tags,omitempty" description:"Tags a Flow must all carry"`
		Statuses *[]string `json:"statuses,omitempty" description:"active, completed, or failed"`
		Limit    *int      `json:"limit,omitempty" description:"Maximum Flows to return"`
		Cursor   *string   `json:"cursor,omitempty" description:"next_cursor from the prior page"`
	}

	registerStepInput struct {
		Step map[string]any `json:"step"`
	}

	updateStepArgs struct {
		Step map[string]any `json:"step"`
		ID   string         `json:"id" description:"Step ID"`
	}

	diffProposedStepsArgs struct {
		Steps *[]map[string]any `json:"steps,omitempty"`
	}

	applyProposedStepsArgs struct {
		Steps        *[]map[string]any `json:"steps,omitempty"`
		ApplyUpdates *bool             `json:"apply_updates,omitempty"`
	}

	previewPlanArgs struct {
		Init  map[string]any `json:"init,omitempty"`
		Goals api.Goals      `json:"goals"`
	}

	previewPlanInput struct {
		Init  *map[string]any `json:"init,omitempty"`
		Goals api.Goals       `json:"goals"`
	}

	startFlowInput struct {
		Init    *map[string]any `json:"init,omitempty"`
		Tags    *[]string       `json:"tags,omitempty"`
		SpaceID *string         `json:"space_id,omitempty" description:"Space to plan in"`
		Goals   api.Goals       `json:"goals"`
		ID      *string         `json:"id,omitempty" description:"Leave out for a fresh ID"`
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
)

func (s *Server) registerTools(srv server.Server) {
	s.registerEngineTools(srv)
	s.registerAnalysisTools(srv)
	s.registerStepTools(srv)
	s.registerFlowReadTools(srv)
	s.registerFlowWriteTools(srv)
	s.registerHealthTools(srv)
	s.registerSpaceTools(srv)
	s.registerFeedTools(srv)
}

func (s *Server) registerEngineTools(srv server.Server) {
	srv.Tool(
		"engine_state",
		"Fetch the engine overview: registered Steps, Attributes, Step health, "+
			"and active Flows",
		pathTool(s.httpGet, "/engine"),
	)
}

func (s *Server) registerAnalysisTools(srv server.Server) {
	srv.Tool(
		"analyze_openapi_contract",
		"Extract neutral REST contract facts from an OpenAPI spec",
		func(_ *server.Context, args openapi.Args) (any, error) {
			return s.analyzeOpenAPIContract(args)
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
		"Analyze multiple service specs and describe cross-service links",
		func(_ *server.Context, args analyzeServiceLandscapeArgs) (any, error) {
			return s.analyzeServiceLandscape(args)
		},
	)
	srv.Tool(
		"generate_step_impl",
		"Generate an SDK implementation draft for a proposed step",
		func(_ *server.Context, args generateStepImplArgs) (any, error) {
			return s.generateStepImpl(args)
		},
	)
}

func (s *Server) registerStepTools(srv server.Server) {
	srv.Tool(
		"list_steps",
		"List every registered Step with its Attributes and configuration",
		pathTool(s.httpGet, "/engine/steps"),
	)
	srv.Tool(
		"get_step",
		"Fetch one Step's registration: Attributes with roles, predicate, and "+
			"type configuration",
		idTool(s.httpGet, "/engine/steps/", ""),
	)
	srv.Tool(
		"unregister_step",
		"Remove a Step from the catalog",
		idTool(s.httpDelete, "/engine/steps/", ""),
	)
	srv.Tool(
		"register_step",
		"Register a new Step in the catalog. "+stepShape,
		func(_ *server.Context, args registerStepInput) (any, error) {
			if len(args.Step) == 0 {
				return nil, errInvalidParams("step body is required")
			}
			return toolResult(s.httpPost("/engine/steps", args.Step))
		},
	)
	srv.Tool(
		"update_step",
		"Replace an existing Step registration. Fetch it with get_step, change "+
			"it, and send the whole Step",
		func(_ *server.Context, args updateStepArgs) (any, error) {
			if args.ID == "" || len(args.Step) == 0 {
				return nil, errInvalidParams("id and step body are required")
			}
			return toolResult(s.httpPut("/engine/steps/"+args.ID, args.Step))
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
}

func (s *Server) registerFlowReadTools(srv server.Server) {
	srv.Tool(
		"list_flows",
		"List one page of Flows, newest first. Pass next_cursor as cursor for "+
			"the next page",
		func(_ *server.Context, args listFlowsArgs) (any, error) {
			q := url.Values{}
			if args.Limit != nil {
				q.Set("limit", strconv.Itoa(*args.Limit))
			}
			if args.Cursor != nil {
				q.Set("cursor", *args.Cursor)
			}
			return toolResult(s.httpGet("/engine/flows?" + q.Encode()))
		},
	)
	srv.Tool(
		"get_flow",
		"Fetch a Flow's full state: plan, Attribute values, and each Step's "+
			"executions with inputs, outputs, and errors. Use it to see why a "+
			"Flow stalled or failed",
		idTool(s.httpGet, "/engine/flows/", ""),
	)
	srv.Tool(
		"get_flow_status",
		"Fetch a Flow's status: active, completed, or failed",
		idTool(s.httpGet, "/engine/flows/", "/status"),
	)
	srv.Tool(
		"get_flow_events",
		"Fetch a Flow's event history in order, including Step starts, "+
			"skips, retries, completions, and failures",
		idTool(s.httpGet, "/engine/flows/", "/events"),
	)
	srv.Tool(
		"query_flows",
		"Query Flows by status, ID prefix, and tags, newest first",
		func(_ *server.Context, args queryFlowsArgs) (any, error) {
			return toolResult(s.httpPost("/engine/flows/query", args))
		},
	)
}

func (s *Server) registerFlowWriteTools(srv server.Server) {
	srv.Tool(
		"start_flow",
		"Start a Flow that runs its Goal Steps. The plan includes every Step "+
			"needed to produce the Goals' inputs. The response names the Flow ID. "+
			"Never reuse an earlier ID. "+goalsShape+". "+initShape,
		func(_ *server.Context, args startFlowInput) (any, error) {
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
			if args.SpaceID != nil {
				body["space_id"] = *args.SpaceID
			}
			return toolResult(s.httpPost("/engine/flows", body))
		},
	)
	srv.Tool(
		"preview_plan",
		"Preview the plan for Goal Steps and initial Attribute values. Shows "+
			"the Steps that would run and the inputs still missing. "+goalsShape+". "+
			initShape,
		func(_ *server.Context, args previewPlanInput) (any, error) {
			if !args.Goals.Valid() {
				return nil, errInvalidParams("goals requires at least one non-empty goal set")
			}
			init := map[string]any{}
			if args.Init != nil {
				init = *args.Init
			}
			return toolResult(s.httpPost("/engine/plan", previewPlanArgs{
				Goals: args.Goals,
				Init:  init,
			}))
		},
	)
}

func (s *Server) registerHealthTools(srv server.Server) {
	srv.Tool(
		"list_step_health",
		"List health for every registered Step",
		pathTool(s.httpGet, "/engine/health"),
	)
	srv.Tool(
		"get_step_health",
		"Fetch one Step's health",
		idTool(s.httpGet, "/engine/health/", ""),
	)
}
