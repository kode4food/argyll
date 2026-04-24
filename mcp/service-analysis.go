package mcp

import (
	"encoding/json"
	"strings"

	"github.com/kode4food/argyll/mcp/openapi"
)

type (
	serviceSummary struct {
		Name             string `json:"name"`
		Operations       int    `json:"operations"`
		CandidateSteps   int    `json:"candidate_steps"`
		RecommendedSteps int    `json:"recommended_steps"`
		Plans            int    `json:"plans"`
	}

	serviceAnalysis struct {
		Name     string         `json:"name"`
		Summary  serviceSummary `json:"summary"`
		Analysis openapi.Result `json:"analysis"`
	}

	landscapeResult struct {
		Services        []serviceAnalysis   `json:"services"`
		RegisteredSteps []catalogStep       `json:"registered_steps"`
		Relationships   []relationship      `json:"relationships"`
		MissingAttrs    []missingAttr       `json:"missing_attributes"`
		BridgeOpps      []bridgeOpportunity `json:"bridge_opportunities"`
		Warnings        []string            `json:"warnings"`
		Ambiguities     []string            `json:"ambiguities"`
	}

	specResult struct {
		ServiceName string         `json:"service_name"`
		Summary     serviceSummary `json:"summary"`
		Analysis    openapi.Result `json:"analysis"`
	}
)

func decodeLandscape(raw map[string]any) (*landscapeResult, error) {
	buf, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return decodeLandscapeJSON(buf)
}

func decodeLandscapeToolResult(raw any) (*landscapeResult, error) {
	text, err := toolTextPayload(raw)
	if err != nil {
		return nil, err
	}
	return decodeLandscapeJSON([]byte(text))
}

func decodeLandscapeJSON(raw []byte) (*landscapeResult, error) {
	var res landscapeResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, errInvalidParams("landscape payload was invalid")
	}
	return &res, nil
}

func toolTextPayload(raw any) (string, error) {
	root, ok := asMap(raw)
	if !ok {
		return "", errInvalidParams("landscape payload was not an object")
	}
	content, ok := root["content"].([]any)
	if !ok || len(content) == 0 {
		return "", errInvalidParams(
			"landscape payload was not valid tool output",
		)
	}
	item, ok := asMap(content[0])
	if !ok {
		return "", errInvalidParams(
			"landscape payload content was not an object",
		)
	}
	return stringValue(item["text"]), nil
}

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
		Name:             name,
		Operations:       len(spec.Operations),
		CandidateSteps:   len(spec.CandidateSteps),
		RecommendedSteps: len(spec.RecommendedSteps),
		Plans:            len(spec.Plans),
	}
}

func analyzeSpecPayload(
	args openapi.Args, existing []openapi.Step, warnings []string,
) (openapi.Result, error) {
	return openapi.Analyze(args, existing, warnings)
}
