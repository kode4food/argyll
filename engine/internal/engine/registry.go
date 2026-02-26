package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	stepSet = util.Set[api.StepID]

	upsertRaise func(*api.Step, *CatalogAggregator) error
	upsertCheck func(oldStep, newStep *api.Step, exists bool) (bool, error)
)

var (
	ErrInvalidStep        = errors.New("invalid step")
	ErrTypeConflict       = errors.New("attribute type conflict")
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrLangNotValid       = errors.New("language not valid in this context")
)

// RegisterStep registers a new step with the engine after validating its
// configuration and checking for conflicts
func (e *Engine) RegisterStep(step *api.Step) error {
	return e.execStepUpsert(step, e.raiseStepRegisteredEvent,
		func(oldStep, newStep *api.Step, exists bool) (bool, error) {
			if !exists {
				return false, nil
			}
			if oldStep.Equal(newStep) {
				return true, nil
			}
			return false, fmt.Errorf("%w: %s", ErrStepExists, newStep.ID)
		},
	)
}

// UpdateStep updates an existing step registration with new configuration
// after validation
func (e *Engine) UpdateStep(step *api.Step) error {
	return e.execStepUpsert(step, e.raiseStepUpdatedEvent,
		func(oldStep, newStep *api.Step, exists bool) (bool, error) {
			if !exists {
				return false, fmt.Errorf("%w: %s", ErrStepNotFound, newStep.ID)
			}
			if oldStep.Equal(newStep) {
				return true, nil
			}
			return false, nil
		},
	)
}

// UpdateStepHealth updates the health status of a registered step, used
// primarily for tracking HTTP service availability and script errors
func (e *Engine) UpdateStepHealth(
	stepID api.StepID, health api.HealthStatus, errMsg string,
) error {
	cmd := func(st *api.PartitionState, ag *PartitionAggregator) error {
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

	_, err := e.execPartition(cmd)
	return err
}

func (e *Engine) execStepUpsert(
	step *api.Step, raise upsertRaise, check upsertCheck,
) error {
	step = step.WithWorkDefaults(&e.config.Work)
	if err := e.validateStep(step); err != nil {
		return err
	}

	cmd := func(st *api.CatalogState, ag *CatalogAggregator) error {
		old, ok := st.Steps[step.ID]
		if noop, err := check(old, step, ok); noop || err != nil {
			return err
		}

		if err := performCalls(
			withArgs(validateAttributeTypes, st, step),
			withArgs(detectStepCycles, st, step),
		); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidStep, err)
		}
		return raise(step, ag)
	}

	if _, err := e.execCatalog(cmd); err != nil {
		return err
	}
	if stepHasScripts(step) {
		err := e.UpdateStepHealth(step.ID, api.HealthHealthy, "")
		if err != nil {
			slog.Error("Failed to update step health",
				log.StepID(step.ID),
				log.Error(err))
		}
	}
	return nil
}

func (e *Engine) validateStep(step *api.Step) error {
	if err := performCalls(
		step.Validate,
		withArg(e.validateStepMappings, step),
		withArg(e.validateStepScripts, step),
	); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidStep, err)
	}
	return nil
}

func (e *Engine) validateStepMappings(step *api.Step) error {
	for name, attr := range step.Attributes {
		if attr.Mapping == nil || attr.Mapping.Script == nil {
			continue
		}

		if _, err := e.mapper.Compile(step, attr.Mapping.Script); err != nil {
			return fmt.Errorf("%w for attribute %q: %v",
				api.ErrInvalidAttributeMapping, name, err,
			)
		}
	}
	return nil
}

func (e *Engine) validateStepScripts(step *api.Step) error {
	if step.Type == api.StepTypeScript && step.Script != nil {
		if step.Script.Language == api.ScriptLangJPath {
			return fmt.Errorf("%w: %s", ErrLangNotValid, step.Script.Language)
		}

		env, err := e.scripts.Get(step.Script.Language)
		if err != nil {
			return err
		}

		if err := env.Validate(step, step.Script.Script); err != nil {
			return err
		}
	}

	if step.Predicate != nil {
		env, err := e.scripts.Get(step.Predicate.Language)
		if err != nil {
			return err
		}

		if err := env.Validate(step, step.Predicate.Script); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) raiseStepRegisteredEvent(
	step *api.Step, ag *CatalogAggregator,
) error {
	if err := events.Raise(ag, api.EventTypeStepRegistered,
		api.StepRegisteredEvent{Step: step},
	); err != nil {
		return err
	}
	ag.OnSuccess(func(*api.CatalogState) {
		if err := e.raisePartitionEvent(api.EventTypeStepHealthChanged,
			api.StepHealthChangedEvent{
				StepID: step.ID,
				Status: api.HealthUnknown,
			},
		); err != nil {
			slog.Error("Failed to raise step health changed event",
				log.StepID(step.ID),
				log.Error(err))
		}
	})
	return nil
}

func (e *Engine) raiseStepUpdatedEvent(
	step *api.Step, ag *CatalogAggregator,
) error {
	return events.Raise(ag, api.EventTypeStepUpdated,
		api.StepUpdatedEvent{Step: step},
	)
}

func stepHasScripts(step *api.Step) bool {
	return (step.Type == api.StepTypeScript && step.Script != nil) ||
		step.Predicate != nil
}

func validateAttributeTypes(st *api.CatalogState, newStep *api.Step) error {
	attributeTypes := collectAttributeTypes(st, newStep.ID)
	return checkAttributeConflicts(newStep.Attributes, attributeTypes)
}

func collectAttributeTypes(
	st *api.CatalogState, excludeID api.StepID,
) api.AttributeTypes {
	attributeTypes := make(api.AttributeTypes)
	for stepID, step := range st.Steps {
		if stepID == excludeID {
			continue
		}
		for name, attr := range step.Attributes {
			attributeTypes[name] = attr.Type
		}
	}
	return attributeTypes
}

func checkAttributeConflicts(
	attrs api.AttributeSpecs, types api.AttributeTypes,
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

func detectStepCycles(st *api.CatalogState, newStep *api.Step) error {
	steps := stepsIncluding(st, newStep)
	deps := st.Attributes
	if existing, ok := st.Steps[newStep.ID]; ok {
		deps = deps.RemoveStep(existing)
	}
	deps = deps.AddStep(newStep)
	return checkCycleFromStep(newStep.ID, deps, steps, stepSet{})
}

func checkCycleFromStep(
	currentID api.StepID, deps api.AttributeGraph, steps api.Steps,
	stack stepSet,
) error {
	if stack.Contains(currentID) {
		return fmt.Errorf("%w: step %s", ErrCircularDependency, currentID)
	}

	stack.Add(currentID)
	defer stack.Remove(currentID)

	step := steps[currentID]
	for name, attr := range step.Attributes {
		if attr.IsInput() {
			if depInfo := deps[name]; depInfo != nil {
				for _, providerID := range depInfo.Providers {
					if err := checkCycleFromStep(
						providerID, deps, steps, stack,
					); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func stepsIncluding(st *api.CatalogState, newStep *api.Step) api.Steps {
	steps := maps.Clone(st.Steps)
	steps[newStep.ID] = newStep
	return steps
}
