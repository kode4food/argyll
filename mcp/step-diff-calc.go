package mcp

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	stepFieldDiff struct {
		current  map[string]any
		proposed map[string]any
	}

	jsonValues struct {
		current  any
		proposed any
	}
)

func (d stepDiff) compare(other stepDiff) int {
	if c := strings.Compare(d.Action, other.Action); c != 0 {
		return c
	}
	return cmp.Compare(d.ID, other.ID)
}

func buildStepDiffs(
	current map[api.StepID]map[string]any, steps []map[string]any,
) ([]stepDiff, map[string]int, error) {
	diffs := make([]stepDiff, 0, len(steps))
	counts := map[string]int{
		"create": 0,
		"update": 0,
		"skip":   0,
	}
	for _, fields := range steps {
		proposed, err := decodeStep(fields)
		if err != nil {
			return nil, nil, err
		}
		id := proposed.ID
		raw, exists := current[id]
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
			curr, err := decodeStep(raw)
			if err != nil {
				return nil, nil, err
			}
			comparison, err := normalizedStepFields(curr, proposed)
			if err != nil {
				return nil, nil, err
			}
			fields, delta := diffStepFields(comparison)
			if len(fields) == 0 {
				diffs = append(diffs, stepDiff{
					ID:       id,
					Action:   "skip",
					Reason:   "proposed step is identical to the current registration",
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

	slices.SortFunc(diffs, stepDiff.compare)
	return diffs, counts, nil
}

func normalizedStepFields(
	current, proposed *api.Step,
) (stepFieldDiff, error) {
	var res stepFieldDiff
	for _, pair := range []struct {
		input  *api.Step
		output *map[string]any
	}{
		{
			input:  current,
			output: &res.current,
		},
		{
			input:  proposed,
			output: &res.proposed,
		},
	} {
		normal := pair.input.Copy()
		normal.Tags = normal.Tags.Normalize()
		fields, err := stepFields(normal)
		if err != nil {
			return res, err
		}
		*pair.output = fields
	}
	return res, nil
}

func stepFields(st *api.Step) (map[string]any, error) {
	raw, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func diffStepFields(steps stepFieldDiff) ([]string, map[string]any) {
	keys := []string{
		"name",
		"description",
		"type",
		"tags",
		"attributes",
		"http",
		"flow",
		"script",
		"agent",
		"predicate",
		"work_config",
		"handling",
	}
	var fields []string
	delta := map[string]any{}
	for _, key := range keys {
		if sameJSON(jsonValues{
			current:  steps.current[key],
			proposed: steps.proposed[key],
		}) {
			continue
		}
		fields = append(fields, key)
		delta[key] = map[string]any{
			"current":  steps.current[key],
			"proposed": steps.proposed[key],
		}
	}
	return fields, delta
}

func sameJSON(values jsonValues) bool {
	aj, err := json.Marshal(values.current)
	if err != nil {
		return false
	}
	bj, err := json.Marshal(values.proposed)
	if err != nil {
		return false
	}
	return string(aj) == string(bj)
}
