package policy

import "github.com/kode4food/argyll/engine/pkg/api"

type (
	// RequiredMatchStep carries layer facts needed to evaluate required match
	// gates for a step
	RequiredMatchStep struct {
		Step      *api.Step
		Values    func(api.Name) []*api.AttributeValue
		Providers func(api.Name) ProviderSummary
		Evaluate  MatchEvaluator
	}

	// RequiredMatchSpec carries layer facts needed to evaluate required match
	// gates for an attribute specification
	RequiredMatchSpec struct {
		Attr     *api.AttributeSpec
		Values   []*api.AttributeValue
		Provider ProviderSummary
		Evaluate MatchEvaluator
	}

	// MatchEvaluator evaluates a required match script against one candidate
	// attribute value
	MatchEvaluator func(*api.ScriptConfig, any) (bool, error)

	// MatchStatus classifies whether a step's required match gates are open,
	// closed, still waiting for more candidate values, or absent
	MatchStatus int
)

const (
	MatchNotGated MatchStatus = iota
	MatchUnknown
	MatchMatched
	MatchUnmatched
)

const (
	RequiredMatchSkipReason = "required match did not match"
	MatchInputName          = api.Name("value")
)

// RequiredInputHasMatch reports whether a required input has a non-empty match
// predicate that should gate demand and candidate collection
func RequiredInputHasMatch(attr *api.AttributeSpec) bool {
	return attr.IsRequired() &&
		attr.Required != nil &&
		attr.Required.Match != nil &&
		attr.Required.Match.Script != ""
}

// RequiredMatchStepStatus combines all required match gates on a step. Any
// unmatched gate closes the step, any unknown gate delays normal demand, and a
// matched result opens demand for the remaining required inputs
func RequiredMatchStepStatus(facts RequiredMatchStep) (MatchStatus, error) {
	status := MatchNotGated
	for name, attr := range facts.Step.Attributes {
		if !RequiredInputHasMatch(attr) {
			continue
		}
		decision, err := RequiredMatchStatus(RequiredMatchSpec{
			Attr:     attr,
			Values:   facts.Values(name),
			Provider: facts.Providers(name),
			Evaluate: facts.Evaluate,
		})
		if err != nil {
			return MatchUnknown, err
		}
		if decision == MatchUnmatched {
			return MatchUnmatched, nil
		}
		if decision == MatchUnknown {
			status = MatchUnknown
		}
		if decision == MatchMatched && status == MatchNotGated {
			status = MatchMatched
		}
	}
	return status, nil
}

// RequiredMatchStatus evaluates one required input's match predicate against
// each candidate value and applies the input's collect policy to those results
func RequiredMatchStatus(facts RequiredMatchSpec) (MatchStatus, error) {
	if !RequiredInputHasMatch(facts.Attr) {
		return MatchNotGated, nil
	}

	matched, unmatched, err := MatchCandidateValues(
		facts.Attr, facts.Values, facts.Evaluate,
	)
	if err != nil {
		return MatchUnknown, err
	}

	switch facts.Attr.Collect() {
	case api.InputCollectNone:
		if len(matched) > 0 {
			return MatchUnmatched, nil
		}
		if facts.Provider.Terminal {
			return MatchMatched, nil
		}
		return MatchUnknown, nil
	case api.InputCollectLast:
		if len(matched) > 0 && facts.Provider.Terminal {
			return MatchMatched, nil
		}
		if facts.Provider.Terminal {
			return MatchUnmatched, nil
		}
		return MatchUnknown, nil
	case api.InputCollectAll:
		if unmatched > 0 {
			return MatchUnmatched, nil
		}
		if len(matched) > 0 && facts.Provider.Terminal &&
			facts.Provider.AllSucceeded {
			return MatchMatched, nil
		}
		if facts.Provider.Terminal {
			return MatchUnmatched, nil
		}
		return MatchUnknown, nil
	default:
		if len(matched) > 0 {
			return MatchMatched, nil
		}
		if facts.Provider.Terminal {
			return MatchUnmatched, nil
		}
		return MatchUnknown, nil
	}
}

// MatchAllowsNormalDemand reports whether a gate status allows non-gate
// required inputs to demand their producers
func MatchAllowsNormalDemand(status MatchStatus) bool {
	return status == MatchNotGated || status == MatchMatched
}

// MatchAllowsStepSkip reports whether a gate status should prune the step
// instead of starting work
func MatchAllowsStepSkip(status MatchStatus) bool {
	return status == MatchUnmatched
}

// StepPrunedByRequiredMatch reports whether a skipped step was pruned by a
// required match gate rather than by a step predicate
func StepPrunedByRequiredMatch(status api.StepStatus, reason string) bool {
	return status == api.StepSkipped && reason == RequiredMatchSkipReason
}

// MatchStep builds the narrow synthetic step used to compile and evaluate a
// required match predicate with only the candidate value in scope
func MatchStep(cfg *api.ScriptConfig) *api.Step {
	return &api.Step{
		ID:   "required-match",
		Name: "Required Match",
		Type: api.StepTypeScript,
		Attributes: api.AttributeSpecs{
			MatchInputName: {
				Role: api.RoleRequired,
				Type: api.TypeAny,
			},
		},
		Script: cfg,
	}
}

// MatchCandidateValues returns only candidate values that satisfy a required
// match predicate, plus the count of candidates that were evaluated and failed
func MatchCandidateValues(
	attr *api.AttributeSpec, values []*api.AttributeValue,
	evaluate MatchEvaluator,
) ([]*api.AttributeValue, int, error) {
	if !RequiredInputHasMatch(attr) {
		return values, 0, nil
	}

	matched := make([]*api.AttributeValue, 0, len(values))
	unmatched := 0
	for _, val := range values {
		ok, err := evaluate(attr.Required.Match, val.Value)
		if err != nil {
			return nil, 0, err
		}
		if ok {
			matched = append(matched, val)
			continue
		}
		unmatched++
	}
	return matched, unmatched, nil
}
