package engine

import (
	"errors"
	"fmt"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	// ScriptRegistry manages script environments for different languages
	ScriptRegistry struct {
		envs map[string]ScriptEnvironment
	}

	// ScriptEnvironment defines the interface for script execution environments
	ScriptEnvironment interface {
		// Validate checks if a script is syntactically valid
		Validate(step *api.Step, script string) error

		// Compile compiles a script and returns the compiled form
		Compile(
			step *api.Step, script string, argNames []string,
		) (Compiled, error)

		// CompileStepScript compiles a step's main script
		CompileStepScript(step *api.Step) (Compiled, error)

		// CompileStepPredicate compiles a step's predicate
		CompileStepPredicate(step *api.Step) (Compiled, error)

		// ExecuteScript executes a compiled script with the given inputs
		ExecuteScript(
			c Compiled, step *api.Step, inputs api.Args,
		) (api.Args, error)

		// EvaluatePredicate evaluates a compiled predicate with given inputs
		EvaluatePredicate(
			c Compiled, step *api.Step, inputs api.Args,
		) (bool, error)
	}

	// Compiled represents a compiled script for any supported language.
	// Concrete types: data.Procedure (Ale), *CompiledLuaScript (Lua)
	Compiled any
)

const (
	scriptType    = "script"
	predicateType = "predicate"
)

var (
	ErrUnsupportedLanguage = errors.New("unsupported script language")
)

// NewScriptRegistry creates a new script registry with Ale and Lua environments
func NewScriptRegistry() *ScriptRegistry {
	return &ScriptRegistry{
		envs: map[string]ScriptEnvironment{
			api.ScriptLangAle: NewAleEnv(),
			api.ScriptLangLua: NewLuaEnv(),
		},
	}
}

// Get returns the script environment for the given language
func (r *ScriptRegistry) Get(language string) (ScriptEnvironment, error) {
	env, ok := r.envs[language]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedLanguage, language)
	}
	return env, nil
}

// CompilePlan compiles all scripts in the execution plan
func (r *ScriptRegistry) CompilePlan(plan *api.ExecutionPlan) error {
	for _, info := range plan.Steps {
		if err := r.compileStepScript(info); err != nil {
			return err
		}
		if err := r.compileStepPredicate(info); err != nil {
			return err
		}
	}

	return nil
}

func (r *ScriptRegistry) compileStepScript(info *api.StepInfo) error {
	step := info.Step
	if step.Type != api.StepTypeScript || step.Script == nil {
		return nil
	}

	env, err := r.Get(step.Script.Language)
	if err != nil {
		return err
	}
	comp, err := env.CompileStepScript(step)
	if err != nil {
		return err
	}
	info.Script = comp
	return nil
}

func (r *ScriptRegistry) compileStepPredicate(info *api.StepInfo) error {
	step := info.Step
	if step.Predicate == nil {
		return nil
	}

	env, err := r.Get(step.Predicate.Language)
	if err != nil {
		return err
	}
	comp, err := env.CompileStepPredicate(step)
	if err != nil {
		return err
	}
	info.Predicate = comp
	return nil
}
