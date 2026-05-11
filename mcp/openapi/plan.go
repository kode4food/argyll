package openapi

import (
	"net/http"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/util"
)

type planSpec struct {
	GoalStepID    string           `json:"goal_step_id"`
	GoalStepName  string           `json:"goal_step_name"`
	GoalSource    string           `json:"goal_source"`
	Steps         []map[string]any `json:"steps"`
	MissingInputs []string         `json:"missing_inputs"`
	SuggestedInit map[string]any   `json:"suggested_init"`
}

func buildAttributes(inputs, outputs []argSpec) map[string]any {
	res := map[string]any{}
	for _, in := range dedupeArgs(inputs) {
		attr := map[string]any{
			"role": requiredRole(in.Required),
			"type": coalesceType(in.Type),
		}
		if in.Service != "" && in.Service != in.Name {
			attr["mapping"] = map[string]any{"name": in.Service}
		}
		res[in.Name] = attr
	}
	for _, out := range dedupeArgs(outputs) {
		attr := map[string]any{
			"role": "output",
			"type": coalesceType(out.Type),
		}
		switch {
		case out.Path != "":
			attr["mapping"] = map[string]any{
				"script": map[string]any{
					"language": "jpath",
					"script":   out.Path,
				},
			}
		case out.Service != "" && out.Service != out.Name:
			attr["mapping"] = map[string]any{"name": out.Service}
		}
		res[out.Name] = attr
	}
	return res
}

func inferPlans(nodes []Step) []planSpec {
	providers := map[string][]Step{}
	for _, st := range nodes {
		for _, out := range st.Outputs {
			providers[out] = append(providers[out], st)
		}
	}
	for name := range providers {
		slices.SortFunc(providers[name], func(a, b Step) int {
			if a.Source != b.Source {
				if a.Source == "existing" {
					return -1
				}
				return 1
			}
			if len(a.Required) != len(b.Required) {
				return len(a.Required) - len(b.Required)
			}
			return strings.Compare(a.ID, b.ID)
		})
	}

	goals := slices.Clone(nodes)
	slices.SortFunc(goals, func(a, b Step) int {
		if goalRank(a) != goalRank(b) {
			return goalRank(b) - goalRank(a)
		}
		return strings.Compare(a.ID, b.ID)
	})

	var res []planSpec
	for _, goal := range goals {
		plan := buildPlan(goal, providers, util.Set[string]{})
		if len(plan.Steps) == 0 {
			continue
		}
		res = append(res, plan)
		if len(res) == 8 {
			break
		}
	}
	return res
}

func buildPlan(
	goal Step, providers map[string][]Step, visiting util.Set[string],
) planSpec {
	steps := map[string]Step{}
	missing := map[string]string{}

	var walk func(Step)
	walk = func(curr Step) {
		if visiting.Contains(curr.ID) {
			return
		}
		visiting.Add(curr.ID)
		steps[curr.ID] = curr
		for _, in := range curr.Required {
			prov := providers[in]
			if len(prov) == 0 {
				missing[in] = curr.InputsByType[in]
				continue
			}
			walk(prov[0])
		}
		delete(visiting, curr.ID)
	}

	walk(goal)
	ordered := make([]Step, 0, len(steps))
	for _, st := range steps {
		ordered = append(ordered, st)
	}
	slices.SortFunc(ordered, func(a, b Step) int {
		if deps(a, steps) != deps(b, steps) {
			return deps(a, steps) - deps(b, steps)
		}
		return strings.Compare(a.ID, b.ID)
	})

	stepPayload := make([]map[string]any, 0, len(ordered))
	for _, st := range ordered {
		stepPayload = append(stepPayload, map[string]any{
			"id":       st.ID,
			"name":     st.Name,
			"source":   st.Source,
			"required": st.Required,
			"outputs":  st.Outputs,
		})
	}
	missingNames := make([]string, 0, len(missing))
	suggested := map[string]any{}
	for name, typ := range missing {
		missingNames = append(missingNames, name)
		suggested[name] = map[string]any{"type": typ}
	}
	slices.Sort(missingNames)

	return planSpec{
		GoalStepID:    goal.ID,
		GoalStepName:  goal.Name,
		GoalSource:    goal.Source,
		Steps:         stepPayload,
		MissingInputs: missingNames,
		SuggestedInit: suggested,
	}
}

func deps(st Step, all map[string]Step) int {
	n := 0
	for _, in := range st.Required {
		for _, cand := range all {
			if slices.Contains(cand.Outputs, in) {
				n++
				break
			}
		}
	}
	return n
}

func goalRank(st Step) int {
	switch st.Method {
	case http.MethodPost, http.MethodPut, http.MethodDelete:
		return 3
	case http.MethodGet:
		if len(st.Required) > 0 {
			return 2
		}
		return 1
	default:
		return 0
	}
}

func criticalOutputs(st Step) []string {
	var res []string
	for _, name := range st.Outputs {
		if strings.HasSuffix(name, "_id") {
			res = append(res, name)
		}
	}
	if len(res) == 0 {
		return slices.Clone(st.Outputs)
	}
	return res
}
