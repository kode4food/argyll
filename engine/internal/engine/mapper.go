package engine

import (
	"errors"
	"fmt"

	"github.com/kode4food/jpath"

	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	ErrInvalidMapping   = errors.New("invalid mapping")
	ErrJPathEnvInvalid  = errors.New("invalid jpath environment")
	ErrJPathEnvNotFound = errors.New("jpath environment not found")
)

// Mapper evaluates attribute mappings through the engine script registry
type Mapper struct {
	engine *Engine
}

// NewMapper creates a mapping evaluator bound to an engine
func NewMapper(engine *Engine) *Mapper {
	return &Mapper{
		engine: engine,
	}
}

// Apply executes a mapping expression against the provided value
func (m *Mapper) Apply(mapping string, value any) ([]any, error) {
	if mapping == "" {
		return nil, nil
	}

	path, err := m.CompilePath(mapping)
	if err != nil {
		return nil, err
	}

	return path(normalizeMappingDoc(value)), nil
}

// CompilePath compiles a mapping expression into an executable JPath path
func (m *Mapper) CompilePath(mapping string) (jpath.Path, error) {
	env, err := m.engine.scripts.Get(api.ScriptLangJPath)
	if err != nil {
		return nil, ErrJPathEnvNotFound
	}

	jPathEnv, ok := env.(*JPathEnv)
	if !ok {
		return nil, ErrJPathEnvInvalid
	}

	compiled, err := jPathEnv.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   mapping,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMapping, mapping)
	}

	path, ok := compiled.(jpath.Path)
	if !ok || path == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMapping, mapping)
	}

	return path, nil
}

// MappingValue applies a mapping and normalizes 0/1/N matches
func (m *Mapper) MappingValue(mapping string, value any) (any, bool, error) {
	if mapping == "" {
		return value, true, nil
	}

	res, err := m.Apply(mapping, value)
	if err != nil {
		return nil, false, err
	}

	switch len(res) {
	case 0:
		return nil, false, nil
	case 1:
		return res[0], true, nil
	default:
		return res, true, nil
	}
}

func normalizeMappingDoc(value any) any {
	switch v := value.(type) {
	case api.Args:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[string(key)] = normalizeMappingDoc(elem)
		}
		return out
	case map[api.Name]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[string(key)] = normalizeMappingDoc(elem)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[key] = normalizeMappingDoc(elem)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for idx, elem := range v {
			out[idx] = normalizeMappingDoc(elem)
		}
		return out
	default:
		return value
	}
}

// CompileMapping validates that a mapping expression compiles
func (m *Mapper) CompileMapping(mapping string) error {
	_, err := m.CompilePath(mapping)
	return err
}
