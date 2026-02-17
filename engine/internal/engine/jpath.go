package engine

import (
	"errors"
	"fmt"

	"github.com/kode4food/jpath"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// JPathEnv provides JPath predicate evaluation
type JPathEnv struct {
	*scriptCompiler[jpath.Path]
}

const jpathCacheSize = 10240

var (
	ErrJPathBadCompiledType = errors.New("expected jpath path")
	ErrJPathCompile         = errors.New("jpath compile error")
	ErrJPathExecuteScript   = errors.New("jpath cannot execute step scripts")
)

// NewJPathEnv creates a JPath predicate evaluation environment
func NewJPathEnv() *JPathEnv {
	env := &JPathEnv{}
	env.scriptCompiler = newScriptCompiler(jpathCacheSize,
		func(_ *api.Step, cfg *api.ScriptConfig) (jpath.Path, error) {
			return env.compile(cfg.Script)
		},
	)
	return env
}

// Validate checks whether the given JPath expression is valid
func (e *JPathEnv) Validate(_ *api.Step, script string) error {
	_, err := e.Compile(nil, &api.ScriptConfig{
		Language: api.ScriptLangJPath,
		Script:   script,
	})
	return err
}

// ExecuteScript is unsupported for JPath language
func (e *JPathEnv) ExecuteScript(
	Compiled, *api.Step, api.Args,
) (api.Args, error) {
	return nil, ErrJPathExecuteScript
}

// EvaluatePredicate applies the compiled JPath expression and treats any
// match as predicate success, including explicit null matches
func (e *JPathEnv) EvaluatePredicate(
	c Compiled, _ *api.Step, inputs api.Args,
) (bool, error) {
	path, ok := c.(jpath.Path)
	if !ok {
		return false, fmt.Errorf("%w, got %T", ErrJPathBadCompiledType, c)
	}

	matches := path(normalizeMappingDoc(inputs))
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
