package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

type (
	CatalogTx struct {
		e  *Engine
		ag *CatalogAggregator
	}

	stepSet = util.Set[api.StepID]
)

var (
	ErrInvalidStep        = errors.New("invalid step")
	ErrStepExists         = errors.New("step exists")
	ErrStepNotFound       = errors.New("step not found")
	ErrTypeConflict       = errors.New("attribute type conflict")
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrLangNotValid       = errors.New("language not valid in this context")
)

// UnregisterStep removes a step from the engine registry
func (e *Engine) UnregisterStep(stepID api.StepID) error {
	return e.CatalogTx(func(tx *CatalogTx) error {
		return tx.Remove(stepID)
	})
}

// RegisterStep registers a new step with the engine after validating its
// configuration and checking for conflicts
func (e *Engine) RegisterStep(step *api.Step) error {
	return e.CatalogTx(func(tx *CatalogTx) error {
		return tx.Register(step)
	})
}

// UpdateStep updates an existing step registration with new configuration
// after validation
func (e *Engine) UpdateStep(step *api.Step) error {
	return e.CatalogTx(func(tx *CatalogTx) error {
		return tx.Update(step)
	})
}

func (e *Engine) CatalogTx(fn func(*CatalogTx) error) error {
	_, err := e.execCatalog(
		func(_ api.CatalogState, ag *CatalogAggregator) error {
			return fn(&CatalogTx{
				e:  e,
				ag: ag,
			})
		},
	)
	return err
}

func (e *Engine) initializeScriptHealth(step *api.Step) {
	status := api.HealthHealthy
	errMsg := ""
	if err := e.VerifyScript(step); err != nil {
		status = api.HealthUnhealthy
		errMsg = err.Error()
	}
	if err := e.UpdateStepHealth(step.ID, status, errMsg); err != nil {
		slog.Error("Failed to update script health",
			log.StepID(step.ID),
			log.Error(err))
	}
}

func (e *Engine) validateStep(step *api.Step) error {
	if err := call.Perform(
		step.Validate,
		call.WithArg(e.validateStepMappings, step),
		call.WithArg(e.validateStepScripts, step),
	); err != nil {
		return errors.Join(ErrInvalidStep, err)
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
	ag.OnSuccess(func(api.CatalogState, []*timebox.Event) {
		e.resetStepHealth(step)
	})
	return nil
}

func (e *Engine) raiseStepUpdatedEvent(
	step *api.Step, ag *CatalogAggregator,
) error {
	if err := events.Raise(ag, api.EventTypeStepUpdated,
		api.StepUpdatedEvent{Step: step},
	); err != nil {
		return err
	}
	ag.OnSuccess(func(api.CatalogState, []*timebox.Event) {
		e.resetStepHealth(step)
	})
	return nil
}

func (e *Engine) resetStepHealth(step *api.Step) {
	err := e.UpdateStepHealth(step.ID, api.HealthUnknown, "")
	if err != nil {
		slog.Error("Failed to update step health",
			log.StepID(step.ID),
			log.Error(err))
	}
	if step.Type == api.StepTypeScript && step.Script != nil {
		e.initializeScriptHealth(step)
	}
}

func (tx *CatalogTx) Register(step *api.Step) error {
	step, err := tx.prepareStep(step)
	if err != nil {
		return err
	}
	st := tx.ag.Value()
	if old, ok := st.Steps[step.ID]; ok {
		if old.Equal(step) {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrStepExists, step.ID)
	}
	if err := validateStepUpsert(st, step); err != nil {
		return err
	}
	return tx.e.raiseStepRegisteredEvent(step, tx.ag)
}

func (tx *CatalogTx) Update(step *api.Step) error {
	step, err := tx.prepareStep(step)
	if err != nil {
		return err
	}
	st := tx.ag.Value()
	old, ok := st.Steps[step.ID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrStepNotFound, step.ID)
	}
	if old.Equal(step) {
		return nil
	}
	if err := validateStepUpsert(st, step); err != nil {
		return err
	}
	return tx.e.raiseStepUpdatedEvent(step, tx.ag)
}

func (tx *CatalogTx) Remove(stepID api.StepID) error {
	return events.Raise(tx.ag, api.EventTypeStepUnregistered,
		api.StepUnregisteredEvent{StepID: stepID},
	)
}

func (tx *CatalogTx) prepareStep(step *api.Step) (*api.Step, error) {
	step = step.WithWorkDefaults(&tx.e.config.Work)
	if err := tx.e.validateStep(step); err != nil {
		return nil, err
	}
	return step, nil
}

func validateStepUpsert(st api.CatalogState, step *api.Step) error {
	if err := call.Perform(
		call.WithArgs(validateAttributeTypes, st, step),
		call.WithArgs(detectStepCycles, st, step),
	); err != nil {
		return errors.Join(ErrInvalidStep, err)
	}
	return nil
}

func validateAttributeTypes(st api.CatalogState, newStep *api.Step) error {
	attributeTypes := collectAttributeTypes(st, newStep.ID)
	return checkAttributeConflicts(newStep.Attributes, attributeTypes)
}

func collectAttributeTypes(
	st api.CatalogState, excludeID api.StepID,
) api.AttributeTypes {
	attributeTypes := make(api.AttributeTypes)
	for sid, st := range st.Steps {
		if sid == excludeID {
			continue
		}
		for name, attr := range st.Attributes {
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

func detectStepCycles(st api.CatalogState, newStep *api.Step) error {
	if err := detectAttributeCycles(st, newStep); err != nil {
		return err
	}
	return detectFlowCycles(st, newStep)
}

func detectAttributeCycles(st api.CatalogState, newStep *api.Step) error {
	steps := stepsIncluding(st, newStep)
	deps := st.Attributes
	if existing, ok := st.Steps[newStep.ID]; ok {
		deps = deps.RemoveStep(existing)
	}
	deps = deps.AddStep(newStep)
	return checkCycleFromStep(newStep.ID, deps, steps, stepSet{})
}

func detectFlowCycles(st api.CatalogState, newStep *api.Step) error {
	steps := stepsIncluding(st, newStep)
	for sid := range steps {
		if err := checkFlowCycleFromStep(sid, steps, stepSet{}); err != nil {
			return err
		}
	}
	return nil
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

	st := steps[currentID]
	for name, attr := range st.Attributes {
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

func checkFlowCycleFromStep(
	currentID api.StepID, steps api.Steps, stack stepSet,
) error {
	if stack.Contains(currentID) {
		return fmt.Errorf("%w: step %s", ErrCircularDependency, currentID)
	}

	step, ok := steps[currentID]
	if !ok || step.Type != api.StepTypeFlow || step.Flow == nil {
		return nil
	}

	stack.Add(currentID)
	defer stack.Remove(currentID)

	for _, goalID := range step.Flow.Goals {
		if err := checkFlowCycleFromStep(goalID, steps, stack); err != nil {
			return err
		}
	}

	return nil
}

func stepsIncluding(st api.CatalogState, newStep *api.Step) api.Steps {
	steps := maps.Clone(st.Steps)
	steps[newStep.ID] = newStep
	return steps
}
