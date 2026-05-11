package mcp

type stepDiff struct {
	ID       string         `json:"id"`
	Action   string         `json:"action"`
	Reason   string         `json:"reason"`
	Fields   []string       `json:"fields,omitempty"`
	Current  map[string]any `json:"current,omitempty"`
	Proposed map[string]any `json:"proposed,omitempty"`
	Diff     map[string]any `json:"diff,omitempty"`
}

func (s *Server) diffProposedSteps(args diffProposedStepsArgs) (any, error) {
	steps, err := proposedSteps(args.Steps, args.Proposal)
	if err != nil {
		return nil, err
	}
	current, err := s.currentSteps()
	if err != nil {
		return nil, err
	}
	diffs, counts := buildStepDiffs(current, steps)

	return toolResult(map[string]any{
		"summary": map[string]any{
			"create": counts["create"],
			"update": counts["update"],
			"skip":   counts["skip"],
			"total":  len(diffs),
		},
		"steps": diffs,
	}, nil)
}

func (s *Server) applyProposedSteps(args applyProposedStepsArgs) (any, error) {
	steps, err := proposedSteps(args.Steps, args.Proposal)
	if err != nil {
		return nil, err
	}
	current, err := s.currentSteps()
	if err != nil {
		return nil, err
	}
	diffs, counts := buildStepDiffs(current, steps)

	applyUpdates := true
	if args.ApplyUpdates != nil {
		applyUpdates = *args.ApplyUpdates
	}

	var applied []map[string]any
	var skipped []map[string]any
	for _, d := range diffs {
		switch d.Action {
		case "create":
			res, err := s.httpPost("/engine/step", d.Proposed)
			if err != nil {
				return nil, err
			}
			applied = append(applied, map[string]any{
				"id":       d.ID,
				"action":   "create",
				"response": res,
			})
		case "update":
			if !applyUpdates {
				skipped = append(skipped, map[string]any{
					"id":     d.ID,
					"action": "update",
					"reason": "updates disabled for this apply call",
					"fields": d.Fields,
				})
				continue
			}
			res, err := s.httpPut("/engine/step/"+d.ID, d.Proposed)
			if err != nil {
				return nil, err
			}
			applied = append(applied, map[string]any{
				"id":       d.ID,
				"action":   "update",
				"fields":   d.Fields,
				"response": res,
			})
		default:
			skipped = append(skipped, map[string]any{
				"id":     d.ID,
				"action": d.Action,
				"reason": d.Reason,
			})
		}
	}

	return toolResult(map[string]any{
		"summary": map[string]any{
			"requested": len(steps),
			"create":    counts["create"],
			"update":    counts["update"],
			"skip":      counts["skip"],
			"applied":   len(applied),
			"skipped":   len(skipped),
		},
		"applied_steps": applied,
		"skipped_steps": skipped,
	}, nil)
}

func proposedSteps(
	steps *[]map[string]any, proposal *map[string]any,
) ([]map[string]any, error) {
	if steps != nil && len(*steps) != 0 {
		return *steps, nil
	}
	if proposal == nil {
		return nil, errInvalidParams("steps or proposal is required")
	}

	raw, ok := (*proposal)["proposed_registrations"]
	if !ok {
		return nil, errInvalidParams(
			"proposal.proposed_registrations is required",
		)
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, errInvalidParams(
			"proposal.proposed_registrations must be an array",
		)
	}

	res := make([]map[string]any, 0, len(items))
	for _, item := range items {
		st, ok := asMap(item)
		if !ok {
			continue
		}
		step, ok := asMap(st["step"])
		if !ok || len(step) == 0 {
			continue
		}
		res = append(res, step)
	}
	if len(res) == 0 {
		return nil, errInvalidParams(
			"proposal.proposed_registrations did not contain step drafts",
		)
	}
	return res, nil
}

func (s *Server) currentSteps() (map[string]map[string]any, error) {
	payload, err := s.httpGet("/engine/step")
	if err != nil {
		return nil, err
	}
	root, ok := asMap(payload)
	if !ok {
		return nil, errInvalidParams("engine step payload was not an object")
	}

	current := map[string]map[string]any{}
	if items, ok := root["steps"].([]any); ok {
		for _, item := range items {
			st, ok := asMap(item)
			if !ok {
				continue
			}
			id := stringValue(st["id"])
			if id == "" {
				continue
			}
			current[id] = st
		}
	}
	return current, nil
}
