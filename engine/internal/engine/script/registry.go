package script

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/kode4food/argex"
	"github.com/kode4food/lru"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// Registry manages script environments for different languages
	Registry struct {
		envs map[string]Environment
	}

	// Environment defines the interface for script environments
	Environment interface {
		// Validate checks if a script is syntactically valid
		Validate(st *api.Step, script string) error

		// Compile compiles a script and returns the compiled form
		Compile(*api.Step, *api.ScriptConfig) (Compiled, error)

		// Evaluate executes a compiled script with the given inputs and
		// returns its JSON-shaped result
		Evaluate(Compiled, *api.Step, api.Args) (any, error)
	}

	// Compiled represents a compiled script for any supported language
	Compiled any

	compiler[T compiledValue] struct {
		cache *lru.Cache[T]
		build compileFunc[T]
	}

	compileFunc[T compiledValue] func(
		st *api.Step, cfg *api.ScriptConfig,
	) (T, error)

	compiledValue interface {
		*CompiledLua | argex.Program
	}

	predicateChecker interface {
		CheckPredicate(*api.ScriptConfig) error
	}
)

const (
	// Selector document keys

	MatchTags                 = "tags"
	MatchType                 = "type"
	MatchHandling             = "handling"
	MatchAttributes           = "attributes"
	MatchAttributeRole        = "role"
	MatchAttributeCompensated = "compensated"

	// MatchValue names the candidate passed to a match script
	MatchValue = api.Name("value")

	scriptResult = api.Name("result")
)

var (
	ErrInvalidScript = errors.New("invalid script")
	ErrNotBoolean    = errors.New("must return boolean")
)

// MatchStep returns the minimal synthetic step used as the compilation context
// for a required match predicate, exposing only the candidate value
var MatchStep = &api.Step{
	Attributes: api.AttributeSpecs{
		MatchValue: {Role: api.RoleRequired, Type: api.TypeAny},
	},
}

// NewRegistry creates a registry with Argyll Expressions and Lua environments
func NewRegistry() *Registry {
	return &Registry{
		envs: map[string]Environment{
			api.ScriptLangArgyll: NewArgExEnv(),
			api.ScriptLangLua:    NewLuaEnv(),
		},
	}
}

// Bool returns a predicate, match, or selector result as a Boolean
func Bool(value any) (bool, error) {
	if res, ok := value.(bool); ok {
		return res, nil
	}
	return false, ErrNotBoolean
}

// Outputs returns an object result as the output arguments of a Script Step,
// and any other result as the output named result
func Outputs(value any) api.Args {
	obj, ok := value.(map[string]any)
	if !ok {
		return api.Args{scriptResult: value}
	}
	outputs := make(api.Args, len(obj))
	for key, elem := range obj {
		outputs[api.Name(key)] = elem
	}
	return outputs
}

func (r *Registry) Register(language string, env Environment) {
	r.envs[language] = env
}

// Get returns the script environment for the given language
func (r *Registry) Get(language string) (Environment, error) {
	env, ok := r.envs[language]
	if !ok {
		return nil, fmt.Errorf("%w: %s", api.ErrInvalidScriptLanguage, language)
	}
	return env, nil
}

// ValidateStep checks that a step's predicate and attribute match scripts
// compile successfully against the registry and can return a Boolean
func (r *Registry) ValidateStep(st *api.Step) error {
	if st.Predicate != nil {
		if err := r.ValidatePredicate(st, st.Predicate); err != nil {
			return fmt.Errorf("predicate: %w", err)
		}
	}
	for name, attr := range st.Attributes {
		cfg := attr.Match()
		if cfg == nil {
			continue
		}
		if err := r.ValidatePredicate(MatchStep, cfg); err != nil {
			return fmt.Errorf("attribute %q match: %w", name, err)
		}
	}
	return nil
}

// Compile compiles a script config
func (r *Registry) Compile(
	st *api.Step, cfg *api.ScriptConfig,
) (Compiled, error) {
	if cfg == nil {
		return nil, nil
	}
	env, err := r.Get(cfg.Language)
	if err != nil {
		return nil, err
	}
	return env.Compile(st, cfg)
}

// ValidatePredicate compiles a predicate or matcher script and checks that it
// can return a Boolean
func (r *Registry) ValidatePredicate(
	st *api.Step, cfg *api.ScriptConfig,
) error {
	if _, err := r.Compile(st, cfg); err != nil {
		return err
	}
	if pc, ok := r.envs[cfg.Language].(predicateChecker); ok {
		return pc.CheckPredicate(cfg)
	}
	return nil
}

func (c *compiler[T]) Validate(st *api.Step, script string) error {
	_, err := c.Compile(st, &api.ScriptConfig{Script: script})
	return err
}

func (c *compiler[T]) Compile(
	st *api.Step, cfg *api.ScriptConfig,
) (Compiled, error) {
	if cfg == nil {
		return nil, nil
	}
	if cfg.Script == "" {
		return nil, api.ErrScriptEmpty
	}

	return c.cache.Get(hashScript(st, cfg.Script), func() (T, error) {
		compiled, err := c.build(st, cfg)
		if err != nil {
			return compiled, err
		}
		if compiled == nil {
			return compiled, ErrInvalidScript
		}
		return compiled, nil
	})
}

func newCompiler[T compiledValue](size int, build compileFunc[T]) *compiler[T] {
	return &compiler[T]{
		cache: lru.NewCache[T](size),
		build: build,
	}
}

func hashScript(st *api.Step, script string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(script))

	if st != nil {
		for _, arg := range st.SortedArgNames() {
			_, _ = h.Write([]byte{0})
			_, _ = h.Write([]byte(arg))
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}
