package engine_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kode4food/argyll/engine/internal/assert/helpers"
	"github.com/kode4food/argyll/engine/internal/engine"
	"github.com/kode4food/argyll/engine/pkg/api"
)

func TestMapperCompile(t *testing.T) {
	t.Run("compiles jpath mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			compiled, err := m.Compile(nil, jpathCfg("$.foo"))
			assert.NoError(t, err)
			assert.NotNil(t, compiled)
		})
	})

	t.Run("compiles ale mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			step := mappingStep("amount")
			compiled, err := m.Compile(step, aleCfg("(* amount 2)"))
			assert.NoError(t, err)
			assert.NotNil(t, compiled)
		})
	})

	t.Run("returns error for invalid mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			_, err := m.Compile(nil, jpathCfg("$..["))
			assert.Error(t, err)
			assert.ErrorIs(t, err, engine.ErrInvalidMapping)
		})
	})
}

func TestMapperMappingValue(t *testing.T) {
	t.Run("returns value as-is for nil mapping config", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			input := api.Args{"foo": "bar"}
			value, ok := m.MapValue(nil, "input", nil, input)
			assert.True(t, ok)
			assert.Equal(t, input, value)
		})
	})

	t.Run("returns no value when jpath finds nothing", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			value, ok := m.MapValue(
				nil, "input", jpathCfg("$.missing"), api.Args{"foo": "bar"},
			)
			assert.False(t, ok)
			assert.Nil(t, value)
		})
	})

	t.Run("returns scalar when jpath finds one value", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			value, ok := m.MapValue(
				nil, "input", jpathCfg("$.foo"), api.Args{"foo": "bar"},
			)
			assert.True(t, ok)
			assert.Equal(t, "bar", value)
		})
	})

	t.Run("returns slice when jpath finds many values", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			input := api.Args{
				"payload": map[string]any{
					"sections": []any{
						map[string]any{"book": "A"},
						map[string]any{"book": "B"},
					},
				},
			}
			value, ok := m.MapValue(
				nil, "output", jpathCfg("$..book"), input,
			)
			assert.True(t, ok)
			assert.Equal(t, []any{"A", "B"}, value)
		})
	})

	t.Run("executes ale mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			step := mappingStep("amount")
			value, ok := m.MapValue(
				step, "amount", aleCfg("(* amount 2)"), float64(5),
			)
			assert.True(t, ok)
			assert.Equal(t, float64(10), value)
		})
	})

	t.Run("returns no value for invalid mapping script", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			value, ok := m.MapValue(
				nil, "input", jpathCfg("$..["), api.Args{"foo": "bar"},
			)
			assert.False(t, ok)
			assert.Nil(t, value)
		})
	})
}

func TestScriptUsesMappedName(t *testing.T) {
	withMapper(t, func(m *engine.Mapper) {
		step := &api.Step{
			ID:   "mapped-input-script",
			Name: "Mapped Input Script",
			Type: api.StepTypeSync,
			HTTP: &api.HTTPConfig{
				Endpoint: "http://example.com",
			},
			Attributes: api.AttributeSpecs{
				"amount": {
					Role: api.RoleRequired,
					Type: api.TypeNumber,
					Mapping: &api.AttributeMapping{
						Name: "value",
						Script: &api.ScriptConfig{
							Language: api.ScriptLangAle,
							Script:   "(* value 2)",
						},
					},
				},
			},
		}

		attr := step.Attributes["amount"]
		mapped := m.MapInput(step, "amount", attr, float64(5))
		assert.Equal(t, float64(10), mapped)
	})
}

func TestMapperJPathMarshaling(t *testing.T) {
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

		v, ok := m.MapValue(
			nil, "input", jpathCfg("$.args.inner"), input,
		)
		assert.True(t, ok)
		assert.Equal(t, "value", v)

		v, ok = m.MapValue(
			nil, "input", jpathCfg("$.named.namedKey"), input,
		)
		assert.True(t, ok)
		assert.Equal(t, "namedValue", v)

		v, ok = m.MapValue(
			nil, "input", jpathCfg("$.plain.plainKey"), input,
		)
		assert.True(t, ok)
		assert.Equal(t, "plainValue", v)

		v, ok = m.MapValue(
			nil, "input", jpathCfg("$.list[0].listNamed"), input,
		)
		assert.True(t, ok)
		assert.Equal(t, "x", v)

		v, ok = m.MapValue(
			nil, "input", jpathCfg("$.list[1].listArgs"), input,
		)
		assert.True(t, ok)
		assert.Equal(t, "y", v)

		v, ok = m.MapValue(
			nil, "input", jpathCfg("$.list[2]"), input,
		)
		assert.True(t, ok)
		assert.Equal(t, 123, v)

		v, ok = m.MapValue(
			nil, "input", jpathCfg("$.scalar"), input,
		)
		assert.True(t, ok)
		assert.Equal(t, 42, v)
	})
}

func TestMapInput(t *testing.T) {
	t.Run("returns value for nil mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			attr := &api.AttributeSpec{
				Role: api.RoleRequired,
				Type: api.TypeString,
			}
			result := m.MapInput(nil, "input", attr, "hello")
			assert.Equal(t, "hello", result)
		})
	})

	t.Run("returns value for nil script", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			attr := &api.AttributeSpec{
				Role:    api.RoleRequired,
				Type:    api.TypeString,
				Mapping: &api.AttributeMapping{Name: "renamed"},
			}
			result := m.MapInput(nil, "input", attr, "hello")
			assert.Equal(t, "hello", result)
		})
	})

	t.Run("maps value with jpath", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			attr := &api.AttributeSpec{
				Role: api.RoleRequired,
				Type: api.TypeString,
				Mapping: &api.AttributeMapping{
					Script: jpathCfg("$.key"),
				},
			}
			input := api.Args{"key": "mapped"}
			result := m.MapInput(nil, "input", attr, input)
			assert.Equal(t, "mapped", result)
		})
	})

	t.Run("returns original on failed mapping", func(t *testing.T) {
		withMapper(t, func(m *engine.Mapper) {
			step := mappingStep("input")
			attr := &api.AttributeSpec{
				Role: api.RoleRequired,
				Type: api.TypeString,
				Mapping: &api.AttributeMapping{
					Script: jpathCfg("$.missing"),
				},
			}
			input := api.Args{"key": "value"}
			result := m.MapInput(step, "input", attr, input)
			assert.Equal(t, input, result)
		})
	})
}

func mappingStep(name api.Name) *api.Step {
	return &api.Step{
		ID:   "mapping-step",
		Name: "Mapping Step",
		Type: api.StepTypeSync,
		HTTP: &api.HTTPConfig{
			Endpoint: "http://example.com",
			Timeout:  30 * api.Second,
		},
		Attributes: api.AttributeSpecs{
			name: {
				Role: api.RoleRequired,
				Type: api.TypeNumber,
			},
		},
	}
}

func jpathCfg(script string) *api.ScriptConfig {
	return &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   script,
	}
}

func aleCfg(script string) *api.ScriptConfig {
	return &api.ScriptConfig{
		Language: api.ScriptLangAle,
		Script:   script,
	}
}

func withMapper(t *testing.T, fn func(*engine.Mapper)) {
	t.Helper()
	helpers.WithEngine(t, func(eng *engine.Engine) {
		fn(engine.NewMapper(eng))
	})
}
