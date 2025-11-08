package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/internal/events"
	"github.com/kode4food/spuds/engine/pkg/api"
)

var (
	ErrInvalidScript     = errors.New("invalid script")
	ErrStepAlreadyExists = errors.New("step already exists")
	ErrStepDoesNotExist  = errors.New("step does not exist")
)

func (e *Engine) RegisterStep(ctx context.Context, step *api.Step) error {
	if err := e.validateScriptStep(step); err != nil {
		return err
	}

	cmd := func(st *api.EngineState, ag *Aggregator) error {
		if existing, ok := st.Steps[step.ID]; ok {
			if existing.Equal(step) {
				return nil
			}
			return fmt.Errorf("%w: %s", ErrStepAlreadyExists, step.ID)
		}
		return e.raiseStepRegisteredEvent(step, ag)
	}

	if _, err := e.engineExec.Exec(ctx, events.EngineID, cmd); err != nil {
		return err
	}

	if step.Type == api.StepTypeScript {
		e.compileScript(ctx, step)
	}
	return nil
}

func (e *Engine) UpdateStep(ctx context.Context, step *api.Step) error {
	if err := e.validateScriptStep(step); err != nil {
		return err
	}

	cmd := func(st *api.EngineState, ag *Aggregator) error {
		if _, ok := st.Steps[step.ID]; !ok {
			return fmt.Errorf("%w: %s", ErrStepDoesNotExist, step.ID)
		}
		return e.raiseStepRegisteredEvent(step, ag)
	}

	if _, err := e.engineExec.Exec(ctx, events.EngineID, cmd); err != nil {
		return err
	}

	if step.Type == api.StepTypeScript {
		e.compileScript(ctx, step)
	}
	return nil
}

func (e *Engine) UpdateStepHealth(
	ctx context.Context, stepID timebox.ID, health api.HealthStatus,
	errMsg string,
) error {
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		if stepHealth, ok := st.Health[stepID]; ok {
			if stepHealth.Status == health && stepHealth.Error == errMsg {
				return nil
			}
		}

		data, err := json.Marshal(api.StepHealthChangedEvent{
			StepID:      stepID,
			Health:      health,
			HealthError: errMsg,
		})
		if err != nil {
			return err
		}
		ag.Raise(api.EventTypeStepHealthChanged, data)
		return nil
	}

	_, err := e.engineExec.Exec(ctx, events.EngineID, cmd)
	return err
}

func (e *Engine) validateScriptStep(step *api.Step) error {
	if step.Type != api.StepTypeScript || step.Script == nil {
		return nil
	}

	env, err := e.scripts.Get(step.Script.Language)
	if err != nil {
		return err
	}

	if err := env.Validate(step, step.Script.Script); err != nil {
		return fmt.Errorf("%s: %w", ErrInvalidScript, err)
	}
	return nil
}

func (e *Engine) raiseStepRegisteredEvent(
	step *api.Step, ag *Aggregator,
) error {
	data, err := json.Marshal(api.StepRegisteredEvent{Step: step})
	if err != nil {
		return err
	}
	ag.Raise(api.EventTypeStepRegistered, data)
	return nil
}

func (e *Engine) compileScript(
	ctx context.Context, step *api.Step,
) {
	if step.Type != api.StepTypeScript || step.Script == nil {
		return
	}

	env, err := e.scripts.Get(step.Script.Language)
	if err != nil {
		slog.Warn("Failed to get script environment",
			slog.Any("step_id", step.ID),
			slog.Any("error", err))
		_ = e.UpdateStepHealth(
			ctx, step.ID, api.HealthUnhealthy, err.Error(),
		)
		return
	}

	names := step.SortedArgNames()
	_, err = env.Compile(step, step.Script.Script, names)

	if err != nil {
		slog.Warn("Failed to compile script",
			slog.Any("step_id", step.ID),
			slog.Any("error", err))
		_ = e.UpdateStepHealth(
			ctx, step.ID, api.HealthUnhealthy, err.Error(),
		)
		return
	}

	_ = e.UpdateStepHealth(ctx, step.ID, api.HealthHealthy, "")
}
