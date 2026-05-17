package mcp

import (
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/mcp/openapi"
)

func (s *Server) analyzeOpenAPIContract(args openapi.Args) (any, error) {
	existing, warnings := s.collectExistingSteps(args)
	payload, err := openapi.AnalyzeContract(args, existing, warnings)
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
			"could not load registered steps from engine; analysis uses "+
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
