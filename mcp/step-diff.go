package mcp

type stepDiff struct {
	Current  map[string]any `json:"current,omitempty"`
	Proposed map[string]any `json:"proposed,omitempty"`
	Diff     map[string]any `json:"diff,omitempty"`
	ID       string         `json:"id"`
	Action   string         `json:"action"`
	Reason   string         `json:"reason"`
	Fields   []string       `json:"fields,omitempty"`
}

func (s *Server) diffProposedSteps(args diffProposedStepsArgs) (any, error) {
	steps, err := proposedSteps(args.Steps)
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
	steps, err := proposedSteps(args.Steps)
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
	var appliedProposed []map[string]any
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
			appliedProposed = append(appliedProposed, d.Proposed)
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
			appliedProposed = append(appliedProposed, d.Proposed)
		default:
			skipped = append(skipped, map[string]any{
				"id":     d.ID,
				"action": d.Action,
				"reason": d.Reason,
			})
		}
	}

	verification, err := s.verifyAppliedSteps(appliedProposed)
	if err != nil {
		return nil, err
	}

	return toolResult(map[string]any{
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
	}, nil)
}

func (s *Server) verifyAppliedSteps(
	proposed []map[string]any,
) (semanticVerification, error) {
	var semantic []map[string]any
	for _, step := range proposed {
		if len(semanticConfigs(step)) != 0 {
			semantic = append(semantic, step)
		}
	}
	if len(semantic) == 0 {
		return semanticVerification{Status: "skipped"}, nil
	}
	current, err := s.currentSteps()
	if err != nil {
		return semanticVerification{}, err
	}
	return verifyAppliedSemantics(semantic, current), nil
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

func proposedSteps(steps *[]map[string]any) ([]map[string]any, error) {
	if steps != nil && len(*steps) != 0 {
		return *steps, nil
	}
	return nil, errInvalidParams("steps is required")
}
