package mcp

import (
	"slices"
	"strings"
)

type (
	stepImplResult struct {
		StepName            string   `json:"step_name"`
		StepType            string   `json:"step_type"`
		Method              string   `json:"method"`
		Code                string   `json:"code"`
		Inputs              []string `json:"inputs"`
		Outputs             []string `json:"outputs"`
		ImplementationNotes []string `json:"implementation_notes"`
	}
)

func (s *Server) generateStepImpl(args generateStepImplArgs) (any, error) {
	if len(args.Step) == 0 {
		return nil, errInvalidParams("step is required")
	}

	inputs, outputs := stepIO(args.Step)
	stepType := "sync"
	method := "POST"
	var scriptLang *string
	var scriptBody *string
	if scriptCfg, ok := asMap(args.Step["script"]); ok {
		stepType = "script"
		method = ""
		lang := strings.ToLower(stringValue(scriptCfg["language"]))
		body := stringValue(scriptCfg["script"])
		if lang == "" {
			lang = "lua"
		}
		scriptLang = &lang
		scriptBody = &body
	}
	if httpCfg, ok := asMap(args.Step["http"]); ok {
		stepType = "external"
		if m := strings.ToUpper(stringValue(httpCfg["method"])); m != "" {
			method = m
		}
	}

	name := stringValue(args.Step["name"])
	if name == "" {
		name = stringValue(args.Step["id"])
	}
	code, err := sdkStepTemplate(sdkStepTemplateInput{
		Language:       args.Language,
		StepName:       name,
		StepType:       stepType,
		Method:         method,
		ScriptLanguage: scriptLang,
		ScriptBody:     scriptBody,
		Inputs:         inputs,
		Outputs:        outputs,
	})
	if err != nil {
		return nil, err
	}

	return toolResult(stepImplResult{
		StepName:            name,
		StepType:            stepType,
		Method:              method,
		Inputs:              inputs,
		Outputs:             outputs,
		ImplementationNotes: stepImplNotes(stepType, method, inputs, outputs),
		Code:                code,
	}, nil)
}

func stepIO(step map[string]any) ([]string, []string) {
	attrs, ok := asMap(step["attributes"])
	if !ok {
		return nil, nil
	}
	var inputs []string
	var outputs []string
	for name, raw := range attrs {
		attr, ok := asMap(raw)
		if !ok {
			continue
		}
		switch stringValue(attr["role"]) {
		case "required":
			inputs = append(inputs, name)
		case "output":
			outputs = append(outputs, name)
		}
	}
	slices.Sort(inputs)
	slices.Sort(outputs)
	return inputs, outputs
}

func stepImplNotes(stepType, method string, inputs, outputs []string) []string {
	var res []string
	if stepType == "external" {
		res = append(res,
			"Use the external service contract already embedded in the "+
				"proposed step instead of inventing a new transport.",
		)
	}
	if method != "POST" {
		res = append(res,
			"This step uses a non-POST HTTP method; map arguments to path "+
				"or query fields carefully.",
		)
	}
	if len(inputs) == 0 {
		res = append(res, "This draft has no required inputs; verify that.")
	}
	if len(outputs) == 0 {
		res = append(res, "This draft has no outputs; verify that.")
	}
	return res
}
