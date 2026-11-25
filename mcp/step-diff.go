package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type stepDiff struct {
	Current  *api.Step      `json:"current,omitempty"`
	Proposed *api.Step      `json:"proposed,omitempty"`
	Diff     map[string]any `json:"diff,omitempty"`
	ID       api.StepID     `json:"id"`
	Action   string         `json:"action"`
	Reason   string         `json:"reason"`
	Fields   []string       `json:"fields,omitempty"`
}

func (s *Server) diffProposedSteps(
	ctx context.Context, args diffProposedStepsArgs,
) (any, error) {
	steps, err := proposedSteps(args.Steps)
	if err != nil {
		return nil, err
	}
	current, err := s.currentSteps(ctx)
	if err != nil {
		return nil, err
	}
	diffs, counts, err := buildStepDiffs(current, steps)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"summary": map[string]any{
			"create": counts["create"],
			"update": counts["update"],
			"skip":   counts["skip"],
			"total":  len(diffs),
		},
		"steps":      diffs,
		"validation": "Comparison only; registration validates scripts and catalog constraints",
	}, nil
}

func (s *Server) applyProposedSteps(
	ctx context.Context, args applyProposedStepsArgs,
) (any, error) {
	steps, err := proposedSteps(args.Steps)
	if err != nil {
		return nil, err
	}
	current, err := s.currentSteps(ctx)
	if err != nil {
		return nil, err
	}
	diffs, counts, err := buildStepDiffs(current, steps)
	if err != nil {
		return nil, err
	}

	var applied []map[string]any
	var skipped []map[string]any
	var appliedProposed []*api.Step
	for i, d := range diffs {
		switch d.Action {
		case "create":
			res, err := s.httpPost(ctx, "/engine/steps", d.Proposed)
			if err != nil {
				return applyFailure(applied, skipped, diffs[i:], err), err
			}
			applied = append(applied, map[string]any{
				"id":       d.ID,
				"action":   "create",
				"response": res,
			})
			appliedProposed = append(appliedProposed, d.Proposed)
		case "update":
			if args.SkipUpdates {
				skipped = append(skipped, map[string]any{
					"id":     d.ID,
					"action": "update",
					"reason": "updates disabled for this apply call",
					"fields": d.Fields,
				})
				continue
			}
			res, err := s.httpPut(ctx, "/engine/steps/"+url.PathEscape(string(d.ID)), d.Proposed)
			if err != nil {
				return applyFailure(applied, skipped, diffs[i:], err), err
			}
			applied = append(applied, map[string]any{
				"id":       d.ID,
				"action":   "update",
				"fields":   d.Fields,
				"response": res,
			})
			appliedProposed = append(appliedProposed, d.Proposed)
		default:
			skipped = append(skipped, map[string]any{
				"id":     d.ID,
				"action": d.Action,
				"reason": d.Reason,
			})
		}
	}

	verification, err := s.verifyAppliedSteps(ctx, appliedProposed)
	if err != nil {
		return applyFailure(applied, skipped, nil, err), err
	}

	return map[string]any{
		"summary": map[string]any{
			"requested":           len(steps),
			"create":              counts["create"],
			"update":              counts["update"],
			"skip":                counts["skip"],
			"applied":             len(applied),
			"skipped":             len(skipped),
			"verification_status": verification.Status,
			"verification_issues": len(verification.Issues),
		},
		"applied_steps": applied,
		"skipped_steps": skipped,
		"verification":  verification,
	}, nil
}

func (s *Server) verifyAppliedSteps(
	ctx context.Context, proposed []*api.Step,
) (semanticVerification, error) {
	var semantic []map[string]any
	for _, st := range proposed {
		fields, err := stepFields(st)
		if err != nil {
			return semanticVerification{}, err
		}
		if len(semanticConfigs(fields)) != 0 {
			semantic = append(semantic, fields)
		}
	}
	if len(semantic) == 0 {
		return semanticVerification{Status: "skipped"}, nil
	}
	current, err := s.currentSteps(ctx)
	if err != nil {
		return semanticVerification{}, err
	}
	return verifyAppliedSemantics(semantic, current), nil
}

func decodeStep(fields map[string]any) (*api.Step, error) {
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	var st api.Step
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Server) currentSteps(ctx context.Context) (map[api.StepID]map[string]any, error) {
	payload, err := s.httpGet(ctx, "/engine/steps")
	if err != nil {
		return nil, err
	}
	root, ok := asMap(payload)
	if !ok {
		return nil, errInvalidParams("engine step payload was not an object")
	}

	current := map[api.StepID]map[string]any{}
	if items, ok := root["steps"].([]any); ok {
		for _, item := range items {
			st, ok := asMap(item)
			if !ok {
				continue
			}
			id := api.StepID(stringValue(st["id"]))
			if id == "" {
				continue
			}
			current[id] = st
		}
	}
	return current, nil
}

func proposedSteps(steps []map[string]any) ([]map[string]any, error) {
	if len(steps) == 0 {
		return nil, errInvalidParams("steps is required")
	}
	seen := util.Set[api.StepID]{}
	for i, st := range steps {
		id := api.StepID(stringValue(st["id"]))
		if strings.TrimSpace(string(id)) == "" {
			return nil, fmt.Errorf("%w: steps[%d].id", ErrInvalidParams, i)
		}
		if api.SanitizeID(id) != id {
			return nil, fmt.Errorf("%w: %s", api.ErrStepIDInvalid, id)
		}
		if seen.Contains(id) {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateStepID, id)
		}
		seen.Add(id)
	}
	return steps, nil
}

func applyFailure(
	applied, skipped []map[string]any, remaining []stepDiff, err error,
) map[string]any {
	res := map[string]any{
		"error":               err.Error(),
		"applied_steps":       applied,
		"skipped_steps":       skipped,
		"verification_status": "unknown",
		"note": "No rollback; a failed request may have committed. " +
			"Read the catalog before retrying",
	}
	if len(remaining) != 0 {
		res["failed_step"] = remaining[0]
		res["pending_steps"] = remaining[1:]
	} else {
		res["phase"] = "verification"
	}
	return res
}
