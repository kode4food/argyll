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
	ErrStepAlreadyExists  = errors.New("step already exists")
	ErrStepDoesNotExist   = errors.New("step does not exist")
	ErrTypeConflict       = errors.New("attribute type conflict")
	ErrCircularDependency = errors.New("circular dependency detected")
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
		if err := validateAttributeTypes(st, step); err != nil {
			return err
		}
		if err := detectCircularDependencies(st, step); err != nil {
			return err
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
		if err := validateAttributeTypes(st, step); err != nil {
			return err
		}
		if err := detectCircularDependencies(st, step); err != nil {
			return err
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
			StepID: stepID,
			Status: health,
			Error:  errMsg,
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
		return err
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

func (e *Engine) compileScript(ctx context.Context, step *api.Step) {
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

func validateAttributeTypes(st *api.EngineState, newStep *api.Step) error {
	attributeTypes := make(map[api.Name]api.AttributeType)

	for stepID, step := range st.Steps {
		if stepID == newStep.ID {
			continue
		}
		for name, attr := range step.Attributes {
			if existingType, exists := attributeTypes[name]; exists {
				if existingType != attr.Type {
					return fmt.Errorf("%w: %s", ErrTypeConflict, name)
				}
			} else {
				attributeTypes[name] = attr.Type
			}
		}
	}

	for name, attr := range newStep.Attributes {
		if existingType, exists := attributeTypes[name]; exists {
			if existingType != attr.Type {
				return fmt.Errorf("%w: %s", ErrTypeConflict, name)
			}
		}
	}

	return nil
}

func detectCircularDependencies(st *api.EngineState, newStep *api.Step) error {
	graph := buildDependencyGraph(st, newStep)
	visited := make(map[timebox.ID]bool)
	recStack := make(map[timebox.ID]bool)

	for stepID := range graph {
		if !visited[stepID] {
			if cycle := findCycle(
				stepID, graph, visited, recStack, nil,
			); cycle != nil {
				return fmt.Errorf("%w: %v", ErrCircularDependency, cycle)
			}
		}
	}

	return nil
}

func buildDependencyGraph(
	st *api.EngineState, newStep *api.Step,
) map[timebox.ID][]timebox.ID {
	attrProducers := make(map[api.Name][]timebox.ID)
	graph := make(map[timebox.ID][]timebox.ID)

	allSteps := make(map[timebox.ID]*api.Step)
	for id, step := range st.Steps {
		if id != newStep.ID {
			allSteps[id] = step
		}
	}
	allSteps[newStep.ID] = newStep

	for stepID, step := range allSteps {
		graph[stepID] = []timebox.ID{}
		for name, attr := range step.Attributes {
			if attr.Role == api.RoleOutput {
				attrProducers[name] = append(attrProducers[name], stepID)
			}
		}
	}

	for stepID, step := range allSteps {
		for name, attr := range step.Attributes {
			if attr.Role == api.RoleRequired || attr.Role == api.RoleOptional {
				if producers, ok := attrProducers[name]; ok {
					graph[stepID] = append(graph[stepID], producers...)
				}
			}
		}
	}

	return graph
}

func findCycle(
	stepID timebox.ID, graph map[timebox.ID][]timebox.ID,
	visited, recStack map[timebox.ID]bool, path []timebox.ID,
) []timebox.ID {
	visited[stepID] = true
	recStack[stepID] = true
	path = append(path, stepID)

	for _, depID := range graph[stepID] {
		if !visited[depID] {
			if cycle := findCycle(
				depID, graph, visited, recStack, path,
			); cycle != nil {
				return cycle
			}
		} else if recStack[depID] {
			cycleStart := -1
			for i, id := range path {
				if id == depID {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				return path[cycleStart:]
			}
		}
	}

	recStack[stepID] = false
	return nil
}
