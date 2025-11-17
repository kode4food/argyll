package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/internal/events"
	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

type cycleDetector struct {
	graph    map[timebox.ID][]timebox.ID
	visited  util.Set[timebox.ID]
	recStack util.Set[timebox.ID]
}

var (
	ErrStepAlreadyExists  = errors.New("step already exists")
	ErrStepDoesNotExist   = errors.New("step does not exist")
	ErrTypeConflict       = errors.New("attribute type conflict")
	ErrCircularDependency = errors.New("circular dependency detected")
)

// RegisterStep registers a new step with the engine after validating its
// configuration and checking for conflicts
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

// UpdateStep updates an existing step registration with new configuration
// after validation
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

// UpdateStepHealth updates the health status of a registered step, used
// primarily for tracking HTTP service availability and script errors
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

		return util.Raise(ag, api.EventTypeStepHealthChanged,
			api.StepHealthChangedEvent{
				StepID: stepID,
				Status: health,
				Error:  errMsg,
			},
		)
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
	return util.Raise(ag, api.EventTypeStepRegistered,
		api.StepRegisteredEvent{Step: step},
	)
}

func (e *Engine) compileScript(ctx context.Context, step *api.Step) {
	if step.Type != api.StepTypeScript || step.Script == nil {
		return
	}

	env, err := e.scripts.Get(step.Script.Language)
	if err != nil {
		e.handleCompileError(
			ctx, step.ID, "Failed to get script environment", err,
		)
		return
	}

	names := step.SortedArgNames()
	_, err = env.Compile(step, step.Script.Script, names)

	if err != nil {
		e.handleCompileError(ctx, step.ID, "Failed to compile script", err)
		return
	}

	_ = e.UpdateStepHealth(ctx, step.ID, api.HealthHealthy, "")
}

func (e *Engine) handleCompileError(
	ctx context.Context, stepID timebox.ID, msg string, err error,
) {
	slog.Warn(msg,
		slog.Any("step_id", stepID),
		slog.Any("error", err))
	_ = e.UpdateStepHealth(ctx, stepID, api.HealthUnhealthy, err.Error())
}

func validateAttributeTypes(st *api.EngineState, newStep *api.Step) error {
	attributeTypes := collectAttributeTypes(st, newStep.ID)
	return checkAttributeConflicts(newStep.Attributes, attributeTypes)
}

func collectAttributeTypes(
	st *api.EngineState, excludeStepID timebox.ID,
) map[api.Name]api.AttributeType {
	attributeTypes := make(map[api.Name]api.AttributeType)
	for stepID, step := range st.Steps {
		if stepID == excludeStepID {
			continue
		}
		for name, attr := range step.Attributes {
			attributeTypes[name] = attr.Type
		}
	}
	return attributeTypes
}

func checkAttributeConflicts(
	attrs api.AttributeSpecs, types map[api.Name]api.AttributeType,
) error {
	for name, attr := range attrs {
		if existingType, ok := types[name]; ok {
			if existingType != attr.Type {
				return fmt.Errorf("%w: %s", ErrTypeConflict, name)
			}
		}
	}
	return nil
}

func detectCircularDependencies(st *api.EngineState, newStep *api.Step) error {
	detector := &cycleDetector{
		graph:    buildDependencyGraph(st, newStep),
		visited:  util.Set[timebox.ID]{},
		recStack: util.Set[timebox.ID]{},
	}

	for stepID := range detector.graph {
		if !detector.visited.Contains(stepID) {
			if cycle := detector.findCycle(stepID, nil); cycle != nil {
				return fmt.Errorf("%w: %v", ErrCircularDependency, cycle)
			}
		}
	}

	return nil
}

func buildDependencyGraph(
	st *api.EngineState, newStep *api.Step,
) map[timebox.ID][]timebox.ID {
	allSteps := stepsIncluding(st, newStep)
	producerIndex := indexAttributeProducers(allSteps)
	return graphFromStepDependencies(allSteps, producerIndex)
}

func stepsIncluding(
	st *api.EngineState, newStep *api.Step,
) map[timebox.ID]*api.Step {
	steps := make(map[timebox.ID]*api.Step, len(st.Steps))
	for id, step := range st.Steps {
		if id != newStep.ID {
			steps[id] = step
		}
	}
	steps[newStep.ID] = newStep
	return steps
}

func indexAttributeProducers(
	steps map[timebox.ID]*api.Step,
) map[api.Name][]timebox.ID {
	index := make(map[api.Name][]timebox.ID)
	for stepID, step := range steps {
		for name, attr := range step.Attributes {
			if attr.IsOutput() {
				index[name] = append(index[name], stepID)
			}
		}
	}
	return index
}

func graphFromStepDependencies(
	steps map[timebox.ID]*api.Step, producerIndex map[api.Name][]timebox.ID,
) map[timebox.ID][]timebox.ID {
	graph := make(map[timebox.ID][]timebox.ID)
	for stepID, step := range steps {
		graph[stepID] = dependenciesFor(step, producerIndex)
	}
	return graph
}

func dependenciesFor(
	step *api.Step, producerIndex map[api.Name][]timebox.ID,
) []timebox.ID {
	var deps []timebox.ID
	for name, attr := range step.Attributes {
		if attr.IsInput() {
			if producers, ok := producerIndex[name]; ok {
				deps = append(deps, producers...)
			}
		}
	}
	return deps
}

func (d *cycleDetector) findCycle(
	stepID timebox.ID, path []timebox.ID,
) []timebox.ID {
	d.visited.Add(stepID)
	d.recStack.Add(stepID)
	path = append(path, stepID)

	for _, depID := range d.graph[stepID] {
		if !d.visited.Contains(depID) {
			if cycle := d.findCycle(depID, path); cycle != nil {
				return cycle
			}
		} else if d.recStack.Contains(depID) {
			return extractCyclePath(path, depID)
		}
	}

	d.recStack.Remove(stepID)
	return nil
}

func extractCyclePath(path []timebox.ID, cycleNode timebox.ID) []timebox.ID {
	for i, id := range path {
		if id == cycleNode {
			return path[i:]
		}
	}
	return nil
}
