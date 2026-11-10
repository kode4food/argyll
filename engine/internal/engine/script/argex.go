package script

import (
	"errors"
	"fmt"

	"github.com/kode4food/argex"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// ArgExEnv evaluates Argyll Expressions
	ArgExEnv struct {
		*compiler[*compiledArgEx]
	}

	// compiledArgEx represents a compiled Argyll Expression and the derived short names it can read
	// bare
	compiledArgEx struct {
		*argex.Program
		aliases []bareName
	}
)

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
			func(st *api.Step, cfg *api.ScriptConfig) (*compiledArgEx, error) {
				program, err := compileArgEx(cfg.Source)
				if err != nil {
					return nil, err
				}
				return &compiledArgEx{
					Program: program,
					aliases: argExAliases(st),
				}, nil
			},
		),
	}
}

// Evaluate evaluates an expression against named inputs
func (*ArgExEnv) Evaluate(c Compiled, _ *api.Step, inputs api.Args) (any, error) {
	proc := c.(*compiledArgEx)
	args := withAliases(inputs, proc.aliases)
	value, err := proc.Eval(marshalArgExValue(args))
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
	result := c.(*compiledArgEx).StaticType()
	if result.Known() && !result.Has(argex.TypeBool) {
		return fmt.Errorf("%w, not %s", ErrNotBoolean, result)
	}
	return nil
}

func compileArgEx(src string) (*argex.Program, error) {
	expr, err := argex.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArgExCompile, err)
	}
	program, err := argex.DefaultEnv().Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrArgExCompile, err)
	}
	return program, nil
}

func argExAliases(st *api.Step) []bareName {
	if st == nil {
		return nil
	}
	names := bareNames(st.SortedArgNames(), identPattern.MatchString)
	return aliasesOf(names)
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
