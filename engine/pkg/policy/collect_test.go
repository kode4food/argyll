package policy_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

func TestRequiresAllProviders(t *testing.T) {
	assert.True(t, policy.RequiresAllProviders(api.InputCollectAll))
	assert.False(t, policy.RequiresAllProviders(api.InputCollectFirst))
	assert.False(t, policy.RequiresAllProviders(api.InputCollectLast))
	assert.False(t, policy.RequiresAllProviders(api.InputCollectSome))
	assert.False(t, policy.RequiresAllProviders(api.InputCollectNone))
}

func TestTimeoutCanUseValues(t *testing.T) {
	assert.True(t, policy.TimeoutCanUseValues(api.InputCollectLast))
	assert.True(t, policy.TimeoutCanUseValues(api.InputCollectSome))
	assert.True(t, policy.TimeoutCanUseValues(api.InputCollectNone))
	assert.False(t, policy.TimeoutCanUseValues(api.InputCollectFirst))
	assert.False(t, policy.TimeoutCanUseValues(api.InputCollectAll))
}

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

	assert.True(t, policy.InitSatisfiesRequired(&policy.RequiredInputState{
		Attr:    &api.AttributeSpec{Role: api.RoleRequired},
		HasInit: true,
		Match:   matcher,
	}))
	assert.False(t, policy.InitSatisfiesRequired(&policy.RequiredInputState{
		Attr:  &api.AttributeSpec{Role: api.RoleRequired},
		Match: matcher,
	}))
	assert.True(t, policy.InitSatisfiesRequired(&policy.RequiredInputState{
		Attr:    requiredMatch(cfg, api.InputCollectFirst),
		HasInit: true,
		Values:  values,
		Match:   matcher,
	}))
	assert.False(t, policy.InitSatisfiesRequired(&policy.RequiredInputState{
		Attr:    requiredMatch(cfg, api.InputCollectFirst),
		HasInit: true,
		Values:  []*api.AttributeValue{{Value: "bad"}},
		Match:   matcher,
	}))
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

func TestInputSatisfiable(t *testing.T) {
	yes := func() bool { return true }
	no := func() bool { return false }
	modes := []api.InputCollect{
		api.InputCollectFirst, api.InputCollectLast, api.InputCollectSome,
	}

	for _, collect := range modes {
		assert.True(t, policy.InputSatisfiable(collect, policy.InputOutlook{
			HasValue: true, Providable: no,
		}))
		assert.True(t, policy.InputSatisfiable(collect, policy.InputOutlook{
			Providable: yes,
		}))
		assert.False(t, policy.InputSatisfiable(collect, policy.InputOutlook{
			Providable: no,
		}))
	}

	all := api.InputCollectAll
	assert.True(t, policy.InputSatisfiable(all, policy.InputOutlook{
		HasValue: true, AllCollectable: true, Providable: no,
	}))
	assert.True(t, policy.InputSatisfiable(all, policy.InputOutlook{
		AllCollectable: true, Providable: yes,
	}))
	assert.False(t, policy.InputSatisfiable(all, policy.InputOutlook{
		HasValue: true, Providable: yes,
	}))
	assert.False(t, policy.InputSatisfiable(all, policy.InputOutlook{
		AllCollectable: true, Providable: no,
	}))

	none := api.InputCollectNone
	assert.True(t, policy.InputSatisfiable(none, policy.InputOutlook{
		Providable: no,
	}))
	assert.False(t, policy.InputSatisfiable(none, policy.InputOutlook{
		HasValue: true, Providable: yes,
	}))
}

func TestFulfilledAt(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	done := base.Add(time.Hour)
	values := []*api.AttributeValue{
		{Value: "a", SetAt: base.Add(time.Minute)},
		{Value: "b", SetAt: base.Add(3 * time.Minute)},
		{Value: "c", SetAt: base.Add(2 * time.Minute)},
	}

	assert.Equal(t, base.Add(time.Minute),
		policy.FulfilledAt(api.InputCollectFirst, values, done),
	)
	assert.Equal(t, base.Add(2*time.Minute),
		policy.FulfilledAt(api.InputCollectLast, values, done),
	)
	assert.Equal(t, base.Add(3*time.Minute),
		policy.FulfilledAt(api.InputCollectSome, values, done),
	)
	assert.Equal(t, base.Add(3*time.Minute),
		policy.FulfilledAt(api.InputCollectAll, values, done),
	)
	assert.Equal(t, done, policy.FulfilledAt(api.InputCollectNone, nil, done))
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
