package engine

import (
	"errors"
	"fmt"
	"log/slog"

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
	step *api.Step, cfg *api.ScriptConfig,
) (Compiled, error) {
	if cfg == nil || cfg.Script == "" {
		return nil, nil
	}

	compiled, err := m.engine.scripts.Compile(step, cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMapping, cfg.Script)
	}
	return compiled, nil
}

// MapValue applies a mapping script and normalizes result presence
func (m *Mapper) MapValue(
	step *api.Step, name api.Name, cfg *api.ScriptConfig, value any,
) (any, bool) {
	if cfg == nil || cfg.Script == "" {
		return value, true
	}

	compiled, err := m.Compile(step, cfg)
	if err != nil {
		return nil, false
	}

	env, err := m.engine.scripts.Get(cfg.Language)
	if err != nil {
		return nil, false
	}

	result, err := env.ExecuteScript(compiled, step, api.Args{name: value})
	if err != nil {
		return nil, false
	}
	return extractScriptResult(result), true
}

// MapInput maps a step input value and falls back to original value
func (m *Mapper) MapInput(
	step *api.Step, name api.Name, attr *api.AttributeSpec, value any,
) any {
	if attr.Mapping == nil || attr.Mapping.Script == nil {
		return value
	}

	argName := m.InputParamName(name, attr)
	if mapped, ok := m.MapValue(step, argName, attr.Mapping.Script, value); ok {
		return mapped
	}

	slog.Warn("Input mapping failed; using original value",
		log.StepID(step.ID),
		slog.String("attribute", string(name)),
		slog.String("language", attr.Mapping.Script.Language),
	)
	return value
}

// InputParamName resolves the outbound parameter name for a mapped input
func (m *Mapper) InputParamName(
	attrName api.Name, attr *api.AttributeSpec,
) api.Name {
	if attr.Mapping != nil && attr.Mapping.Name != "" {
		return api.Name(attr.Mapping.Name)
	}
	return attrName
}

// MapOutputs maps raw step outputs to declared output attributes
func (m *Mapper) MapOutputs(step *api.Step, outputs api.Args) api.Args {
	if step == nil {
		return outputs
	}

	res := api.Args{}
	for name, attr := range step.Attributes {
		if !attr.IsOutput() {
			continue
		}

		value, ok := m.mapOutput(step, name, attr, outputs)
		if ok {
			res[name] = value
		}
	}
	return res
}

func (m *Mapper) mapOutput(
	step *api.Step, name api.Name, attr *api.AttributeSpec, outputs api.Args,
) (any, bool) {
	if attr.Mapping != nil && attr.Mapping.Script != nil {
		return m.MapValue(step, name, attr.Mapping.Script, outputs)
	}
	return m.outputByName(name, attr, outputs)
}

func (m *Mapper) outputByName(
	name api.Name, attr *api.AttributeSpec, outputs api.Args,
) (any, bool) {
	sourceKey := name
	if attr.Mapping != nil && attr.Mapping.Name != "" {
		sourceKey = api.Name(attr.Mapping.Name)
	}
	value, ok := outputs[sourceKey]
	return value, ok
}
