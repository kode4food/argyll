package mcp

import (
	"context"
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
		Name     string         `json:"name,omitempty" jsonschema:"Service name"`
		Spec     map[string]any `json:"spec,omitempty" jsonschema:"OpenAPI object"`
		SpecText string         `json:"spec_text,omitempty" jsonschema:"OpenAPI text"`
	}

	analyzeServiceSpecArgs struct {
		Name          string         `json:"name,omitempty" jsonschema:"Service name"`
		Spec          map[string]any `json:"spec,omitempty" jsonschema:"OpenAPI object"`
		ExistingSteps map[string]any `json:"existing_steps,omitempty" jsonschema:"Steps by ID"`
		IncludeReg    *bool          `json:"include_registered,omitempty"`
		SpecText      string         `json:"spec_text,omitempty" jsonschema:"OpenAPI text"`
	}

	analyzeServiceLandscapeArgs struct {
		ExistingSteps map[string]any     `json:"existing_steps,omitempty" jsonschema:"Steps by ID"`
		IncludeReg    *bool              `json:"include_registered,omitempty"`
		Services      []serviceSpecInput `json:"services" jsonschema:"Service specs"`
	}

	generateStepImplArgs struct {
		Step     map[string]any `json:"step" jsonschema:"Step registration"`
		Language string         `json:"language" jsonschema:"go or python"`
	}
)

func (s *Server) analyzeServiceSpec(
	ctx context.Context, args analyzeServiceSpecArgs,
) (any, error) {
	specArgs := openapi.Args{
		SpecText:          args.SpecText,
		Spec:              args.Spec,
		ExistingSteps:     args.ExistingSteps,
		IncludeRegistered: args.IncludeReg,
	}
	existing, warnings := s.collectExistingSteps(ctx, specArgs)
	spec, err := openapi.AnalyzeContract(specArgs, existing, warnings)
	if err != nil {
		return nil, err
	}

	name := serviceName(args.Name, spec)
	return specResult{
		ServiceName: name,
		Summary:     summaryPayload(name, spec),
		Analysis:    spec,
	}, nil
}

func (s *Server) analyzeServiceLandscape(
	ctx context.Context, args analyzeServiceLandscapeArgs,
) (any, error) {
	if len(args.Services) == 0 {
		return nil, errInvalidParams("services is required")
	}

	existing, warnings := s.collectExistingSteps(ctx, openapi.Args{
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
			IncludeRegistered: new(false),
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
	return landscapeResult{
		Services:        services,
		RegisteredSteps: existingCatalogPayload(existing),
		Relationships:   relationships(nodes),
		MissingAttrs:    missing,
		Warnings:        uniqueStrings(allWarnings),
		Ambiguities:     uniqueStrings(ambiguities),
	}, nil
}

func serviceName(name string, spec openapi.Result) string {
	if name = strings.TrimSpace(name); name != "" {
		return name
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
