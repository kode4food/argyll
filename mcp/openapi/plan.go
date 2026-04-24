package openapi

import (
	"net/http"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	coverageDetail struct {
		Status    string
		CoveredBy []string
		Rationale []string
		Missing   []string
		Overlap   []string
		Redundant bool
	}

	planSpec struct {
		GoalStepID    string           `json:"goal_step_id"`
		GoalStepName  string           `json:"goal_step_name"`
		GoalSource    string           `json:"goal_source"`
		Steps         []map[string]any `json:"steps"`
		MissingInputs []string         `json:"missing_inputs"`
		SuggestedInit map[string]any   `json:"suggested_init"`
	}
)

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

func coverage(existing, candidates []Step) map[string]coverageDetail {
	res := map[string]coverageDetail{}
	for _, cand := range candidates {
		best := coverageDetail{Status: "recommended"}
		for _, ex := range existing {
			d := compareCoverage(cand, ex)
			if coverageRank(d.Status) > coverageRank(best.Status) {
				best = d
			}
		}
		if len(best.CoveredBy) != 0 {
			slices.Sort(best.CoveredBy)
		}
		res[cand.ID] = best
	}
	return res
}

func compareCoverage(cand, ex Step) coverageDetail {
	crit := criticalOutputs(cand)
	overlap := intersect(cand.Outputs, ex.Outputs)
	missingReq := diff(cand.Required, ex.Required)
	missingCrit := diff(crit, ex.Outputs)
	switch {
	case len(missingReq) == 0 && len(missingCrit) == 0:
		if slices.Equal(cand.Required, ex.Required) &&
			slices.Equal(cand.Outputs, ex.Outputs) {
			return coverageDetail{
				Status:    "covered_exact",
				CoveredBy: []string{ex.ID},
				Rationale: []string{
					"existing step already matches required inputs and " +
						"critical outputs",
				},
				Redundant: true,
			}
		}
		return coverageDetail{
			Status:    "covered_superset",
			CoveredBy: []string{ex.ID},
			Rationale: []string{
				"existing step already satisfies required inputs and " +
					"planner-critical outputs",
			},
			Overlap:   overlap,
			Redundant: true,
		}
	case len(overlap) > 0:
		return coverageDetail{
			Status:    "partial_overlap",
			CoveredBy: []string{ex.ID},
			Rationale: []string{
				"existing step overlaps with this candidate but does not " +
					"fully cover the same planning edge",
			},
			Missing: unionStrings(missingReq, missingCrit),
			Overlap: overlap,
		}
	default:
		return coverageDetail{Status: "recommended"}
	}
}

func coverageRank(status string) int {
	switch status {
	case "covered_exact":
		return 4
	case "covered_superset":
		return 3
	case "partial_overlap":
		return 2
	default:
		return 1
	}
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

func intersect(a, b []string) []string {
	set := util.Set[string]{}
	for _, v := range b {
		set.Add(v)
	}
	var res []string
	for _, v := range a {
		if set.Contains(v) {
			res = append(res, v)
		}
	}
	return uniqueStrings(res)
}

func diff(a, b []string) []string {
	set := util.Set[string]{}
	for _, v := range b {
		set.Add(v)
	}
	var res []string
	for _, v := range a {
		if !set.Contains(v) {
			res = append(res, v)
		}
	}
	return uniqueStrings(res)
}

func unionStrings(parts ...[]string) []string {
	var all []string
	for _, part := range parts {
		all = append(all, part...)
	}
	return uniqueStrings(all)
}

func uniqueStrings(in []string) []string {
	seen := util.Set[string]{}
	var res []string
	for _, v := range in {
		if v == "" {
			continue
		}
		if seen.Contains(v) {
			continue
		}
		seen.Add(v)
		res = append(res, v)
	}
	slices.Sort(res)
	return res
}
