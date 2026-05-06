package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestInitPolicy(t *testing.T) {
	assert.True(t, InitSatisfiesInput(api.InputCollectFirst, true))
	assert.False(t, InitSatisfiesInput(api.InputCollectLast, true))
	assert.False(t, InitSatisfiesInput(api.InputCollectFirst, false))

	assert.True(t, InitBlocksInput(api.InputCollectNone, true))
	assert.False(t, InitBlocksInput(api.InputCollectFirst, true))
	assert.False(t, InitBlocksInput(api.InputCollectNone, false))
}

func TestRequiredInputMissing(t *testing.T) {
	assert.False(t, RequiredInputMissing(api.InputCollectFirst, true, false))
	assert.False(t, RequiredInputMissing(api.InputCollectNone, false, false))
	assert.False(t, RequiredInputMissing(api.InputCollectFirst, false, true))
	assert.True(t, RequiredInputMissing(api.InputCollectFirst, false, false))
}

func TestProviderOutputNeeded(t *testing.T) {
	assert.False(t, ProviderOutputNeeded(api.InputCollectFirst, true, true))
	assert.True(t, ProviderOutputNeeded(api.InputCollectFirst, false, true))
	assert.True(t, ProviderOutputNeeded(api.InputCollectLast, true, true))
	assert.True(t, ProviderOutputNeeded(api.InputCollectSome, true, true))
	assert.True(t, ProviderOutputNeeded(api.InputCollectAll, true, true))
	assert.False(t, ProviderOutputNeeded(api.InputCollectAll, true, false))
}

func TestInputFulfilled(t *testing.T) {
	terminal := ProviderSummary{Terminal: true, AllSucceeded: true}
	incomplete := ProviderSummary{Terminal: false, AllSucceeded: true}
	partial := ProviderSummary{Terminal: true, AllSucceeded: false}

	assert.True(t, InputFulfilled(api.InputCollectNone, 0, terminal))
	assert.False(t, InputFulfilled(api.InputCollectNone, 1, terminal))
	assert.False(t, InputFulfilled(api.InputCollectNone, 0, incomplete))

	assert.True(t, InputFulfilled(api.InputCollectFirst, 1, incomplete))
	assert.False(t, InputFulfilled(api.InputCollectFirst, 0, terminal))

	assert.True(t, InputFulfilled(api.InputCollectLast, 1, terminal))
	assert.False(t, InputFulfilled(api.InputCollectLast, 1, incomplete))

	assert.True(t, InputFulfilled(api.InputCollectSome, 1, terminal))
	assert.False(t, InputFulfilled(api.InputCollectSome, 1, incomplete))

	assert.True(t, InputFulfilled(api.InputCollectAll, 1, terminal))
	assert.False(t, InputFulfilled(api.InputCollectAll, 1, incomplete))
	assert.False(t, InputFulfilled(api.InputCollectAll, 1, partial))
}

func TestResolveInputValue(t *testing.T) {
	values := []*api.AttributeValue{
		{Value: "a"},
		{Value: "b"},
	}

	val, ok := ResolveInputValue(api.InputCollectFirst, values)
	assert.True(t, ok)
	assert.Equal(t, "a", val)

	val, ok = ResolveInputValue(api.InputCollectLast, values)
	assert.True(t, ok)
	assert.Equal(t, "b", val)

	val, ok = ResolveInputValue(api.InputCollectSome, values)
	assert.True(t, ok)
	assert.Equal(t, []any{"a", "b"}, val)

	val, ok = ResolveInputValue(api.InputCollectAll, values)
	assert.True(t, ok)
	assert.Equal(t, []any{"a", "b"}, val)

	_, ok = ResolveInputValue(api.InputCollectNone, values)
	assert.False(t, ok)

	_, ok = ResolveInputValue(api.InputCollectFirst, nil)
	assert.False(t, ok)
}
