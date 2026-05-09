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
