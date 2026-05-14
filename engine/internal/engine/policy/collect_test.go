package policy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestInitPolicy(t *testing.T) {
	assert.True(t, policy.InitSatisfiesInput(api.InputCollectFirst, true))
	assert.False(t, policy.InitSatisfiesInput(api.InputCollectLast, true))
	assert.False(t, policy.InitSatisfiesInput(api.InputCollectFirst, false))

	assert.True(t, policy.InitBlocksInput(api.InputCollectNone, true))
	assert.False(t, policy.InitBlocksInput(api.InputCollectFirst, true))
	assert.False(t, policy.InitBlocksInput(api.InputCollectNone, false))

	assert.True(t, policy.InitProviderComplete(true, false))
	assert.False(t, policy.InitProviderComplete(true, true))
	assert.False(t, policy.InitProviderComplete(false, false))
}

func TestInitSatisfiesRequired(t *testing.T) {
	cfg := &api.ScriptConfig{Language: "test", Script: "match"}
	matcher := func(_ *api.ScriptConfig, value any) (bool, error) {
		return value == "ok", nil
	}
	values := []*api.AttributeValue{{Value: "ok"}}

	assert.True(t, policy.InitSatisfiesRequired(
		&api.AttributeSpec{Role: api.RoleRequired},
		true, false, nil, matcher,
	))
	assert.False(t, policy.InitSatisfiesRequired(
		&api.AttributeSpec{Role: api.RoleRequired},
		false, false, nil, matcher,
	))
	assert.True(t, policy.InitSatisfiesRequired(
		requiredMatch(cfg, api.InputCollectFirst),
		true, false, values, matcher,
	))
	assert.False(t, policy.InitSatisfiesRequired(
		requiredMatch(cfg, api.InputCollectFirst),
		true, false, []*api.AttributeValue{{Value: "bad"}}, matcher,
	))
}

func TestInitBlocksRuntime(t *testing.T) {
	assert.True(t, policy.InitBlocksRuntime(
		&api.AttributeSpec{
			Role: api.RoleRequired,
			Required: &api.RequiredConfig{
				Collect: api.InputCollectNone,
			},
		},
		true,
	))
	assert.False(t, policy.InitBlocksRuntime(
		&api.AttributeSpec{
			Role: api.RoleConst,
			Const: &api.ConstConfig{
				Value: "fixed",
			},
		},
		true,
	))
}

func TestRequiredInputMissing(t *testing.T) {
	assert.False(t,
		policy.RequiredInputMissing(api.InputCollectFirst, true, false),
	)
	assert.False(t,
		policy.RequiredInputMissing(api.InputCollectNone, false, false),
	)
	assert.False(t,
		policy.RequiredInputMissing(api.InputCollectFirst, false, true),
	)
	assert.True(t,
		policy.RequiredInputMissing(api.InputCollectFirst, false, false),
	)
}

func TestProviderOutputNeeded(t *testing.T) {
	assert.False(t,
		policy.ProviderOutputNeeded(api.InputCollectFirst, true, true),
	)
	assert.True(t,
		policy.ProviderOutputNeeded(api.InputCollectFirst, false, true),
	)
	assert.True(t,
		policy.ProviderOutputNeeded(api.InputCollectLast, true, true),
	)
	assert.True(t,
		policy.ProviderOutputNeeded(api.InputCollectSome, true, true),
	)
	assert.True(t,
		policy.ProviderOutputNeeded(api.InputCollectAll, true, true),
	)
	assert.False(t,
		policy.ProviderOutputNeeded(api.InputCollectAll, true, false),
	)
}

func TestRequiredInputsAvailable(t *testing.T) {
	st := &api.Step{
		Attributes: api.AttributeSpecs{
			"input": {Role: api.RoleRequired},
			"out":   {Role: api.RoleOutput},
		},
	}

	assert.True(t, policy.RequiredInputsAvailable(st,
		func(name api.Name) bool {
			return name == "input"
		},
	))
	assert.False(t, policy.RequiredInputsAvailable(st,
		func(api.Name) bool {
			return false
		},
	))
}

func TestStepOutputsSatisfied(t *testing.T) {
	st := &api.Step{
		Attributes: api.AttributeSpecs{
			"in":  {Role: api.RoleRequired},
			"out": {Role: api.RoleOutput},
		},
	}

	assert.True(t, policy.StepOutputsSatisfied(st,
		func(name api.Name) bool {
			return name == "out"
		},
	))
	assert.False(t, policy.StepOutputsSatisfied(st,
		func(api.Name) bool {
			return false
		},
	))
	assert.False(t, policy.StepOutputsSatisfied(&api.Step{},
		func(api.Name) bool {
			return true
		},
	))
}

func TestStepInputGuaranteed(t *testing.T) {
	assert.True(t, policy.StepInputGuaranteed(&api.AttributeSpec{
		Role: api.RoleRequired,
	}))
	assert.True(t, policy.StepInputGuaranteed(&api.AttributeSpec{
		Role: api.RoleConst,
	}))
	assert.True(t, policy.StepInputGuaranteed(&api.AttributeSpec{
		Role: api.RoleOptional,
		Optional: &api.OptionalConfig{
			Default: "true",
		},
	}))
	assert.False(t, policy.StepInputGuaranteed(&api.AttributeSpec{
		Role: api.RoleOptional,
	}))
}

func TestInputFulfilled(t *testing.T) {
	terminal := policy.ProviderSummary{Terminal: true, AllSucceeded: true}
	incomplete := policy.ProviderSummary{Terminal: false, AllSucceeded: true}
	partial := policy.ProviderSummary{Terminal: true, AllSucceeded: false}

	assert.True(t, policy.InputFulfilled(api.InputCollectNone, 0, terminal))
	assert.False(t, policy.InputFulfilled(api.InputCollectNone, 1, terminal))
	assert.False(t, policy.InputFulfilled(api.InputCollectNone, 0, incomplete))

	assert.True(t, policy.InputFulfilled(api.InputCollectFirst, 1, incomplete))
	assert.False(t, policy.InputFulfilled(api.InputCollectFirst, 0, terminal))

	assert.True(t, policy.InputFulfilled(api.InputCollectLast, 1, terminal))
	assert.False(t, policy.InputFulfilled(api.InputCollectLast, 1, incomplete))

	assert.True(t, policy.InputFulfilled(api.InputCollectSome, 1, terminal))
	assert.False(t, policy.InputFulfilled(api.InputCollectSome, 1, incomplete))

	assert.True(t, policy.InputFulfilled(api.InputCollectAll, 1, terminal))
	assert.False(t, policy.InputFulfilled(api.InputCollectAll, 1, incomplete))
	assert.False(t, policy.InputFulfilled(api.InputCollectAll, 1, partial))
}

func TestResolveInputValue(t *testing.T) {
	values := []*api.AttributeValue{
		{Value: "a"},
		{Value: "b"},
	}

	val, ok := policy.ResolveInputValue(api.InputCollectFirst, values)
	assert.True(t, ok)
	assert.Equal(t, "a", val)

	val, ok = policy.ResolveInputValue(api.InputCollectLast, values)
	assert.True(t, ok)
	assert.Equal(t, "b", val)

	val, ok = policy.ResolveInputValue(api.InputCollectSome, values)
	assert.True(t, ok)
	assert.Equal(t, []any{"a", "b"}, val)

	val, ok = policy.ResolveInputValue(api.InputCollectAll, values)
	assert.True(t, ok)
	assert.Equal(t, []any{"a", "b"}, val)

	_, ok = policy.ResolveInputValue(api.InputCollectNone, values)
	assert.False(t, ok)

	_, ok = policy.ResolveInputValue(api.InputCollectFirst, nil)
	assert.False(t, ok)
}
