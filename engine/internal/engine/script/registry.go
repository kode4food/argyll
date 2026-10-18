package script

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/kode4food/jpath"
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

		// EvaluateScript executes a compiled script with the given inputs
		EvaluateScript(Compiled, *api.Step, api.Args) (api.Args, error)

		// EvaluatePredicate evaluates a compiled predicate with given inputs
		EvaluatePredicate(Compiled, *api.Step, api.Args) (bool, error)

		// EvaluateMatch evaluates a compiled matcher with a single input
		EvaluateMatch(c Compiled, input any) (bool, error)
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
		*CompiledLua | jpath.Path
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

	matchValue = api.Name("value")
)

var (
	ErrInvalidCompiledScript = errors.New("invalid compiled script")
)

// MatchStep returns the minimal synthetic step used as the compilation context
// for a required match predicate, exposing only the candidate value
var MatchStep = &api.Step{
	Attributes: api.AttributeSpecs{
		matchValue: {Role: api.RoleRequired, Type: api.TypeAny},
	},
}

// NewRegistry creates a new registry with JPath and Lua environments
func NewRegistry() *Registry {
	return &Registry{
		envs: map[string]Environment{
			api.ScriptLangJPath: NewJPathEnv(),
			api.ScriptLangLua:   NewLuaEnv(),
		},
	}
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
// compile successfully against the registry
func (r *Registry) ValidateStep(st *api.Step) error {
	if st.Predicate != nil {
		if _, err := r.Compile(st, st.Predicate); err != nil {
			return err
		}
	}
	for name, attr := range st.Attributes {
		if attr.Required == nil || attr.Required.Match == nil {
			continue
		}
		if _, err := r.Compile(MatchStep, attr.Required.Match); err != nil {
			return fmt.Errorf("%w for attribute %q: %w",
				api.ErrInvalidScriptLanguage, name, err)
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
			return compiled, ErrInvalidCompiledScript
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
