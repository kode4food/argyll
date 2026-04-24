package mcp

import "github.com/kode4food/argyll/mcp/openapi"

type (
	serviceSpecInput struct {
		Name     *string         `json:"name,omitempty"`
		SpecText string          `json:"spec_text,omitempty"`
		Spec     *map[string]any `json:"spec,omitempty"`
	}

	analyzeServiceSpecArgs struct {
		Name          *string         `json:"name,omitempty"`
		SpecText      string          `json:"spec_text,omitempty"`
		Spec          *map[string]any `json:"spec,omitempty"`
		ExistingSteps *map[string]any `json:"existing_steps,omitempty"`
		IncludeReg    *bool           `json:"include_registered,omitempty"`
	}

	analyzeServiceLandscapeArgs struct {
		Services      []serviceSpecInput `json:"services"`
		ExistingSteps *map[string]any    `json:"existing_steps,omitempty"`
		IncludeReg    *bool              `json:"include_registered,omitempty"`
	}

	proposeBridgeStepsArgs struct {
		Landscape     *map[string]any     `json:"landscape,omitempty"`
		Services      *[]serviceSpecInput `json:"services,omitempty"`
		ExistingSteps *map[string]any     `json:"existing_steps,omitempty"`
		IncludeReg    *bool               `json:"include_registered,omitempty"`
	}

	generateStepImplArgs struct {
		Step     map[string]any `json:"step"`
		Language string         `json:"language"`
	}
)

func (s *Server) analyzeServiceSpec(args analyzeServiceSpecArgs) (any, error) {
	specArgs := openapi.Args{
		SpecText:          args.SpecText,
		Spec:              args.Spec,
		ExistingSteps:     args.ExistingSteps,
		IncludeRegistered: args.IncludeReg,
	}
	existing, warnings := s.collectExistingSteps(specArgs)
	spec, err := analyzeSpecPayload(specArgs, existing, warnings)
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
		spec, err := analyzeSpecPayload(openapi.Args{
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
		Relationships:   inferRelationships(nodes),
		MissingAttrs:    missing,
		BridgeOpps:      bridgeOpportunities(nodes, missing),
		Warnings:        uniqueStrings(allWarnings),
		Ambiguities:     uniqueStrings(ambiguities),
	}, nil)
}

func (s *Server) landscapePayload(
	args proposeBridgeStepsArgs,
) (*landscapeResult, error) {
	if args.Landscape != nil {
		return decodeLandscape(*args.Landscape)
	}
	if args.Services == nil || len(*args.Services) == 0 {
		return nil, errInvalidParams("landscape or services is required")
	}

	payload, err := s.analyzeServiceLandscape(analyzeServiceLandscapeArgs{
		Services:      *args.Services,
		ExistingSteps: args.ExistingSteps,
		IncludeReg:    args.IncludeReg,
	})
	if err != nil {
		return nil, err
	}
	return decodeLandscapeToolResult(payload)
}

func boolPtr(v bool) *bool {
	return &v
}
