package mcp

import (
	"strings"

	"github.com/kode4food/argyll/mcp/openapi"
)

type (
	serviceSummary struct {
		Name       string `json:"name"`
		Operations int    `json:"operations"`
	}

	serviceAnalysis struct {
		Analysis openapi.Result `json:"analysis"`
		Name     string         `json:"name"`
		Summary  serviceSummary `json:"summary"`
	}

	landscapeResult struct {
		Services        []serviceAnalysis `json:"services"`
		RegisteredSteps []catalogStep     `json:"registered_steps"`
		Relationships   []relationship    `json:"relationships"`
		MissingAttrs    []missingAttr     `json:"missing_attributes"`
		Warnings        []string          `json:"warnings"`
		Ambiguities     []string          `json:"ambiguities"`
	}

	specResult struct {
		Analysis    openapi.Result `json:"analysis"`
		ServiceName string         `json:"service_name"`
		Summary     serviceSummary `json:"summary"`
	}

	serviceSpecInput struct {
		Name     *string         `json:"name,omitempty"`
		Spec     *map[string]any `json:"spec,omitempty"`
		SpecText string          `json:"spec_text,omitempty"`
	}

	analyzeServiceSpecArgs struct {
		Name          *string         `json:"name,omitempty"`
		Spec          *map[string]any `json:"spec,omitempty"`
		ExistingSteps *map[string]any `json:"existing_steps,omitempty"`
		IncludeReg    *bool           `json:"include_registered,omitempty"`
		SpecText      string          `json:"spec_text,omitempty"`
	}

	analyzeServiceLandscapeArgs struct {
		ExistingSteps *map[string]any    `json:"existing_steps,omitempty"`
		IncludeReg    *bool              `json:"include_registered,omitempty"`
		Services      []serviceSpecInput `json:"services"`
	}

	generateStepImplArgs struct {
		Step     map[string]any `json:"step"`
		Language string         `json:"language"`
	}
)

func serviceName(name *string, spec openapi.Result) string {
	if name != nil && strings.TrimSpace(*name) != "" {
		return strings.TrimSpace(*name)
	}
	title := strings.TrimSpace(spec.Info.Title)
	if title == "" {
		return "service"
	}
	return title
}

func summaryPayload(name string, spec openapi.Result) serviceSummary {
	return serviceSummary{
		Name:       name,
		Operations: len(spec.Operations),
	}
}

func (s *Server) analyzeServiceSpec(args analyzeServiceSpecArgs) (any, error) {
	specArgs := openapi.Args{
		SpecText:          args.SpecText,
		Spec:              args.Spec,
		ExistingSteps:     args.ExistingSteps,
		IncludeRegistered: args.IncludeReg,
	}
	existing, warnings := s.collectExistingSteps(specArgs)
	spec, err := openapi.AnalyzeContract(specArgs, existing, warnings)
	if err != nil {
		return nil, err
	}

	name := serviceName(args.Name, spec)
	return toolResult(specResult{
		ServiceName: name,
		Summary:     summaryPayload(name, spec),
		Analysis:    spec,
	}, nil)
}

func (s *Server) analyzeServiceLandscape(
	args analyzeServiceLandscapeArgs,
) (any, error) {
	if len(args.Services) == 0 {
		return nil, errInvalidParams("services is required")
	}

	existing, warnings := s.collectExistingSteps(openapi.Args{
		ExistingSteps:     args.ExistingSteps,
		IncludeRegistered: args.IncludeReg,
	})

	services := make([]serviceAnalysis, 0, len(args.Services))
	nodes := landscapeNodes(existing)
	var ambiguities []string
	allWarnings := append([]string{}, warnings...)

	for _, svc := range args.Services {
		spec, err := openapi.AnalyzeContract(openapi.Args{
			SpecText:          svc.SpecText,
			Spec:              svc.Spec,
			IncludeRegistered: boolPtr(false),
		}, nil, nil)
		if err != nil {
			return nil, err
		}
		name := serviceName(svc.Name, spec)
		services = append(services, serviceAnalysis{
			Name:     name,
			Summary:  summaryPayload(name, spec),
			Analysis: spec,
		})
		nodes = append(nodes, serviceNodes(name, spec)...)
		allWarnings = append(allWarnings, spec.Warnings...)
		ambiguities = append(ambiguities, spec.Ambiguities...)
	}

	missing := missingAttributes(nodes)
	return toolResult(landscapeResult{
		Services:        services,
		RegisteredSteps: existingCatalogPayload(existing),
		Relationships:   relationships(nodes),
		MissingAttrs:    missing,
		Warnings:        uniqueStrings(allWarnings),
		Ambiguities:     uniqueStrings(ambiguities),
	}, nil)
}

func boolPtr(v bool) *bool {
	return &v
}
