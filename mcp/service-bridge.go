package mcp

import (
	"fmt"
	"slices"
	"strings"
)

type (
	bridgeProposal struct {
		ID                 string         `json:"id"`
		Kind               string         `json:"kind"`
		SourceAttr         string         `json:"source_attribute"`
		TargetAttr         string         `json:"target_attribute"`
		SourceService      string         `json:"source_service"`
		TargetService      string         `json:"target_service"`
		Confidence         string         `json:"confidence"`
		Rationale          string         `json:"rationale"`
		ImplementationHint string         `json:"implementation_hint"`
		Step               map[string]any `json:"step"`
	}

	bridgeProposalResult struct {
		Proposed []bridgeProposal `json:"proposed_bridge_steps"`
	}

	stepImplResult struct {
		StepName            string   `json:"step_name"`
		StepType            string   `json:"step_type"`
		Method              string   `json:"method"`
		Inputs              []string `json:"inputs"`
		Outputs             []string `json:"outputs"`
		ImplementationNotes []string `json:"implementation_notes"`
		Code                string   `json:"code"`
	}
)

func (s *Server) proposeBridgeSteps(args proposeBridgeStepsArgs) (any, error) {
	landscape, err := s.landscapePayload(args)
	if err != nil {
		return nil, err
	}

	proposals := make([]bridgeProposal, 0, len(landscape.BridgeOpps))
	for _, opp := range landscape.BridgeOpps {
		if opp.TargetAttr == "" || opp.SourceAttr == "" {
			continue
		}
		step := scriptBridgeStep(opp)
		proposals = append(proposals, bridgeProposal{
			ID:                 stringValue(step["id"]),
			Kind:               opp.Kind,
			SourceAttr:         opp.SourceAttr,
			TargetAttr:         opp.TargetAttr,
			SourceService:      opp.SourceService,
			TargetService:      opp.TargetService,
			Confidence:         opp.Confidence,
			Rationale:          opp.Rationale,
			ImplementationHint: bridgeHint(opp.Kind),
			Step:               step,
		})
	}
	return toolResult(bridgeProposalResult{Proposed: proposals}, nil)
}

func (s *Server) generateStepImpl(args generateStepImplArgs) (any, error) {
	if len(args.Step) == 0 {
		return nil, errInvalidParams("step is required")
	}

	inputs, outputs := stepIO(args.Step)
	stepType := "sync"
	method := "POST"
	bridgeKind := ""
	var scriptLang *string
	var scriptBody *string
	if labels, ok := asMap(args.Step["labels"]); ok {
		bridgeKind = stringValue(labels["argyll.bridge_kind"])
	}
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
	if stepType == "script" && (scriptBody == nil || *scriptBody == "") {
		body := bridgeScriptBody(inputs, outputs)
		scriptBody = &body
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
		StepName: name,
		StepType: stepType,
		Method:   method,
		Inputs:   inputs,
		Outputs:  outputs,
		ImplementationNotes: bridgeImplNotes(
			bridgeKind, stepType, method, inputs, outputs,
		),
		Code: code,
	}, nil)
}

func scriptBridgeStep(opp bridgeOpportunity) map[string]any {
	id := "bridge-" + strings.ReplaceAll(
		opp.SourceAttr+"-to-"+opp.TargetAttr, "_", "-",
	)
	return map[string]any{
		"id":   id,
		"name": humanizeID(id),
		"type": "script",
		"labels": map[string]any{
			"argyll.source":      "bridge-proposal",
			"argyll.bridge_kind": opp.Kind,
		},
		"attributes": map[string]any{
			opp.SourceAttr: map[string]any{
				"role": "required",
				"type": opp.SourceType,
			},
			opp.TargetAttr: map[string]any{
				"role": "output",
				"type": opp.TargetType,
			},
		},
		"script": map[string]any{
			"language": "lua",
			"script": bridgeScriptBody(
				[]string{opp.SourceAttr}, []string{opp.TargetAttr},
			),
		},
	}
}

func bridgeScriptBody(inputs, outputs []string) string {
	if len(outputs) == 0 {
		return "return {}"
	}
	var b strings.Builder
	b.WriteString("return {\n")
	for i, out := range outputs {
		src := "nil"
		if len(inputs) > 0 {
			src = inputs[0]
			if i < len(inputs) {
				src = inputs[i]
			}
		}
		fmt.Fprintf(&b, "  %s = %s,\n", out, src)
	}
	b.WriteString("}")
	return b.String()
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

func stepTypes(raw any) map[string]string {
	step, ok := asMap(raw)
	if !ok {
		return map[string]string{}
	}
	attrs, ok := asMap(step["attributes"])
	if !ok {
		return map[string]string{}
	}
	res := map[string]string{}
	for name, rawAttr := range attrs {
		attr, ok := asMap(rawAttr)
		if !ok {
			continue
		}
		res[name] = coalesceType(stringValue(attr["type"]))
	}
	return res
}

func bridgeHint(kind string) string {
	switch kind {
	default:
		return "prefer declarative name mappings first; otherwise use a Lua " +
			"script step to reshape the value"
	}
}

func bridgeImplNotes(
	kind, stepType, method string, inputs, outputs []string,
) []string {
	var res []string
	if stepType == "script" {
		res = append(res,
			"Prefer declarative name mappings first; use Lua only for the "+
				"value reshaping the mapping layer cannot express.",
		)
	}
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
	if kind != "" {
		res = append(res, "Bridge kind: "+kind)
	}
	return res
}

func humanizeID(id string) string {
	parts := strings.Split(id, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func cloneMap(src any) map[string]any {
	in, ok := asMap(src)
	if !ok || len(in) == 0 {
		return nil
	}
	res := make(map[string]any, len(in))
	for k, v := range in {
		res[k] = cloneValue(v)
	}
	return res
}

func cloneValue(src any) any {
	switch v := src.(type) {
	case map[string]any:
		return cloneMap(v)
	case []any:
		res := make([]any, 0, len(v))
		for _, item := range v {
			res = append(res, cloneValue(item))
		}
		return res
	case []map[string]any:
		res := make([]any, 0, len(v))
		for _, item := range v {
			res = append(res, cloneMap(item))
		}
		return res
	default:
		return src
	}
}
