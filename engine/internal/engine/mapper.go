package engine

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/log"
)

// Mapper evaluates attribute mappings through the engine script registry
type Mapper struct {
	engine *Engine
}

var (
	ErrInvalidMapping = errors.New("invalid mapping")
)

// NewMapper creates a mapping evaluator bound to an engine
func NewMapper(engine *Engine) *Mapper {
	return &Mapper{
		engine: engine,
	}
}

// Compile compiles a mapping script for the provided step context
func (m *Mapper) Compile(
	st *api.Step, cfg *api.ScriptConfig,
) (script.Compiled, error) {
	if cfg == nil || cfg.Script == "" {
		return nil, nil
	}

	compiled, err := m.engine.scripts.Compile(st, cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMapping, cfg.Script)
	}
	return compiled, nil
}

// MapValue applies a mapping script and normalizes result presence
func (m *Mapper) MapValue(
	st *api.Step, name api.Name, cfg *api.ScriptConfig, value any,
) (any, bool) {
	if cfg == nil || cfg.Script == "" {
		return value, true
	}

	compiled, err := m.Compile(st, cfg)
	if err != nil {
		return nil, false
	}

	env, err := m.engine.scripts.Get(cfg.Language)
	if err != nil {
		return nil, false
	}

	result, err := env.ExecuteScript(compiled, st, api.Args{name: value})
	if err != nil {
		return nil, false
	}
	return extractScriptResult(result), true
}

// MapInput maps a step input value and falls back to original value
func (m *Mapper) MapInput(
	st *api.Step, name api.Name, attr *api.AttributeSpec, value any,
) any {
	mapping := attr.Mapping()
	if mapping == nil || mapping.Script == nil {
		return value
	}

	mapped, _ := st.MappedName(name)
	if argName, ok := m.MapValue(st, mapped, mapping.Script, value); ok {
		return argName
	}

	args := []any{
		slog.String("attribute", string(name)),
		slog.String("language", mapping.Script.Language),
	}
	args = append(args, log.StepID(st.ID))
	slog.Warn("Input mapping failed; using original value", args...)
	return value
}

// MapOutputs maps raw step outputs to declared output attributes
func (m *Mapper) MapOutputs(st *api.Step, outputs api.Args) api.Args {
	res := api.Args{}
	for name, attr := range st.Attributes {
		if !attr.IsOutput() {
			continue
		}

		value, ok := m.mapOutput(st, name, attr, outputs)
		if ok {
			res[name] = value
		}
	}
	return res
}

func (m *Mapper) validateStep(st *api.Step) error {
	for name, attr := range st.Attributes {
		mapping := attr.Mapping()
		if mapping == nil || mapping.Script == nil {
			continue
		}

		if _, err := m.Compile(st, mapping.Script); err != nil {
			return fmt.Errorf("%w for attribute %q: %v",
				api.ErrInvalidMappingConfig, name, err,
			)
		}
	}
	return nil
}

func (m *Mapper) mapOutput(
	st *api.Step, name api.Name, attr *api.AttributeSpec, outputs api.Args,
) (any, bool) {
	if mapping := attr.Mapping(); mapping != nil && mapping.Script != nil {
		return m.MapValue(st, name, mapping.Script, outputs)
	}
	return outputByName(st, name, outputs)
}

func outputByName(st *api.Step, name api.Name, outputs api.Args) (any, bool) {
	mapped, _ := st.MappedName(name)
	value, ok := outputs[mapped]
	return value, ok
}

func extractScriptResult(result api.Args) any {
	if len(result) == 0 {
		return nil
	}
	if val, ok := result["value"]; ok {
		return val
	}
	if len(result) == 1 {
		for _, value := range result {
			return value
		}
	}
	return result
}
