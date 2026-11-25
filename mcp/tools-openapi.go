package mcp

import (
	"context"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/mcp/openapi"
)

func (s *Server) analyzeOpenAPIContract(
	ctx context.Context, args openapi.Args,
) (any, error) {
	existing, warnings := s.collectExistingSteps(ctx, args)
	payload, err := openapi.AnalyzeContract(args, existing, warnings)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Server) collectExistingSteps(
	ctx context.Context, args openapi.Args,
) ([]openapi.Step, []string) {
	res := normalizeExisting(args.ExistingSteps)
	if args.IncludeRegistered != nil && !*args.IncludeRegistered {
		return res, nil
	}

	loaded, err := s.currentSteps(ctx)
	if err != nil {
		return res, []string{
			"could not load registered steps from engine; analysis uses provided steps only",
		}
	}

	seen := util.Set[string]{}
	for _, st := range res {
		seen.Add(st.ID)
	}
	for id, st := range loaded {
		if !seen.Contains(id) {
			res = append(res, normalizeExistingStep(id, st))
		}
	}
	slices.SortFunc(res, func(a, b openapi.Step) int {
		return strings.Compare(a.ID, b.ID)
	})
	return res, nil
}

func normalizeExisting(steps map[string]any) []openapi.Step {
	var res []openapi.Step
	for id, item := range steps {
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
	return res
}

func normalizeExistingStep(id string, st map[string]any) openapi.Step {
	attrs, _ := asMap(st["attributes"])
	res := openapi.Step{
		ID:       stringValue(st["id"]),
		Name:     stringValue(st["name"]),
		Source:   "existing",
		Required: []string{},
		Optional: []string{},
		Outputs:  []string{},
	}
	if res.ID == "" {
		res.ID = id
	}
	if httpCfg, ok := asMap(st["http"]); ok {
		if invoke, ok := asMap(httpCfg["invoke"]); ok {
			res.Method = strings.ToUpper(stringValue(invoke["method"]))
			res.Path = stringValue(invoke["endpoint"])
		}
	}

	for name, rawAttr := range attrs {
		attr, ok := asMap(rawAttr)
		if !ok {
			continue
		}
		switch stringValue(attr["role"]) {
		case "required":
			res.Required = append(res.Required, name)
		case "optional":
			res.Optional = append(res.Optional, name)
		case "output":
			res.Outputs = append(res.Outputs, name)
		}
	}

	slices.Sort(res.Required)
	slices.Sort(res.Optional)
	slices.Sort(res.Outputs)
	return res
}
