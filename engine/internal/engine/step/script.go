package step

import (
	"errors"
	"fmt"

	"github.com/kode4food/argyll/engine/internal/engine/script"
	"github.com/kode4food/argyll/engine/pkg/api"
)

var (
	ErrLangNotValid        = errors.New("language not valid in this context")
	ErrScriptCompileFailed = errors.New("failed to compile scripts for step")
)

func scriptHandler(scripts *script.Registry) *Handler {
	return &Handler{
		Validate: scriptConfigValidator(scripts),
		Execute:  scriptExecutor(scripts),
		Health:   scriptHealth(scripts),
	}
}

func scriptConfigValidator(scripts *script.Registry) ValidateFunc {
	return func(st *api.Step) error {
		if st.Script == nil {
			return api.ErrScriptRequired
		}
		if st.Script.Language == api.ScriptLangJPath {
			return fmt.Errorf("%w: %s", ErrLangNotValid, st.Script.Language)
		}
		if _, err := scripts.Compile(st, st.Script); err != nil {
			return err
		}
		return nil
	}
}

func scriptHealth(scripts *script.Registry) HealthFunc {
	return func(st *api.Step) api.HealthState {
		if _, err := scripts.Compile(st, st.Script); err != nil {
			return api.HealthState{
				Status: api.HealthUnhealthy,
				Error:  err.Error(),
			}
		}
		return api.HealthState{Status: api.HealthHealthy}
	}
}

func scriptExecutor(scripts *script.Registry) ExecuteFunc {
	return func(
		rt Runtime, st *api.Step, inputs api.Args, tkn api.Token,
	) error {
		c, err := scripts.Compile(st, st.Script)
		if err != nil {
			return errors.Join(
				ErrScriptCompileFailed, err, markScriptUnhealthy(rt, err),
			)
		}

		inputs = applyMetaInputs(st, inputs, httpMetaForToken(rt, tkn))
		outputs, err := executeScript(scripts, st, c, inputs)
		if err != nil {
			return errors.Join(err, markScriptUnhealthy(rt, err))
		}

		if err := rt.CompleteWork(tkn, outputs); err != nil {
			return err
		}
		return rt.UpdateHealth(api.HealthHealthy, "")
	}
}

func markScriptUnhealthy(rt Runtime, err error) error {
	return rt.UpdateHealth(api.HealthUnhealthy, err.Error())
}

func executeScript(
	scripts *script.Registry, st *api.Step, c script.Compiled, inputs api.Args,
) (api.Args, error) {
	language := api.ScriptLangAle
	if st.Script != nil {
		language = st.Script.Language
	}
	env, err := scripts.Get(language)
	if err != nil {
		return nil, err
	}
	return env.ExecuteScript(c, st, inputs)
}
