package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kode4food/spuds/engine/internal/events"
	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
)

type cycleDetector struct {
	graph    map[api.StepID][]api.StepID
	visited  util.Set[api.StepID]
	recStack util.Set[api.StepID]
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
	ctx context.Context, stepID api.StepID, health api.HealthStatus,
	errMsg string,
) error {
	cmd := func(st *api.EngineState, ag *Aggregator) error {
		if stepHealth, ok := st.Health[stepID]; ok {
			if stepHealth.Status == health && stepHealth.Error == errMsg {
				return nil
			}
		}

		return events.Raise(ag, api.EventTypeStepHealthChanged,
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
	return events.Raise(ag, api.EventTypeStepRegistered,
		api.StepRegisteredEvent{Step: step},
	)
}

func (e *Engine) compileScript(ctx context.Context, step *api.Step) {
	if step.Type != api.StepTypeScript || step.Script == nil {
		return
	}

	_, err := e.scripts.Compile(step, step.Script)
	if err != nil {
		e.handleCompileError(ctx, step.ID, "Failed to compile script", err)
		return
	}

	_ = e.UpdateStepHealth(ctx, step.ID, api.HealthHealthy, "")
}

func (e *Engine) handleCompileError(
	ctx context.Context, stepID api.StepID, msg string, err error,
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
	st *api.EngineState, excludeStepID api.StepID,
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
		visited:  util.Set[api.StepID]{},
		recStack: util.Set[api.StepID]{},
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
) map[api.StepID][]api.StepID {
	allSteps := stepsIncluding(st, newStep)
	producerIndex := indexAttributeProducers(allSteps)
	return graphFromStepDependencies(allSteps, producerIndex)
}

func stepsIncluding(
	st *api.EngineState, newStep *api.Step,
) map[api.StepID]*api.Step {
	steps := make(map[api.StepID]*api.Step, len(st.Steps))
	for id, step := range st.Steps {
		if id != newStep.ID {
			steps[id] = step
		}
	}
	steps[newStep.ID] = newStep
	return steps
}

func indexAttributeProducers(
	steps map[api.StepID]*api.Step,
) map[api.Name][]api.StepID {
	index := make(map[api.Name][]api.StepID)
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
	steps map[api.StepID]*api.Step, producerIndex map[api.Name][]api.StepID,
) map[api.StepID][]api.StepID {
	graph := make(map[api.StepID][]api.StepID)
	for stepID, step := range steps {
		graph[stepID] = dependenciesFor(step, producerIndex)
	}
	return graph
}

func dependenciesFor(
	step *api.Step, producerIndex map[api.Name][]api.StepID,
) []api.StepID {
	var deps []api.StepID
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
	stepID api.StepID, path []api.StepID,
) []api.StepID {
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

func extractCyclePath(path []api.StepID, cycleNode api.StepID) []api.StepID {
	for i, id := range path {
		if id == cycleNode {
			return path[i:]
		}
	}
	return nil
}
