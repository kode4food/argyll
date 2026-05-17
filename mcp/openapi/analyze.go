package openapi

import argyllfacts "github.com/kode4food/argyll/mcp/openapi/argyll"

type (
	Args struct {
		Spec              *map[string]any `json:"spec,omitempty"`
		ExistingSteps     *map[string]any `json:"existing_steps,omitempty"`
		IncludeRegistered *bool           `json:"include_registered,omitempty"`
		SpecText          string          `json:"spec_text,omitempty"`
	}

	Step struct {
		InputsByType  map[string]string
		OutputsByType map[string]string
		ID            string
		Name          string
		Source        string
		Method        string
		Path          string
		Required      []string
		Optional      []string
		Outputs       []string
	}

	Info struct {
		Title       string `json:"title"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}

	Arg struct {
		Schema     *SchemaFacts `json:"schema,omitempty"`
		Name       string       `json:"name"`
		Type       string       `json:"type,omitempty"`
		Location   string       `json:"location,omitempty"`
		Confidence string       `json:"confidence,omitempty"`
		Service    string       `json:"service_name,omitempty"`
		Path       string       `json:"path,omitempty"`
		Required   bool         `json:"required,omitempty"`
	}

	SchemaFacts struct {
		Properties map[string]SchemaFacts `json:"properties,omitempty"`
		Type       string                 `json:"type,omitempty"`
		Required   []string               `json:"required,omitempty"`
		Enum       []any                  `json:"enum,omitempty"`
	}

	Capabilities       = argyllfacts.Capabilities
	MappingCapability  = argyllfacts.MappingCapability
	MatchCapability    = argyllfacts.MatchCapability
	EndpointCapability = argyllfacts.EndpointCapability

	Operation struct {
		ID          string   `json:"id"`
		Method      string   `json:"method,omitempty"`
		Path        string   `json:"path,omitempty"`
		Endpoint    string   `json:"endpoint,omitempty"`
		Summary     string   `json:"summary,omitempty"`
		Description string   `json:"description,omitempty"`
		Entity      string   `json:"entity,omitempty"`
		Inputs      []Arg    `json:"inputs"`
		Outputs     []Arg    `json:"outputs"`
		Rationale   []string `json:"rationale"`
		Ambiguities []string `json:"ambiguities"`
	}

	RegisteredStep struct {
		ID       string   `json:"id"`
		Name     string   `json:"name"`
		Source   string   `json:"source"`
		Path     string   `json:"path,omitempty"`
		Required []string `json:"required"`
		Optional []string `json:"optional"`
		Outputs  []string `json:"outputs"`
	}

	Result struct {
		Info          Info             `json:"info"`
		Mode          string           `json:"mode,omitempty"`
		BaseURL       string           `json:"base_url,omitempty"`
		LLMHandoff    string           `json:"llm_handoff_prompt,omitempty"`
		ExistingSteps []RegisteredStep `json:"existing_steps,omitempty"`
		Operations    []Operation      `json:"operations,omitempty"`
		Ambiguities   []string         `json:"ambiguities,omitempty"`
		Warnings      []string         `json:"warnings,omitempty"`
		Capabilities  Capabilities     `json:"argyll_capabilities"`
	}
)

func AnalyzeContract(
	args Args, existing []Step, warnings []string,
) (Result, error) {
	doc, err := parseDoc(args)
	if err != nil {
		return Result{}, err
	}

	ops := collectOperations(doc)
	var ambiguities []string
	for _, op := range ops {
		ambiguities = append(ambiguities, opAmbiguities(op)...)
	}
	info := infoPayload(doc)
	existingPayload := existingStepsPayload(existing)
	operations := operationsPayload(ops)
	capabilities := argyllfacts.Defaults()

	return Result{
		Capabilities:  capabilities,
		Mode:          "contract",
		Info:          info,
		BaseURL:       resolveBaseURL(doc),
		Warnings:      warnings,
		Ambiguities:   uniqueStrings(ambiguities),
		Operations:    operations,
		ExistingSteps: existingPayload,
		LLMHandoff: llmHandoffPrompt(handoffInput{
			Info:          info,
			Capabilities:  capabilities,
			Operations:    operations,
			ExistingSteps: existingPayload,
		}),
	}, nil
}

func coalesceType(s string) string {
	if s == "" {
		return "any"
	}
	return s
}
