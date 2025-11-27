package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kode4food/ale"
	"github.com/kode4food/ale/core/bootstrap"
	"github.com/kode4food/ale/data"
	"github.com/kode4food/ale/env"
	"github.com/kode4food/ale/eval"

	"github.com/kode4food/spuds/engine/internal/util"
	"github.com/kode4food/spuds/engine/pkg/api"
)

// AleEnv provides an Ale script execution environment
type AleEnv struct {
	env   *env.Environment
	cache *util.LRUCache[data.Procedure]
}

const (
	aleLambdaTemplate = "(lambda (%s) %s)"
	aleCacheSize      = 1024
)

var (
	ErrAleBadCompiledType = errors.New("expected data.Procedure")
	ErrAleNotProcedure    = errors.New("not a procedure")
	ErrAleCompile         = errors.New("script compile error")
	ErrAleCall            = errors.New("error calling procedure")
)

// NewAleEnv creates a new Ale script execution environment with the standard
// library bootstrapped
func NewAleEnv() *AleEnv {
	e := env.NewEnvironment()
	bootstrap.Into(e)
	return &AleEnv{
		env:   e,
		cache: util.NewLRUCache[data.Procedure](aleCacheSize),
	}
}

// Compile compiles a script configuration
func (e *AleEnv) Compile(
	step *api.Step, cfg *api.ScriptConfig,
) (Compiled, error) {
	if cfg.Script == "" {
		return nil, nil
	}

	argNames := step.SortedArgNames()
	src := fmt.Sprintf(
		aleLambdaTemplate, strings.Join(argNames, " "), cfg.Script,
	)

	return e.cache.Get(src, func() (data.Procedure, error) {
		return e.compileSource(src)
	})
}

// Validate checks if an Ale script is syntactically correct without running it
func (e *AleEnv) Validate(step *api.Step, script string) error {
	_, err := e.Compile(step, &api.ScriptConfig{
		Script:   script,
		Language: api.ScriptLangAle,
	})
	return err
}

// ExecuteScript runs a compiled Ale procedure with the provided inputs and
// returns the output arguments
func (e *AleEnv) ExecuteScript(
	c Compiled, step *api.Step, inputs api.Args,
) (api.Args, error) {
	proc, ok := c.(data.Procedure)
	if !ok {
		return nil, fmt.Errorf("%w, got %T", ErrAleBadCompiledType, c)
	}

	result, err := executeScript(proc, step, inputs)
	if err != nil {
		return nil, err
	}

	jsonValue := aleToJSON(result)

	m, ok := jsonValue.(map[string]any)
	if !ok {
		return api.Args{"result": jsonValue}, nil
	}

	args := make(api.Args, len(m))
	for k, v := range m {
		args[api.Name(k)] = v
	}
	return args, nil
}

// EvaluatePredicate executes a compiled Ale predicate with the provided inputs
// and returns the boolean result
func (e *AleEnv) EvaluatePredicate(
	c Compiled, step *api.Step, inputs api.Args,
) (bool, error) {
	proc, ok := c.(data.Procedure)
	if !ok {
		return false, fmt.Errorf("%s, got %T", ErrAleBadCompiledType, c)
	}

	return evaluatePredicate(proc, step, inputs)
}

func (e *AleEnv) compileSource(src string) (proc data.Procedure, err error) {
	return catchPanic(ErrAleCompile,
		func() (data.Procedure, error) {
			ns := e.env.GetAnonymous()
			res, err := eval.String(ns, data.String(src))
			if err != nil {
				return nil, err
			}

			proc, ok := res.(data.Procedure)
			if !ok {
				return nil, fmt.Errorf("%w, got: %T", ErrAleNotProcedure, res)
			}
			return proc, nil
		},
	)
}

func executeScript(
	proc data.Procedure, step *api.Step, inputs api.Args,
) (res ale.Value, err error) {
	names := step.SortedArgNames()

	args := make(data.Vector, 0, len(names))
	for _, name := range names {
		args = append(args, getArgValue(inputs, name))
	}

	return catchPanic(ErrAleCall,
		func() (ale.Value, error) {
			return proc.Call(args...), nil
		},
	)
}

func getArgValue(inputs api.Args, argName string) ale.Value {
	value, ok := inputs[api.Name(argName)]
	if !ok {
		return data.Null
	}
	return jsonToAle(value)
}

func evaluatePredicate(
	proc data.Procedure, step *api.Step, inputs api.Args,
) (bool, error) {
	result, err := executeScript(proc, step, inputs)
	if err != nil {
		return false, err
	}
	return result != data.False, nil
}

func jsonToAle(value any) ale.Value {
	switch v := value.(type) {
	case string:
		return data.String(v)
	case bool:
		return data.Bool(v)
	case int:
		return data.Integer(v)
	case int64:
		return data.Integer(v)
	case float64:
		return data.Float(v)
	case []any:
		return jsonArrayToAle(v)
	case map[string]any:
		return jsonMapToAle(v)
	case nil:
		return data.Null
	default:
		return data.String(fmt.Sprintf("%v", v))
	}
}

func jsonArrayToAle(arr []any) data.Vector {
	vec := make(data.Vector, len(arr))
	for i, item := range arr {
		vec[i] = jsonToAle(item)
	}
	return vec
}

func jsonMapToAle(m map[string]any) *data.Object {
	obj := data.NewObject()
	for k, val := range m {
		pair := data.NewCons(data.Keyword(k), jsonToAle(val))
		obj = obj.Put(pair).(*data.Object)
	}
	return obj
}

func aleToJSON(value ale.Value) any {
	switch v := value.(type) {
	case data.Bool:
		return bool(v)
	case data.Keyword:
		return string(v)
	case data.Integer:
		return int(v)
	case data.Float:
		return float64(v)
	case data.Vector:
		return aleVectorToJSON(v)
	case *data.List:
		return aleListToJSON(v)
	case *data.Object:
		return aleObjectToJSON(v)
	default:
		return aleDefaultToJSON(value, v)
	}
}

func aleVectorToJSON(v data.Vector) []any {
	result := make([]any, len(v))
	for i, item := range v {
		result[i] = aleToJSON(item)
	}
	return result
}

func aleListToJSON(list *data.List) []any {
	var result []any
	for l := list; !l.IsEmpty(); {
		head, tail, ok := l.Split()
		if !ok {
			break
		}
		result = append(result, aleToJSON(head))
		l = tail.(*data.List)
	}
	return result
}

func aleObjectToJSON(obj *data.Object) map[string]any {
	result := map[string]any{}
	for _, pair := range obj.Pairs() {
		keyStr := fmt.Sprintf("%v", aleToJSON(pair.Car()))
		result[keyStr] = aleToJSON(pair.Cdr())
	}
	return result
}

func aleDefaultToJSON(value ale.Value, v any) any {
	if value == data.Null {
		return nil
	}
	return fmt.Sprintf("%v", v)
}

func catchPanic[T any](baseErr error, fn func() (T, error)) (res T, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		e, ok := r.(error)
		if ok {
			err = e
			return
		}
		err = fmt.Errorf("%w: %v", baseErr, r)
	}()
	return fn()
}
