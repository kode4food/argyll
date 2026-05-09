package policy_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestRequiredMatchStatus(t *testing.T) {
	cfg := &api.ScriptConfig{Language: api.ScriptLangJPath, Script: "$.kind"}
	matched := func(*api.ScriptConfig, any) (bool, error) {
		return true, nil
	}
	unmatched := func(*api.ScriptConfig, any) (bool, error) {
		return false, nil
	}
	failed := func(*api.ScriptConfig, any) (bool, error) {
		return false, errors.New("boom")
	}

	tests := []struct {
		name     string
		attr     *api.AttributeSpec
		values   []*api.AttributeValue
		provider policy.ProviderSummary
		evaluate func(*api.ScriptConfig, any) (bool, error)
		expected policy.MatchStatus
		err      bool
	}{
		{
			name:     "not gated",
			attr:     &api.AttributeSpec{Role: api.RoleRequired},
			expected: policy.MatchNotGated,
		},
		{
			name:     "unknown without values",
			attr:     requiredMatch(cfg, api.InputCollectFirst),
			provider: policy.ProviderSummary{},
			evaluate: matched,
			expected: policy.MatchUnknown,
		},
		{
			name:     "matched",
			attr:     requiredMatch(cfg, api.InputCollectFirst),
			values:   matchValues(map[string]any{"kind": "email"}),
			evaluate: matched,
			expected: policy.MatchMatched,
		},
		{
			name:     "unmatched",
			attr:     requiredMatch(cfg, api.InputCollectFirst),
			values:   matchValues(map[string]any{"kind": "postal"}),
			provider: terminalProvider(),
			evaluate: unmatched,
			expected: policy.MatchUnmatched,
		},
		{
			name: "last collect waits for terminal providers",
			attr: requiredMatch(cfg, api.InputCollectLast),
			values: matchValues(
				map[string]any{"kind": "email"},
				map[string]any{"kind": "postal"},
			),
			evaluate: matched,
			expected: policy.MatchUnknown,
		},
		{
			name: "last collect matches after terminal providers",
			attr: requiredMatch(cfg, api.InputCollectLast),
			values: matchValues(
				map[string]any{"kind": "email"},
				map[string]any{"kind": "postal"},
			),
			provider: terminalProvider(),
			evaluate: matched,
			expected: policy.MatchMatched,
		},
		{
			name:     "some collect evaluates individual candidates",
			attr:     requiredMatch(cfg, api.InputCollectSome),
			values:   matchValues("email", "postal"),
			evaluate: matchString("email"),
			expected: policy.MatchMatched,
		},
		{
			name:     "all collect fails on any unmatched candidate",
			attr:     requiredMatch(cfg, api.InputCollectAll),
			values:   matchValues("email", "postal"),
			evaluate: matchString("email"),
			expected: policy.MatchUnmatched,
		},
		{
			name:     "all collect matches after terminal providers",
			attr:     requiredMatch(cfg, api.InputCollectAll),
			values:   matchValues("email", "email"),
			provider: terminalProvider(),
			evaluate: matchString("email"),
			expected: policy.MatchMatched,
		},
		{
			name:     "none collect fails on matched candidate",
			attr:     requiredMatch(cfg, api.InputCollectNone),
			values:   matchValues("email"),
			evaluate: matchString("email"),
			expected: policy.MatchUnmatched,
		},
		{
			name:     "none collect matches after terminal providers",
			attr:     requiredMatch(cfg, api.InputCollectNone),
			values:   matchValues("postal"),
			provider: terminalProvider(),
			evaluate: matchString("email"),
			expected: policy.MatchMatched,
		},
		{
			name:     "eval error",
			attr:     requiredMatch(cfg, api.InputCollectFirst),
			values:   matchValues("email"),
			evaluate: failed,
			expected: policy.MatchUnknown,
			err:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := policy.RequiredMatchStatus(policy.RequiredMatchSpec{
				Attr:     tt.attr,
				Values:   tt.values,
				Provider: tt.provider,
				Evaluate: tt.evaluate,
			})
			assert.Equal(t, tt.err, err != nil)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestRequiredMatchStepStatus(t *testing.T) {
	cfg := &api.ScriptConfig{Language: api.ScriptLangJPath, Script: "$.kind"}
	step := &api.Step{
		Attributes: api.AttributeSpecs{
			"kind":   requiredMatch(cfg, api.InputCollectFirst),
			"region": requiredMatch(cfg, api.InputCollectFirst),
			"other":  {Role: api.RoleRequired},
		},
	}
	values := map[api.Name][]*api.AttributeValue{
		"kind":   matchValues("email"),
		"region": matchValues("us"),
	}
	valuesFor := func(name api.Name) []*api.AttributeValue {
		return values[name]
	}
	providers := func(api.Name) policy.ProviderSummary {
		return terminalProvider()
	}

	status, err := policy.RequiredMatchStepStatus(policy.RequiredMatchStep{
		Step:      step,
		Values:    valuesFor,
		Providers: providers,
		Evaluate: func(_ *api.ScriptConfig, value any) (bool, error) {
			return value == "email" || value == "us", nil
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, policy.MatchMatched, status)

	status, err = policy.RequiredMatchStepStatus(policy.RequiredMatchStep{
		Step:      step,
		Values:    valuesFor,
		Providers: providers,
		Evaluate: func(_ *api.ScriptConfig, value any) (bool, error) {
			return value == "email", nil
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, policy.MatchUnmatched, status)

	status, err = policy.RequiredMatchStepStatus(policy.RequiredMatchStep{
		Step: step,
		Values: func(name api.Name) []*api.AttributeValue {
			if name == "region" {
				return nil
			}
			return values[name]
		},
		Providers: func(name api.Name) policy.ProviderSummary {
			if name == "region" {
				return policy.ProviderSummary{}
			}
			return terminalProvider()
		},
		Evaluate: func(_ *api.ScriptConfig, value any) (bool, error) {
			return value == "email", nil
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, policy.MatchUnknown, status)
}

func matchString(want string) func(*api.ScriptConfig, any) (bool, error) {
	return func(_ *api.ScriptConfig, value any) (bool, error) {
		return value == want, nil
	}
}

func matchValues(values ...any) []*api.AttributeValue {
	res := make([]*api.AttributeValue, 0, len(values))
	for _, value := range values {
		res = append(res, &api.AttributeValue{Value: value})
	}
	return res
}

func terminalProvider() policy.ProviderSummary {
	return policy.ProviderSummary{
		Terminal:     true,
		AllSucceeded: true,
	}
}

func requiredMatch(
	cfg *api.ScriptConfig, collect api.InputCollect,
) *api.AttributeSpec {
	return &api.AttributeSpec{
		Role: api.RoleRequired,
		Required: &api.RequiredConfig{
			Collect: collect,
			Match:   cfg,
		},
	}
}
