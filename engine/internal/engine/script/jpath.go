package script

import (
	"errors"
	"fmt"

	"github.com/kode4food/jpath"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// JPathEnv provides JPath predicate evaluation
type JPathEnv struct {
	*compiler[jpath.Path]
}

const jpathCacheSize = 10240

var (
	ErrJPathCompile = errors.New("jpath compile error")
	ErrJPathNoMatch = errors.New("jpath produced no matches")
)

// NewJPathEnv creates a JPath predicate evaluation environment
func NewJPathEnv() *JPathEnv {
	env := &JPathEnv{}
	env.compiler = newCompiler(jpathCacheSize,
		func(_ *api.Step, cfg *api.ScriptConfig) (jpath.Path, error) {
			return env.compile(cfg.Script)
		},
	)
	return env
}

// ExecuteScript evaluates a compiled JPath expression against mapping inputs
func (e *JPathEnv) ExecuteScript(
	c Compiled, _ *api.Step, inputs api.Args,
) (api.Args, error) {
	doc := marshalJPathValue(inputs)
	if len(inputs) == 1 {
		for _, v := range inputs {
			doc = marshalJPathValue(v)
		}
	}

	value, ok := collapseJPathMatches(c.(jpath.Path)(doc))
	if !ok {
		return nil, ErrJPathNoMatch
	}
	return api.Args{"value": value}, nil
}

// EvaluatePredicate applies the compiled JPath expression and treats any
// match as predicate success, including explicit null matches
func (e *JPathEnv) EvaluatePredicate(
	c Compiled, _ *api.Step, inputs api.Args,
) (bool, error) {
	doc := marshalJPathValue(inputs)
	matches := c.(jpath.Path)(doc)
	return len(matches) > 0, nil
}

// EvaluateMatch applies the compiled JPath expression to a single input and
// treats any match as success, including explicit null matches
func (e *JPathEnv) EvaluateMatch(c Compiled, input any) (bool, error) {
	doc := marshalJPathValue(input)
	matches := c.(jpath.Path)(doc)
	return len(matches) > 0, nil
}

func (e *JPathEnv) compile(source string) (jpath.Path, error) {
	parsed, err := jpath.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrJPathCompile, source)
	}

	compiled, err := jpath.Compile(parsed)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrJPathCompile, source)
	}
	return compiled, nil
}

func collapseJPathMatches(matches []any) (any, bool) {
	switch len(matches) {
	case 0:
		return nil, false
	case 1:
		return matches[0], true
	default:
		return matches, true
	}
}

func marshalJPathValue(value any) any {
	switch v := value.(type) {
	case api.Args:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[string(key)] = marshalJPathValue(elem)
		}
		return out
	case map[api.Name]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[string(key)] = marshalJPathValue(elem)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[key] = marshalJPathValue(elem)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for idx, elem := range v {
			out[idx] = marshalJPathValue(elem)
		}
		return out
	default:
		return value
	}
}
