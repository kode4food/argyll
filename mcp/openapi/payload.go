package openapi

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	openapi "github.com/getkin/kin-openapi/openapi3"
)

func operationsPayload(ops []opSpec) []Operation {
	res := make([]Operation, 0, len(ops))
	for _, op := range ops {
		res = append(res, Operation{
			ID:          op.ID,
			Method:      op.Method,
			Path:        op.Path,
			Summary:     op.Summary,
			Description: op.Description,
			Entity:      op.Entity,
			Inputs:      payloadArgs(op.Inputs),
			Outputs:     payloadArgs(op.Outputs),
			Rationale:   opRationale(op),
			Ambiguities: opAmbiguities(op),
		})
	}
	return res
}

func existingStepsPayload(steps []Step) []RegisteredStep {
	res := make([]RegisteredStep, 0, len(steps))
	for _, st := range steps {
		res = append(res, RegisteredStep{
			ID:       st.ID,
			Name:     st.Name,
			Source:   st.Source,
			Path:     st.Path,
			Required: st.Required,
			Optional: st.Optional,
			Outputs:  st.Outputs,
		})
	}
	return res
}

func recommendedStepsPayload(steps []Step) []CandidateStep {
	res := make([]CandidateStep, 0, len(steps))
	for _, st := range steps {
		res = append(res, candidateStepPayload(st))
	}
	return res
}

func candidateStepPayload(st Step) CandidateStep {
	return CandidateStep{
		ID:          st.ID,
		Name:        st.Name,
		Source:      st.Source,
		Path:        st.Path,
		Entity:      st.Entity,
		Required:    st.Required,
		Optional:    st.Optional,
		Outputs:     st.Outputs,
		Rationale:   st.Rationale,
		Ambiguities: st.Ambiguities,
		Step:        st.Step,
	}
}

func payloadArgs(args []argSpec) []Arg {
	res := make([]Arg, 0, len(args))
	for _, arg := range args {
		item := Arg{
			Name:       arg.Name,
			Type:       coalesceType(arg.Type),
			Required:   arg.Required,
			Location:   arg.Location,
			Confidence: arg.Confidence,
		}
		if arg.Service != "" {
			item.Service = arg.Service
		}
		if arg.Path != "" {
			item.Path = arg.Path
		}
		res = append(res, item)
	}
	return res
}

func opRationale(op opSpec) []string {
	var res []string
	if op.Entity != "" {
		res = append(res, "canonicalized operation around entity "+op.Entity)
	}
	if len(op.Inputs) != 0 {
		res = append(
			res,
			fmt.Sprintf("declares %d inferred inputs", len(op.Inputs)),
		)
	}
	if len(op.Outputs) != 0 {
		res = append(
			res,
			fmt.Sprintf(
				"exposes %d inferred outputs for planning",
				len(op.Outputs),
			),
		)
	}
	if op.Method == "POST" || op.Method == "PUT" || op.Method == "DELETE" {
		res = append(
			res,
			"treated as goal-like because it mutates remote state",
		)
	}
	return res
}

func opAmbiguities(op opSpec) []string {
	var res []string
	if op.Entity == "" {
		res = append(
			res,
			"could not confidently infer a canonical entity from path "+op.Path,
		)
	}
	for _, in := range op.Inputs {
		if in.Confidence == "low" {
			res = append(
				res,
				"low-confidence canonical input name for service field "+
					in.Service,
			)
		}
	}
	return uniqueStrings(res)
}

func resolveBaseURL(doc *openapi.T) string {
	if len(doc.Servers) == 0 || doc.Servers[0] == nil {
		return ""
	}
	url := doc.Servers[0].URL
	for name, variable := range doc.Servers[0].Variables {
		if variable == nil {
			continue
		}
		url = strings.ReplaceAll(url, "{"+name+"}", variable.Default)
	}
	return strings.TrimRight(url, "/")
}

func joinURL(baseURL, path string) string {
	if baseURL == "" {
		return path
	}
	return strings.TrimRight(baseURL, "/") + path
}

func infoPayload(doc *openapi.T) Info {
	if doc.Info == nil {
		return Info{}
	}
	return Info{
		Title:       doc.Info.Title,
		Version:     doc.Info.Version,
		Description: doc.Info.Description,
	}
}

func dedupeArgs(args []argSpec) []argSpec {
	seen := map[string]argSpec{}
	for _, arg := range args {
		key := arg.Name + "|" + arg.Location
		prev, ok := seen[key]
		if !ok || (!prev.Required && arg.Required) {
			seen[key] = arg
		}
	}
	res := make([]argSpec, 0, len(seen))
	for _, arg := range seen {
		res = append(res, arg)
	}
	slices.SortFunc(res, func(a, b argSpec) int {
		if cmp := strings.Compare(a.Name, b.Name); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Location, b.Location)
	})
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

func llmHandoffPrompt(
	info Info, candidateSteps []CandidateStep, existingSteps []RegisteredStep,
	plans []planSpec,
) string {
	data, _ := json.Marshal(struct {
		Info           Info             `json:"info"`
		CandidateSteps []CandidateStep  `json:"candidate_steps"`
		ExistingSteps  []RegisteredStep `json:"existing_steps"`
		Plans          []planSpec       `json:"plans"`
	}{
		Info:           info,
		CandidateSteps: candidateSteps,
		ExistingSteps:  existingSteps,
		Plans:          plans,
	})
	return "Review this normalized OpenAPI-to-Argyll graph. Refine canonical " +
		"attribute names, mark ambiguous planning edges, and suggest which " +
		"candidate registrations should be kept, merged, or dropped:\n" +
		string(data)
}
