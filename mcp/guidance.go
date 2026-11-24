package mcp

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

const markdownType = "text/markdown"

func (s *Server) registerGuidance(reg *toolRegistry) {
	srv := reg.server
	addGuide(srv, &mcp.Resource{
		URI:         "/sdk/steps",
		Description: "Guidance for implementing Argyll steps with the Go and Python SDKs",
	}, "sdk-steps.md")
	addGuide(srv, &mcp.Resource{
		URI:         "/sdk/go/steps",
		Description: "Go SDK patterns for implementing Argyll steps",
	}, "go-steps.md")
	addGuide(srv, &mcp.Resource{
		URI:         "/sdk/python/steps",
		Description: "Python SDK patterns for implementing Argyll steps",
	}, "python-steps.md")
	addGuide(srv, &mcp.Resource{
		URI:         "/openapi/ingestion",
		Description: "Guidance for ingesting OpenAPI services into Argyll",
	}, "openapi-ingestion.md")

	addPrompt(srv, &mcp.Prompt{
		Name:        "implement_step",
		Description: "Guide an agent through implementing an Argyll step with the SDKs",
	}, "implement-step-prompt.md", "language", "step_name", "requirements")
	addPrompt(srv, &mcp.Prompt{
		Name:        "ingest_openapi_services",
		Description: "Guide an agent through ingesting bespoke OpenAPI services",
	}, "openapi-ingestion.md", "payload")

	addTool(reg, &mcp.Tool{
		Name:        "sdk_step_template",
		Description: "Return a minimal Go or Python Argyll SDK step implementation template",
	},
		func(_ context.Context, args sdkStepTemplateInput) (any, error) {
			code, err := sdkStepTemplate(args)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"language": args.Language,
				"code":     code,
			}, nil
		},
	)
}

func addGuide(srv *mcp.Server, res *mcp.Resource, file string) {
	res.Name = res.URI
	res.MIMEType = markdownType
	read := func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		text, err := guide.Read(file)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI:      res.URI,
			MIMEType: markdownType,
			Text:     text,
		}}}, nil
	}
	srv.AddResource(res, read)
}

// addPrompt serves a guidance file as a user prompt, filling each {{argument}} placeholder
func addPrompt(srv *mcp.Server, p *mcp.Prompt, file string, argNames ...string) {
	args := make([]*mcp.PromptArgument, 0, len(argNames))
	for _, n := range argNames {
		args = append(args, &mcp.PromptArgument{Name: n})
	}
	p.Arguments = args
	read := func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		text, err := guide.Read(file)
		if err != nil {
			return nil, err
		}
		for k, v := range req.Params.Arguments {
			text = strings.ReplaceAll(text, "{{"+k+"}}", v)
		}
		return &mcp.GetPromptResult{Messages: []*mcp.PromptMessage{{
			Role:    "user",
			Content: &mcp.TextContent{Text: text},
		}}}, nil
	}
	srv.AddPrompt(p, read)
}

func sdkStepTemplate(args sdkStepTemplateInput) (string, error) {
	isExternal := args.External != nil && *args.External
	return renderStepTemplate(args, isExternal)
}

func renderStepTemplate(args sdkStepTemplateInput, isExternal bool) (string, error) {
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
		return "", errInvalidParams("flow steps do not have SDK handler implementations")
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
			return "", errInvalidParams("external is only valid for service steps")
		}
		if args.ActionMode != nil {
			return "", errInvalidParams("action_mode is only valid for service steps")
		}
		method = ""
	}
	if !isExternal && stepType != "script" && method != "POST" {
		return "", errInvalidParams("SDK-hosted service templates require method POST")
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
