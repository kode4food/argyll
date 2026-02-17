package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func withMapper(t *testing.T, fn func(*engine.Mapper)) {
	t.Helper()
	helpers.WithEngine(t, func(eng *engine.Engine) {
		fn(engine.NewMapper(eng))
	})
}

func TestMapperCompilePath(t *testing.T) {
	t.Run("compiles valid mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			path, err := m.CompilePath("$.foo")
			assert.NoError(t, err)
			assert.NotNil(t, path)
		})
	})

	t.Run("returns parse error for invalid syntax", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			_, err := m.CompilePath("$..[")
			assert.Error(t, err)
			assert.ErrorIs(t, err, engine.ErrInvalidMapping)
			assert.ErrorContains(t, err, "$..[")
		})
	})

	t.Run("returns compile error for unknown function", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			_, err := m.CompilePath("$[?unknown(@)]")
			assert.Error(t, err)
			assert.ErrorIs(t, err, engine.ErrInvalidMapping)
			assert.ErrorContains(t, err, "$[?unknown(@)]")
		})
	})
}

func TestMapperApply(t *testing.T) {
	t.Run("returns nil for empty mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			res, err := m.Apply("", api.Args{"foo": "value"})
			assert.NoError(t, err)
			assert.Nil(t, res)
		})
	})

	t.Run("applies valid mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			res, err := m.Apply("$.foo", api.Args{"foo": "value"})
			assert.NoError(t, err)
			assert.Equal(t, []any{"value"}, res)
		})
	})

	t.Run("returns error for invalid mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			_, err := m.Apply("$..[", api.Args{"foo": "value"})
			assert.Error(t, err)
			assert.ErrorIs(t, err, engine.ErrInvalidMapping)
		})
	})
}

func TestMapperMappingValue(t *testing.T) {
	t.Run("returns value as-is for empty mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			input := api.Args{"foo": "bar"}
			value, ok, err := m.MappingValue("", input)
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, input, value)
		})
	})

	t.Run("returns no value when mapping finds nothing", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			value, ok, err := m.MappingValue("$.missing", api.Args{"foo": "bar"})
			assert.NoError(t, err)
			assert.False(t, ok)
			assert.Nil(t, value)
		})
	})

	t.Run("returns scalar when mapping finds one value", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			value, ok, err := m.MappingValue("$.foo", api.Args{"foo": "bar"})
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, "bar", value)
		})
	})

	t.Run("returns slice when mapping finds many values", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			input := api.Args{
				"payload": map[string]any{
					"sections": []any{
						map[string]any{"book": "A"},
						map[string]any{"book": "B"},
					},
				},
			}
			value, ok, err := m.MappingValue("$..book", input)
			assert.NoError(t, err)
			assert.True(t, ok)
			assert.Equal(t, []any{"A", "B"}, value)
		})
	})

	t.Run("returns error when mapping is invalid", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			value, ok, err := m.MappingValue("$..[", api.Args{"foo": "bar"})
			assert.Error(t, err)
			assert.ErrorIs(t, err, engine.ErrInvalidMapping)
			assert.False(t, ok)
			assert.Nil(t, value)
		})
	})
}

func TestMapperNormalization(t *testing.T) {
	withMapper(t, func(m *engine.Mapper) {
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

		v, ok, err := m.MappingValue("$.args.inner", input)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "value", v)

		v, ok, err = m.MappingValue("$.named.namedKey", input)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "namedValue", v)

		v, ok, err = m.MappingValue("$.plain.plainKey", input)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "plainValue", v)

		v, ok, err = m.MappingValue("$.list[0].listNamed", input)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "x", v)

		v, ok, err = m.MappingValue("$.list[1].listArgs", input)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "y", v)

		v, ok, err = m.MappingValue("$.list[2]", input)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, 123, v)

		v, ok, err = m.MappingValue("$.scalar", input)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, 42, v)
	})
}
