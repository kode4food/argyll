package mcp

import (
	"strings"

	"github.com/localrivet/gomcp/server"

	guide "github.com/kode4food/argyll/mcp/internal/guidance"
)

type (
	sdkStepTemplateInput struct {
		Language       string   `json:"language"`
		StepName       string   `json:"step_name"`
		StepType       string   `json:"step_type,omitempty"`
		ActionMode     *string  `json:"action_mode,omitempty"`
		External       *bool    `json:"external,omitempty"`
		Method         string   `json:"method,omitempty"`
		ScriptLanguage *string  `json:"script_language,omitempty"`
		ScriptBody     *string  `json:"script_body,omitempty"`
		Inputs         []string `json:"inputs,omitempty"`
		Outputs        []string `json:"outputs,omitempty"`
	}

	stepTemplateData struct {
		StepName         string
		Method           string
		ScriptLanguage   string
		ScriptBody       string
		Inputs           []string
		Outputs          []string
		IsAsync          bool
		IsExternal       bool
		IsScript         bool
		HasNonPostMethod bool
	}
)

func (s *Server) registerGuidance(srv server.Server) {
	srv.Resource(
		"/sdk/steps",
		"Guidance for implementing Argyll steps with the Go and Python SDKs",
		func(*server.Context, any) (any, error) {
			return guide.Read("sdk-steps.md")
		},
	)

	srv.Resource(
		"/sdk/go/steps",
		"Go SDK patterns for implementing Argyll steps",
		func(*server.Context, any) (any, error) {
			return guide.Read("go-steps.md")
		},
	)

	srv.Resource(
		"/sdk/python/steps",
		"Python SDK patterns for implementing Argyll steps",
		func(*server.Context, any) (any, error) {
			return guide.Read("python-steps.md")
		},
	)

	srv.Resource(
		"/openapi/ingestion",
		"Guidance for ingesting OpenAPI services into Argyll",
		func(*server.Context, any) (any, error) {
			return guide.Read("openapi-ingestion.md")
		},
	)

	srv.Prompt(
		"implement_step",
		"Guide an agent through implementing an Argyll step with the SDKs",
		server.User(guide.MustRead("implement-step-prompt.md")),
	)

	srv.Prompt(
		"ingest_openapi_services",
		"Guide an agent through ingesting bespoke OpenAPI services",
		server.User(guide.MustRead("openapi-ingestion.md")),
	)

	srv.Tool(
		"sdk_step_template",
		"Return a minimal Go or Python Argyll SDK step implementation template",
		func(_ *server.Context, args sdkStepTemplateInput) (any, error) {
			code, err := sdkStepTemplate(args)
			if err != nil {
				return nil, err
			}
			return toolResult(map[string]any{
				"language": args.Language,
				"code":     code,
			}, nil)
		},
	)
}

func sdkStepTemplate(args sdkStepTemplateInput) (string, error) {
	isExternal := args.External != nil && *args.External
	return renderStepTemplate(args, isExternal)
}

func renderStepTemplate(
	args sdkStepTemplateInput, isExternal bool,
) (string, error) {
	lang := strings.ToLower(strings.TrimSpace(args.Language))
	if lang != "go" && lang != "python" {
		return "", errInvalidParams("language must be go or python")
	}
	if strings.TrimSpace(args.StepName) == "" {
		return "", errInvalidParams("step_name is required")
	}
	stepType := strings.ToLower(strings.TrimSpace(args.StepType))
	if stepType == "" {
		stepType = "service"
	}
	switch stepType {
	case "service", "script":
	case "flow":
		return "", errInvalidParams(
			"flow steps do not have SDK handler implementations",
		)
	default:
		return "", errInvalidParams(
			"step_type must identify an SDK-implemented service or script step",
		)
	}
	actionMode := ""
	if args.ActionMode != nil {
		actionMode = strings.ToLower(strings.TrimSpace(*args.ActionMode))
	}
	if actionMode == "" {
		actionMode = "sync"
	}
	if actionMode != "sync" && actionMode != "async" {
		return "", errInvalidParams("action_mode must be sync or async")
	}
	method := strings.ToUpper(strings.TrimSpace(args.Method))
	if method == "" {
		method = "POST"
	}
	switch method {
	case "GET", "POST", "PUT", "DELETE":
	default:
		return "", errInvalidParams("method must be GET, POST, PUT, or DELETE")
	}
	if stepType == "script" {
		if isExternal {
			return "", errInvalidParams(
				"external is only valid for service steps",
			)
		}
		if args.ActionMode != nil {
			return "", errInvalidParams(
				"action_mode is only valid for service steps",
			)
		}
		method = ""
	}
	if !isExternal && stepType != "script" && method != "POST" {
		return "", errInvalidParams(
			"SDK-hosted service templates require method POST",
		)
	}
	data := stepTemplateData{
		StepName:         args.StepName,
		Method:           method,
		Inputs:           args.Inputs,
		Outputs:          args.Outputs,
		IsAsync:          actionMode == "async",
		IsExternal:       isExternal,
		IsScript:         stepType == "script",
		HasNonPostMethod: method != "POST" && isExternal,
	}
	if args.ScriptLanguage != nil {
		data.ScriptLanguage = *args.ScriptLanguage
	}
	if args.ScriptBody != nil {
		data.ScriptBody = *args.ScriptBody
	}
	if lang == "go" {
		return guide.RenderTemplate("go-step.tmpl", data)
	}
	return guide.RenderTemplate("python-step.tmpl", data)
}
