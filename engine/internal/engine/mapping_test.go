package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestCompileMapping(t *testing.T) {
	t.Run("compiles valid mapping", func(t *testing.T) {
		path, err := compileMapping("$.foo")
		assert.NoError(t, err)
		assert.NotNil(t, path)
	})

	t.Run("returns parse error for invalid syntax", func(t *testing.T) {
		_, err := compileMapping("$..[")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMapping)
		assert.ErrorContains(t, err, "$..[")
	})

	t.Run("returns compile error for unknown function", func(t *testing.T) {
		_, err := compileMapping("$[?unknown(@)]")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMapping)
		assert.ErrorContains(t, err, "$[?unknown(@)]")
	})
}

func TestApplyMapping(t *testing.T) {
	t.Run("returns nil for empty mapping", func(t *testing.T) {
		res, err := applyMapping("", api.Args{"foo": "value"})
		assert.NoError(t, err)
		assert.Nil(t, res)
	})

	t.Run("applies valid mapping", func(t *testing.T) {
		res, err := applyMapping("$.foo", api.Args{"foo": "value"})
		assert.NoError(t, err)
		assert.Equal(t, []any{"value"}, res)
	})

	t.Run("returns error for invalid mapping", func(t *testing.T) {
		_, err := applyMapping("$..[", api.Args{"foo": "value"})
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMapping)
	})
}

func TestMappingValue(t *testing.T) {
	t.Run("returns value as-is for empty mapping", func(t *testing.T) {
		input := api.Args{"foo": "bar"}
		value, ok, err := mappingValue("", input)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, input, value)
	})

	t.Run("returns no value when mapping finds nothing", func(t *testing.T) {
		value, ok, err := mappingValue("$.missing", api.Args{"foo": "bar"})
		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Nil(t, value)
	})

	t.Run("returns scalar when mapping finds one value", func(t *testing.T) {
		value, ok, err := mappingValue("$.foo", api.Args{"foo": "bar"})
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "bar", value)
	})

	t.Run("returns slice when mapping finds many values", func(t *testing.T) {
		input := api.Args{
			"payload": map[string]any{
				"sections": []any{
					map[string]any{"book": "A"},
					map[string]any{"book": "B"},
				},
			},
		}
		value, ok, err := mappingValue("$..book", input)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []any{"A", "B"}, value)
	})

	t.Run("returns error when mapping is invalid", func(t *testing.T) {
		value, ok, err := mappingValue("$..[", api.Args{"foo": "bar"})
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidMapping)
		assert.False(t, ok)
		assert.Nil(t, value)
	})
}

func TestNormalizeMappingDoc(t *testing.T) {
	input := api.Args{
		"args": api.Args{
			"inner": "value",
		},
		"named": map[api.Name]any{
			"namedKey": "namedValue",
		},
		"plain": map[string]any{
			"plainKey": "plainValue",
		},
		"list": []any{
			map[api.Name]any{"listNamed": "x"},
			api.Args{"listArgs": "y"},
			123,
		},
		"scalar": 42,
	}

	raw := normalizeMappingDoc(input)
	doc, ok := raw.(map[string]any)
	assert.True(t, ok)

	argsMap, ok := doc["args"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "value", argsMap["inner"])

	namedMap, ok := doc["named"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "namedValue", namedMap["namedKey"])

	plainMap, ok := doc["plain"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "plainValue", plainMap["plainKey"])

	list, ok := doc["list"].([]any)
	assert.True(t, ok)
	assert.Len(t, list, 3)

	listNamed, ok := list[0].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "x", listNamed["listNamed"])

	listArgs, ok := list[1].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "y", listArgs["listArgs"])

	assert.Equal(t, 123, list[2])
	assert.Equal(t, 42, doc["scalar"])
}
