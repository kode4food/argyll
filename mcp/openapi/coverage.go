package openapi

import "slices"

type coverageDetail struct {
	Status    string
	CoveredBy []string
	Rationale []string
	Missing   []string
	Overlap   []string
	Redundant bool
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
