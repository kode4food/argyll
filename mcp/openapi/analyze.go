package openapi

import "slices"

type (
	Args struct {
		Mode              *string         `json:"mode,omitempty"`
		SpecText          string          `json:"spec_text,omitempty"`
		Spec              *map[string]any `json:"spec,omitempty"`
		ExistingSteps     *map[string]any `json:"existing_steps,omitempty"`
		IncludeRegistered *bool           `json:"include_registered,omitempty"`
	}

	Step struct {
		ID            string
		Name          string
		Source        string
		Method        string
		Path          string
		Required      []string
		Optional      []string
		Outputs       []string
		Step          map[string]any
		InputsByType  map[string]string
		OutputsByType map[string]string
		Rationale     []string
		Ambiguities   []string
		Entity        string
	}

	Info struct {
		Title       string `json:"title"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}

	Arg struct {
		Name       string `json:"name"`
		Type       string `json:"type,omitempty"`
		Required   bool   `json:"required,omitempty"`
		Location   string `json:"location,omitempty"`
		Confidence string `json:"confidence,omitempty"`
		Service    string `json:"service_name,omitempty"`
		Path       string `json:"path,omitempty"`
	}

	Operation struct {
		ID          string   `json:"id"`
		Method      string   `json:"method,omitempty"`
		Path        string   `json:"path,omitempty"`
		Summary     string   `json:"summary,omitempty"`
		Description string   `json:"description,omitempty"`
		Entity      string   `json:"entity,omitempty"`
		Inputs      []Arg    `json:"inputs"`
		Outputs     []Arg    `json:"outputs"`
		Rationale   []string `json:"rationale"`
		Ambiguities []string `json:"ambiguities"`
	}

	CandidateStep struct {
		ID                string         `json:"id"`
		Name              string         `json:"name,omitempty"`
		Source            string         `json:"source,omitempty"`
		Path              string         `json:"path,omitempty"`
		Entity            string         `json:"entity,omitempty"`
		Required          []string       `json:"required"`
		Optional          []string       `json:"optional"`
		Outputs           []string       `json:"outputs"`
		Rationale         []string       `json:"rationale"`
		Ambiguities       []string       `json:"ambiguities"`
		Step              map[string]any `json:"step,omitempty"`
		CoverageStatus    string         `json:"coverage_status,omitempty"`
		CoveredBy         []string       `json:"covered_by,omitempty"`
		CoverageRationale []string       `json:"coverage_rationale,omitempty"`
		CoverageMissing   []string       `json:"coverage_missing,omitempty"`
		CoverageOverlap   []string       `json:"coverage_overlap,omitempty"`
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
		Mode             string           `json:"mode,omitempty"`
		Info             Info             `json:"info"`
		BaseURL          string           `json:"base_url,omitempty"`
		Warnings         []string         `json:"warnings"`
		Ambiguities      []string         `json:"ambiguities"`
		Operations       []Operation      `json:"operations"`
		ExistingSteps    []RegisteredStep `json:"existing_steps"`
		CandidateSteps   []CandidateStep  `json:"candidate_steps"`
		RecommendedSteps []CandidateStep  `json:"recommended_steps"`
		Plans            []planSpec       `json:"plans"`
		Proposed         []CandidateStep  `json:"proposed_registrations"`
		ExamplePlans     []planSpec       `json:"example_plans"`
		LLMHandoff       string           `json:"llm_handoff_prompt,omitempty"`
	}
)

func Analyze(
	args Args, existing []Step, warnings []string,
) (Result, error) {
	doc, err := parseDoc(args)
	if err != nil {
		return Result{}, err
	}

	mode := normalizeMode(args.Mode)
	ops := collectOperations(doc)
	candidates := make([]Step, 0, len(ops))
	var ambiguities []string
	for _, op := range ops {
		node := buildCandidate(op)
		candidates = append(candidates, node)
		ambiguities = append(ambiguities, node.Ambiguities...)
	}

	covered := coverage(existing, candidates)
	recommended := make([]Step, 0, len(candidates))
	candidatePayload := make([]CandidateStep, 0, len(candidates))
	for _, c := range candidates {
		item := candidateStepPayload(c)
		cov := covered[c.ID]
		item.CoverageStatus = cov.Status
		item.CoveredBy = cov.CoveredBy
		item.CoverageRationale = cov.Rationale
		item.CoverageMissing = cov.Missing
		item.CoverageOverlap = cov.Overlap
		if !cov.Redundant {
			recommended = append(recommended, c)
		}
		candidatePayload = append(candidatePayload, item)
	}

	graph := append(slices.Clone(existing), recommended...)
	plans := inferPlans(graph)
	info := infoPayload(doc)
	existingPayload := existingStepsPayload(existing)
	recommendedPayload := recommendedStepsPayload(recommended)

	res := Result{
		Mode:             mode,
		Info:             info,
		BaseURL:          resolveBaseURL(doc),
		Warnings:         warnings,
		Ambiguities:      uniqueStrings(ambiguities),
		Operations:       operationsPayload(ops),
		ExistingSteps:    existingPayload,
		CandidateSteps:   candidatePayload,
		RecommendedSteps: recommendedPayload,
		Plans:            plans,
		LLMHandoff: llmHandoffPrompt(
			info, candidatePayload, existingPayload, plans,
		),
	}
	if mode == "propose_registrations" {
		res = Result{
			Mode:         mode,
			Warnings:     warnings,
			Ambiguities:  uniqueStrings(ambiguities),
			Proposed:     recommendedPayload,
			ExamplePlans: plans,
			LLMHandoff: llmHandoffPrompt(
				info, candidatePayload, existingPayload, plans,
			),
		}
	}
	return res, nil
}

func normalizeMode(mode *string) string {
	if mode == nil {
		return "analyze"
	}
	switch *mode {
	case "", "analyze":
		return "analyze"
	case "propose_registrations":
		return "propose_registrations"
	default:
		return "analyze"
	}
}

func buildCandidate(op opSpec) Step {
	node := Step{
		ID:            op.ID,
		Name:          op.Summary,
		Source:        "spec",
		Method:        op.Method,
		Path:          op.Path,
		Required:      []string{},
		Optional:      []string{},
		Outputs:       []string{},
		Step:          op.Step,
		InputsByType:  map[string]string{},
		OutputsByType: map[string]string{},
		Rationale:     opRationale(op),
		Ambiguities:   opAmbiguities(op),
		Entity:        op.Entity,
	}
	for _, in := range op.Inputs {
		if in.Required {
			node.Required = append(node.Required, in.Name)
		} else {
			node.Optional = append(node.Optional, in.Name)
		}
		node.InputsByType[in.Name] = coalesceType(in.Type)
	}
	for _, out := range op.Outputs {
		node.Outputs = append(node.Outputs, out.Name)
		node.OutputsByType[out.Name] = coalesceType(out.Type)
	}
	slices.Sort(node.Required)
	slices.Sort(node.Optional)
	slices.Sort(node.Outputs)
	return node
}

func coalesceType(s string) string {
	if s == "" {
		return "any"
	}
	return s
}
