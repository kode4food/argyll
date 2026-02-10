package engine

import (
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// ScriptRegistry manages script environments for different languages
	ScriptRegistry struct {
		envs map[string]ScriptEnvironment
	}

	// ScriptEnvironment defines the interface for script environments
	ScriptEnvironment interface {
		// Validate checks if a script is syntactically valid
		Validate(step *api.Step, script string) error

		// Compile compiles a script and returns the compiled form
		Compile(step *api.Step, cfg *api.ScriptConfig) (Compiled, error)

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

var (
	ErrUnsupportedLanguage = api.ErrInvalidScriptLanguage
)

// NewScriptRegistry creates a new script registry with Ale and Lua execution
// environments
func NewScriptRegistry() *ScriptRegistry {
	return &ScriptRegistry{
		envs: map[string]ScriptEnvironment{
			api.ScriptLangAle: NewAleEnv(),
			api.ScriptLangLua: NewLuaEnv(),
		},
	}
}

func (r *ScriptRegistry) Register(language string, env ScriptEnvironment) {
	r.envs[language] = env
}

// Get returns the script environment for the given language
func (r *ScriptRegistry) Get(language string) (ScriptEnvironment, error) {
	env, ok := r.envs[language]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedLanguage, language)
	}
	return env, nil
}

// Compile compiles a script config
func (r *ScriptRegistry) Compile(
	step *api.Step, cfg *api.ScriptConfig,
) (Compiled, error) {
	if cfg == nil {
		return nil, nil
	}
	env, err := r.Get(cfg.Language)
	if err != nil {
		return nil, err
	}
	return env.Compile(step, cfg)
}

// GetCompiledPredicate retrieves the compiled predicate for a flow step
func (e *Engine) GetCompiledPredicate(fs api.FlowStep) (any, error) {
	step, err := e.getStepFromPlan(fs)
	if err != nil {
		return nil, err
	}
	return e.scripts.Compile(step, step.Predicate)
}

// GetCompiledScript retrieves the compiled script for a step in a flow
func (e *Engine) GetCompiledScript(fs api.FlowStep) (any, error) {
	step, err := e.getStepFromPlan(fs)
	if err != nil {
		return nil, err
	}
	return e.scripts.Compile(step, step.Script)
}

func (e *Engine) getStepFromPlan(fs api.FlowStep) (*api.Step, error) {
	flow, err := e.GetFlowState(fs.FlowID)
	if err != nil {
		return nil, err
	}

	if step, ok := flow.Plan.Steps[fs.StepID]; ok {
		return step, nil
	}
	return nil, ErrStepNotInPlan
}
