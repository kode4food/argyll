package builtins

import (
	"errors"
	"fmt"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/step"
)

var (
	ErrLangNotValid        = errors.New("invalid language")
	ErrScriptCompileFailed = errors.New("script compile failed")
)

// Script returns the standard Lua script handler
func Script() *step.Handler {
	scripts := script.NewRegistry()
	return &step.Handler{
		Validate: func(st *api.Step) error {
			if st.Script == nil {
				return api.ErrScriptRequired
			}
			if st.Script.Language == api.ScriptLangJPath {
				return fmt.Errorf("%w: %s", ErrLangNotValid, st.Script.Language)
			}
			_, err := scripts.Compile(st, st.Script)
			return err
		},
		Health: func(st *api.Step) api.HealthState {
			if _, err := scripts.Compile(st, st.Script); err != nil {
				return api.HealthState{
					Status: api.HealthUnhealthy,
					Error:  err.Error(),
				}
			}
			return api.HealthState{Status: api.HealthHealthy}
		},
		Execute: func(
			rt step.Runtime, st *api.Step, inputs api.Args, token api.Token,
		) error {
			compiled, err := scripts.Compile(st, st.Script)
			if err != nil {
				return errors.Join(
					ErrScriptCompileFailed, err,
					rt.UpdateHealth(api.HealthUnhealthy, err.Error()),
				)
			}
			inputs = applyMetaInputs(st, inputs, metadata(rt, token))
			env, err := scripts.Get(st.Script.Language)
			if err != nil {
				return err
			}
			outputs, err := env.ExecuteScript(compiled, st, inputs)
			if err != nil {
				return errors.Join(
					err, rt.UpdateHealth(api.HealthUnhealthy, err.Error()),
				)
			}
			if err := rt.CompleteWork(token, outputs); err != nil {
				return err
			}
			return rt.UpdateHealth(api.HealthHealthy, "")
		},
	}
}
