package api_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	testMap   map[string]int
	testSlice []string
)

func TestGenericImmutableFunctionsPreserveNamedTypes(t *testing.T) {
	m := testMap{"a": 1}
	set := api.Set(m, "b", 2)
	deleted := api.Delete(set, "a")
	applied := api.Apply(deleted, testMap{"c": 3})

	assert.Equal(t, testMap{"a": 1}, m)
	assert.Equal(t, testMap{"b": 2, "c": 3}, applied)

	s := testSlice{"a"}
	appended := api.Append(s, "b")
	removed := api.Remove(appended, func(v string) bool { return v == "a" })

	assert.Equal(t, testSlice{"a"}, s)
	assert.Equal(t, testSlice{"b"}, removed)
}

func TestMapSet(t *testing.T) {
	t.Run("creates the map when nil", func(t *testing.T) {
		var original api.Map[string, int]

		result := original.Set("a", 1)

		assert.Nil(t, original)
		assert.NotNil(t, result)
		assert.Equal(t, api.Map[string, int]{"a": 1}, result)
	})

	t.Run("creates the map when zero valued", func(t *testing.T) {
		original := api.Map[string, int]{}

		result := original.Set("a", 1)

		assert.Empty(t, original)
		assert.Equal(t, api.Map[string, int]{"a": 1}, result)
	})

	t.Run("leaves the receiver untouched", func(t *testing.T) {
		original := api.Map[string, int]{"a": 1}

		result := original.Set("b", 2)

		assert.Equal(t, api.Map[string, int]{"a": 1}, original)
		assert.Equal(t, api.Map[string, int]{"a": 1, "b": 2}, result)
	})

	t.Run("overwrites an existing key", func(t *testing.T) {
		original := api.Map[string, int]{"a": 1}

		result := original.Set("a", 2)

		assert.Equal(t, 1, original["a"])
		assert.Equal(t, 2, result["a"])
		assert.Len(t, result, 1)
	})

	t.Run("does not alias the receiver on later writes", func(t *testing.T) {
		original := api.Map[string, int]{"a": 1}

		first := original.Set("b", 2)
		second := original.Set("b", 3)

		assert.Equal(t, 2, first["b"])
		assert.Equal(t, 3, second["b"])
		assert.NotContains(t, original, "b")
	})

	t.Run("stores a zero value", func(t *testing.T) {
		result := api.Map[string, int]{}.Set("a", 0)

		value, ok := result["a"]
		assert.True(t, ok)
		assert.Equal(t, 0, value)
	})
}

func TestMapDelete(t *testing.T) {
	t.Run("returns the receiver when the key is absent", func(t *testing.T) {
		original := api.Map[string, int]{"a": 1}

		result := original.Delete("b")

		assert.Equal(t, original, result)
	})

	t.Run("tolerates a nil map", func(t *testing.T) {
		var original api.Map[string, int]

		result := original.Delete("a")

		assert.Nil(t, result)
	})

	t.Run("leaves the receiver untouched", func(t *testing.T) {
		original := api.Map[string, int]{"a": 1, "b": 2}

		result := original.Delete("a")

		assert.Equal(t, api.Map[string, int]{"a": 1, "b": 2}, original)
		assert.Equal(t, api.Map[string, int]{"b": 2}, result)
	})

	t.Run("empties without going nil", func(t *testing.T) {
		original := api.Map[string, int]{"a": 1}

		result := original.Delete("a")

		assert.NotNil(t, result)
		assert.Empty(t, result)
	})
}

func TestMapJSON(t *testing.T) {
	t.Run("marshals as an object", func(t *testing.T) {
		b, err := json.Marshal(api.Map[string, int]{"a": 1})
		assert.NoError(t, err)
		assert.JSONEq(t, `{"a":1}`, string(b))
	})

	t.Run("marshals nil as null", func(t *testing.T) {
		var m api.Map[string, int]

		b, err := json.Marshal(m)
		assert.NoError(t, err)
		assert.Equal(t, "null", string(b))
	})

	t.Run("round trips", func(t *testing.T) {
		var result api.Map[string, int]

		err := json.Unmarshal([]byte(`{"a":1,"b":2}`), &result)
		assert.NoError(t, err)
		assert.Equal(t, api.Map[string, int]{"a": 1, "b": 2}, result)
	})
}

func TestSliceAppend(t *testing.T) {
	t.Run("creates the slice when nil", func(t *testing.T) {
		var original api.Slice[string]

		result := original.Append("a")

		assert.Nil(t, original)
		assert.Equal(t, api.Slice[string]{"a"}, result)
	})

	t.Run("leaves the receiver untouched", func(t *testing.T) {
		original := api.Slice[string]{"a"}

		result := original.Append("b")

		assert.Equal(t, api.Slice[string]{"a"}, original)
		assert.Equal(t, api.Slice[string]{"a", "b"}, result)
	})

	t.Run("appends several values", func(t *testing.T) {
		result := api.Slice[string]{"a"}.Append("b", "c")

		assert.Equal(t, api.Slice[string]{"a", "b", "c"}, result)
	})

	t.Run("copies when given no values", func(t *testing.T) {
		original := api.Slice[string]{"a"}

		result := original.Append()

		assert.Equal(t, original, result)

		result[0] = "changed"
		assert.Equal(t, api.Slice[string]{"a"}, original)
	})

	t.Run("returns an empty slice for an empty receiver", func(t *testing.T) {
		var original api.Slice[string]

		result := original.Append()

		assert.Empty(t, result)
	})

	// A receiver with spare capacity is where a plain append would write into
	// the shared backing array
	t.Run("does not share the receiver's spare capacity", func(t *testing.T) {
		original := make(api.Slice[string], 1, 10)
		original[0] = "a"

		first := original.Append("first")
		second := original.Append("second")

		assert.Equal(t, api.Slice[string]{"a", "first"}, first)
		assert.Equal(t, api.Slice[string]{"a", "second"}, second)
		assert.Len(t, original, 1)
	})

	t.Run("allocates no spare capacity", func(t *testing.T) {
		result := make(api.Slice[string], 0, 10).Append("a")

		assert.Equal(t, 1, cap(result))
	})
}

func TestSliceRemove(t *testing.T) {
	isB := func(s string) bool { return s == "b" }

	t.Run("leaves the receiver untouched", func(t *testing.T) {
		original := api.Slice[string]{"a", "b", "c"}

		result := original.Remove(isB)

		assert.Equal(t, api.Slice[string]{"a", "b", "c"}, original)
		assert.Equal(t, api.Slice[string]{"a", "c"}, result)
	})

	t.Run("tolerates a nil slice", func(t *testing.T) {
		var original api.Slice[string]

		result := original.Remove(isB)

		assert.Empty(t, result)
	})

	t.Run("copies when nothing matches", func(t *testing.T) {
		original := api.Slice[string]{"a", "c"}

		result := original.Remove(isB)

		assert.Equal(t, original, result)

		result[0] = "changed"
		assert.Equal(t, api.Slice[string]{"a", "c"}, original)
	})

	t.Run("removes every match", func(t *testing.T) {
		original := api.Slice[string]{"b", "a", "b", "b"}

		result := original.Remove(isB)

		assert.Equal(t, api.Slice[string]{"a"}, result)
	})

	t.Run("empties when everything matches", func(t *testing.T) {
		original := api.Slice[string]{"b", "b"}

		result := original.Remove(isB)

		assert.Empty(t, result)
	})
}

func TestSliceJSON(t *testing.T) {
	t.Run("marshals as an array", func(t *testing.T) {
		b, err := json.Marshal(api.Slice[string]{"a", "b"})
		assert.NoError(t, err)
		assert.JSONEq(t, `["a","b"]`, string(b))
	})

	t.Run("marshals nil as null", func(t *testing.T) {
		var s api.Slice[string]

		b, err := json.Marshal(s)
		assert.NoError(t, err)
		assert.Equal(t, "null", string(b))
	})

	t.Run("round trips", func(t *testing.T) {
		var result api.Slice[string]

		err := json.Unmarshal([]byte(`["a","b"]`), &result)
		assert.NoError(t, err)
		assert.Equal(t, api.Slice[string]{"a", "b"}, result)
	})
}
