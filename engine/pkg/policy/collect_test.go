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

func TestCollectWindow(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		timeout int64
		arrival []time.Duration
		count   int
		end     time.Duration
	}{
		{name: "empty", timeout: 1000, end: time.Second},
		{
			name:    "already available",
			timeout: 1000,
			arrival: []time.Duration{-time.Second, 0},
			count:   2,
			end:     time.Second,
		},
		{
			name:    "each arrival extends",
			timeout: 1000,
			arrival: []time.Duration{
				500 * time.Millisecond, 1200 * time.Millisecond,
			},
			count: 2,
			end:   2200 * time.Millisecond,
		},
		{
			name:    "a gap closes permanently",
			timeout: 1000,
			arrival: []time.Duration{
				500 * time.Millisecond, 1600 * time.Millisecond,
				1700 * time.Millisecond,
			},
			count: 1,
			end:   1500 * time.Millisecond,
		},
		{
			name:    "expiry boundary",
			timeout: 1000,
			arrival: []time.Duration{time.Second},
			end:     time.Second,
		},
		{
			name:    "zero does not wait",
			arrival: []time.Duration{0, time.Nanosecond},
			count:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var values []*api.AttributeValue
			for _, at := range tt.arrival {
				values = append(values, &api.AttributeValue{
					SetAt: base.Add(at),
				})
			}
			got, at := policy.CollectWindow(base, tt.timeout, values)
			assert.Len(t, got, tt.count)
			assert.Equal(t, base.Add(tt.end), at)
		})
	}
}

func TestInitBlocksRuntime(t *testing.T) {
	required := func(collect api.InputCollect) *api.AttributeSpec {
		return &api.AttributeSpec{
			Role:     api.RoleRequired,
			Required: &api.RequiredConfig{Collect: collect},
		}
	}
	assert.False(t,
		policy.InitBlocksRuntime(required(api.InputCollectFirst), true))
	assert.False(t,
		policy.InitBlocksRuntime(required(api.InputCollectNone), false))

	assert.False(t,
		policy.InitBlocksRuntime(
			&api.AttributeSpec{
				Role: api.RoleOptional,
				Optional: &api.OptionalConfig{
					Collect: api.InputCollectNone,
				},
			},
			true,
		))
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
	attr := func(collect api.InputCollect) *api.AttributeSpec {
		return &api.AttributeSpec{
			Role:     api.RoleRequired,
			Required: &api.RequiredConfig{Collect: collect},
		}
	}
	assert.False(t,
		policy.RequiredInputMissing(attr(api.InputCollectFirst), true))
	assert.False(t,
		policy.RequiredInputMissing(attr(api.InputCollectNone), false))
	assert.True(t,
		policy.RequiredInputMissing(attr(api.InputCollectFirst), false))
}

func TestProviderOutputNeeded(t *testing.T) {
	assert.False(t, policy.ProviderOutputNeeded(api.InputCollectFirst,
		policy.InputOutlook{
			HasValue:       true,
			AllCollectable: true,
		}))
	assert.True(t, policy.ProviderOutputNeeded(api.InputCollectFirst,
		policy.InputOutlook{
			HasValue:       false,
			AllCollectable: true,
		}))
	assert.True(t, policy.ProviderOutputNeeded(api.InputCollectLast,
		policy.InputOutlook{
			HasValue:       true,
			AllCollectable: true,
		}))
	assert.True(t, policy.ProviderOutputNeeded(api.InputCollectSome,
		policy.InputOutlook{
			HasValue:       true,
			AllCollectable: true,
		}))
	assert.True(t, policy.ProviderOutputNeeded(api.InputCollectAll,
		policy.InputOutlook{
			HasValue:       true,
			AllCollectable: true,
		}))
	assert.False(t, policy.ProviderOutputNeeded(api.InputCollectAll,
		policy.InputOutlook{
			HasValue:       true,
			AllCollectable: false,
		}))
	assert.False(t, policy.ProviderOutputNeeded(api.InputCollectAll,
		policy.InputOutlook{
			HasInit:        true,
			HasValue:       false,
			AllCollectable: true,
		}))
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

	none := api.InputCollectNone
	assert.True(t, policy.InputFulfilled(none, 0, terminal, false))
	assert.False(t, policy.InputFulfilled(none, 1, terminal, false))
	assert.False(t, policy.InputFulfilled(none, 0, incomplete, false))
	assert.True(t, policy.InputFulfilled(none, 0, terminal, true))

	first := api.InputCollectFirst
	assert.True(t, policy.InputFulfilled(first, 1, incomplete, false))
	assert.False(t, policy.InputFulfilled(first, 0, terminal, false))
	assert.False(t, policy.InputFulfilled(first, 0, terminal, true))

	last := api.InputCollectLast
	assert.True(t, policy.InputFulfilled(last, 1, terminal, false))
	assert.False(t, policy.InputFulfilled(last, 1, incomplete, false))
	assert.False(t, policy.InputFulfilled(last, 0, terminal, true))

	some := api.InputCollectSome
	assert.True(t, policy.InputFulfilled(some, 1, terminal, false))
	assert.False(t, policy.InputFulfilled(some, 1, incomplete, false))
	assert.False(t, policy.InputFulfilled(some, 0, terminal, false))
	assert.True(t, policy.InputFulfilled(some, 0, terminal, true))

	all := api.InputCollectAll
	assert.True(t, policy.InputFulfilled(all, 1, terminal, false))
	assert.False(t, policy.InputFulfilled(all, 1, incomplete, false))
	assert.False(t, policy.InputFulfilled(all, 1, partial, false))
	assert.False(t, policy.InputFulfilled(all, 0, terminal, false))
	assert.True(t, policy.InputFulfilled(all, 0, terminal, true))
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
	assert.True(t, policy.InputSatisfiable(all, policy.InputOutlook{
		HasInit: true, AllCollectable: true, Providable: no,
	}))

	assert.True(t, policy.InputSatisfiable(api.InputCollectSome,
		policy.InputOutlook{HasInit: true, Providable: no},
	))
	assert.False(t, policy.InputSatisfiable(api.InputCollectFirst,
		policy.InputOutlook{HasInit: true, Providable: no},
	))
	assert.False(t, policy.InputSatisfiable(api.InputCollectLast,
		policy.InputOutlook{HasInit: true, Providable: no},
	))

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

	assert.Equal(t,
		base.Add(time.Minute),
		policy.FulfilledAt(api.InputCollectFirst, values, done))
	for _, collect := range []api.InputCollect{
		api.InputCollectLast, api.InputCollectSome, api.InputCollectAll,
	} {
		t.Run(string(collect), func(t *testing.T) {
			assert.Equal(t, done, policy.FulfilledAt(collect, values, done))
			at := base.Add(3 * time.Minute)
			if collect == api.InputCollectLast {
				at = values[len(values)-1].SetAt
			}
			assert.Equal(t, at, policy.FulfilledAt(collect, values, base))
			assert.Equal(t,
				at, policy.FulfilledAt(collect, values, time.Time{}))
		})
	}
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

	_, ok = policy.ResolveInputValue(api.InputCollectLast, nil)
	assert.False(t, ok)

	val, ok = policy.ResolveInputValue(api.InputCollectSome, nil)
	assert.True(t, ok)
	assert.Equal(t, []any{}, val)

	val, ok = policy.ResolveInputValue(api.InputCollectAll, nil)
	assert.True(t, ok)
	assert.Equal(t, []any{}, val)
}
