package mcp

import (
	"encoding/json"
	"slices"
	"strings"
)

func buildStepDiffs(
	current map[string]map[string]any, steps []map[string]any,
) ([]stepDiff, map[string]int) {
	diffs := make([]stepDiff, 0, len(steps))
	counts := map[string]int{
		"create": 0,
		"update": 0,
		"skip":   0,
	}
	for _, proposed := range steps {
		id := stringValue(proposed["id"])
		if id == "" {
			continue
		}
		curr, exists := current[id]
		switch {
		case !exists:
			diffs = append(diffs, stepDiff{
				ID:       id,
				Action:   "create",
				Reason:   "step does not exist in the current catalog",
				Proposed: proposed,
			})
			counts["create"]++
		default:
			fields, delta := diffStepFields(curr, proposed)
			if len(fields) == 0 {
				diffs = append(diffs, stepDiff{
					ID:     id,
					Action: "skip",
					Reason: "proposed step is identical to the current " +
						"registration",
					Current:  curr,
					Proposed: proposed,
				})
				counts["skip"]++
				continue
			}
			diffs = append(diffs, stepDiff{
				ID:       id,
				Action:   "update",
				Reason:   "proposed step differs from the current registration",
				Fields:   fields,
				Current:  curr,
				Proposed: proposed,
				Diff:     delta,
			})
			counts["update"]++
		}
	}

	slices.SortFunc(diffs, func(a, b stepDiff) int {
		if cmp := strings.Compare(a.Action, b.Action); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.ID, b.ID)
	})
	return diffs, counts
}

func diffStepFields(curr, proposed map[string]any) ([]string, map[string]any) {
	keys := []string{
		"name",
		"type",
		"labels",
		"attributes",
		"http",
		"flow",
		"script",
		"predicate",
		"work_config",
		"memoizable",
	}
	var fields []string
	delta := map[string]any{}
	for _, key := range keys {
		if sameJSON(curr[key], proposed[key]) {
			continue
		}
		fields = append(fields, key)
		delta[key] = map[string]any{
			"current":  curr[key],
			"proposed": proposed[key],
		}
	}
	return fields, delta
}

func sameJSON(a, b any) bool {
	aj, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bj, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aj) == string(bj)
}
