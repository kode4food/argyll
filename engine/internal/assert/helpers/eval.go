package helpers

import (
	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

// Matcher returns a match evaluator backed by a new local script registry for
// use in tests that don't have an engine instance
func Matcher() api.Matcher {
	scripts := script.NewRegistry()
	return func(cfg *api.ScriptConfig, value any) (bool, error) {
		comp, err := scripts.Compile(script.MatchStep, cfg)
		if err != nil {
			return false, err
		}
		env, err := scripts.Get(cfg.Language)
		if err != nil {
			return false, err
		}
		res, err := env.Evaluate(
			comp, script.MatchStep, api.Args{script.MatchValue: value},
		)
		if err != nil {
			return false, err
		}
		return script.Bool(res)
	}
}
