package script

import (
	"errors"
	"fmt"

	"github.com/kode4food/argex"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// ArgExEnv evaluates Argyll Expressions
type ArgExEnv struct {
	*compiler[*argex.Program]
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
			func(_ *api.Step, cfg *api.ScriptConfig) (*argex.Program, error) {
				return compileArgEx(cfg.Script)
			},
		),
	}
}

// Evaluate evaluates an expression against named inputs
func (*ArgExEnv) Evaluate(
	c Compiled, _ *api.Step, inputs api.Args,
) (any, error) {
	value, err := c.(*argex.Program).Eval(marshalArgExValue(inputs))
	if err != nil {
		return nil, err
	}
	if argex.IsMissing(value) {
		return nil, ErrArgExNoValue
	}
	return value, nil
}

// CheckPredicate rejects an expression whose result can never be a Boolean
func (*ArgExEnv) CheckPredicate(c Compiled) error {
	result := c.(*argex.Program).StaticType()
	if result.Known() && !result.Has(argex.TypeBool) {
		return fmt.Errorf("%w, not %s", ErrNotBoolean, result)
	}
	return nil
}

func compileArgEx(source string) (*argex.Program, error) {
	expr, err := argex.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArgExCompile, err)
	}
	program, err := argex.DefaultEnv().Compile(expr)
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
