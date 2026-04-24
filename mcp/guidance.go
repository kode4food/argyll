package mcp

import (
	"bytes"
	"embed"
	"strconv"
	"strings"
	"text/template"

	"github.com/localrivet/gomcp/server"
)

type (
	sdkStepTemplateInput struct {
		Language       string   `json:"language"`
		StepName       string   `json:"step_name"`
		StepType       string   `json:"step_type,omitempty"`
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

//go:embed guidance/*.md guidance/*.tmpl
var guidanceFS embed.FS

func (s *Server) registerGuidance(srv server.Server) {
	srv.Resource(
		"/sdk/steps",
		"Guidance for implementing Argyll steps with the Go and Python SDKs",
		func(*server.Context, any) (any, error) {
			return readGuidance("sdk-steps.md")
		},
	)

	srv.Resource(
		"/sdk/go/steps",
		"Go SDK patterns for implementing Argyll steps",
		func(*server.Context, any) (any, error) {
			return readGuidance("go-steps.md")
		},
	)

	srv.Resource(
		"/sdk/python/steps",
		"Python SDK patterns for implementing Argyll steps",
		func(*server.Context, any) (any, error) {
			return readGuidance("python-steps.md")
		},
	)

	srv.Prompt(
		"implement_step",
		"Guide an agent through implementing an Argyll step with the SDKs",
		server.User(mustReadGuidance("implement-step-prompt.md")),
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

func readGuidance(name string) (string, error) {
	raw, err := guidanceFS.ReadFile("guidance/" + name)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func mustReadGuidance(name string) string {
	text, err := readGuidance(name)
	if err != nil {
		panic(err)
	}
	return text
}

func sdkStepTemplate(args sdkStepTemplateInput) (string, error) {
	lang := strings.ToLower(strings.TrimSpace(args.Language))
	if lang != "go" && lang != "python" {
		return "", errInvalidParams("language must be go or python")
	}
	if strings.TrimSpace(args.StepName) == "" {
		return "", errInvalidParams("step_name is required")
	}
	stepType := strings.ToLower(strings.TrimSpace(args.StepType))
	if stepType == "" {
		stepType = "sync"
	}
	switch stepType {
	case "sync", "async", "external", "script":
	default:
		return "", errInvalidParams(
			"step_type must be sync, async, external, or script",
		)
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
		method = ""
	}
	if stepType != "external" && stepType != "script" && method != "POST" {
		return "", errInvalidParams(
			"non-POST methods require step_type external",
		)
	}
	data := stepTemplateData{
		StepName:         args.StepName,
		Method:           method,
		Inputs:           args.Inputs,
		Outputs:          args.Outputs,
		IsAsync:          stepType == "async",
		IsExternal:       stepType == "external",
		IsScript:         stepType == "script",
		HasNonPostMethod: method != "POST" && stepType == "external",
	}
	if args.ScriptLanguage != nil {
		data.ScriptLanguage = *args.ScriptLanguage
	}
	if args.ScriptBody != nil {
		data.ScriptBody = *args.ScriptBody
	}
	if lang == "go" {
		return renderStepTemplate("go-step.tmpl", data)
	}
	return renderStepTemplate("python-step.tmpl", data)
}

func renderStepTemplate(name string, data stepTemplateData) (string, error) {
	raw, err := guidanceFS.ReadFile("guidance/" + name)
	if err != nil {
		return "", err
	}
	tpl, err := template.New(name).Funcs(template.FuncMap{
		"quote": strconv.Quote,
	}).Parse(string(raw))
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := tpl.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}
