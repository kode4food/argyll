package script

import (
	"errors"
	"fmt"

	"github.com/kode4food/argex"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// ArgExEnv evaluates Argyll Expressions
type ArgExEnv struct {
	*compiler[argex.Program]
}

const argExCacheSize = 10240

var (
	ErrArgExCompile = errors.New("ArgEx compile error")
	ErrArgExNoValue = errors.New("ArgEx produced no value")
)

// NewArgExEnv creates an Argyll Expressions evaluation environment
func NewArgExEnv() *ArgExEnv {
	return &ArgExEnv{
		compiler: newCompiler(
			argExCacheSize,
			func(_ *api.Step, cfg *api.ScriptConfig) (argex.Program, error) {
				return compileArgEx(cfg.Script)
			},
		),
	}
}

// EvaluateScript evaluates an expression against named mapping inputs
func (*ArgExEnv) EvaluateScript(
	c Compiled, _ *api.Step, inputs api.Args,
) (api.Args, error) {
	value, err := c.(argex.Program)(marshalArgExValue(inputs).(map[string]any))
	if err != nil {
		return nil, err
	}
	if argex.IsMissing(value) {
		return nil, ErrArgExNoValue
	}
	return api.Args{"value": value}, nil
}

// EvaluatePredicate evaluates a Boolean expression against Step inputs
func (*ArgExEnv) EvaluatePredicate(
	c Compiled, _ *api.Step, inputs api.Args,
) (bool, error) {
	value, err := c.(argex.Program)(marshalArgExValue(inputs).(map[string]any))
	if err != nil {
		return false, err
	}
	result, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf(
			"%w: predicate must return Boolean", argex.ErrType,
		)
	}
	return result, nil
}

// EvaluateMatch exposes a single candidate as value to a Boolean expression
func (e *ArgExEnv) EvaluateMatch(c Compiled, input any) (bool, error) {
	return e.EvaluatePredicate(c, nil, api.Args{matchValue: input})
}

func compileArgEx(source string) (argex.Program, error) {
	expr, err := argex.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArgExCompile, err)
	}
	program, err := argex.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArgExCompile, err)
	}
	return program, nil
}

func marshalArgExValue(value any) any {
	switch v := value.(type) {
	case api.Args:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[string(key)] = marshalArgExValue(elem)
		}
		return out

	case map[api.Name]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[string(key)] = marshalArgExValue(elem)
		}
		return out

	case map[string]any:
		out := make(map[string]any, len(v))
		for key, elem := range v {
			out[key] = marshalArgExValue(elem)
		}
		return out

	case []any:
		out := make([]any, len(v))
		for i, elem := range v {
			out[i] = marshalArgExValue(elem)
		}
		return out

	default:
		return value
	}
}
