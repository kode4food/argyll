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
	if err := validateStepUpsert(st, step, tx.e.steps.Children); err != nil {
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
	if err := validateStepUpsert(st, step, tx.e.steps.Children); err != nil {
		return err
	}
	return tx.e.raiseStepUpdatedEvent(step, tx.ag)
}

func (tx *CatalogTx) Remove(stepID api.StepID) error {
	return events.Raise(tx.ag, api.EventTypeStepUnregistered,
		api.StepUnregisteredEvent{StepID: stepID},
	)
}

func (e *Engine) validateStep(st *api.Step) error {
	if err := call.Perform(
		st.Validate,
		call.WithArg(e.mapper.validateStep, st),
		call.WithArg(e.scripts.ValidateStep, st),
		call.WithArg(e.steps.Validate, st),
	); err != nil {
		return errors.Join(ErrInvalidStep, err)
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
	h, err := e.steps.Health(step)
	if err != nil {
		slog.Error("Failed to evaluate step health",
			log.StepID(step.ID),
			log.Error(err))
		return
	}
	if err := e.UpdateStepHealth(step.ID, h.Status, h.Error); err != nil {
		slog.Error("Failed to update step health",
			log.StepID(step.ID),
			log.Error(err))
	}
}

func (tx *CatalogTx) prepareStep(step *api.Step) (*api.Step, error) {
	step = step.WithWorkDefaults(&tx.e.config.Work)
	if err := tx.e.validateStep(step); err != nil {
		return nil, err
	}
	return step, nil
}

func validateStepUpsert(
	st api.CatalogState, newStep *api.Step,
	children func(*api.Step) ([]api.StepID, error),
) error {
	if err := call.Perform(
		call.WithArgs(validateAttributeTypes, st, newStep),
		func() error { return detectStepCycles(st, newStep, children) },
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

func detectStepCycles(
	st api.CatalogState, newStep *api.Step,
	children func(*api.Step) ([]api.StepID, error),
) error {
	if err := detectAttributeCycles(st, newStep); err != nil {
		return err
	}
	return detectFlowCycles(st, newStep, children)
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

func detectFlowCycles(
	st api.CatalogState, newStep *api.Step,
	children func(*api.Step) ([]api.StepID, error),
) error {
	steps := stepsIncluding(st, newStep)
	for sid := range steps {
		if err := checkFlowCycleFromStep(
			sid, steps, children, stepSet{},
		); err != nil {
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
	currentID api.StepID, steps api.Steps,
	children func(*api.Step) ([]api.StepID, error), stack stepSet,
) error {
	if stack.Contains(currentID) {
		return fmt.Errorf("%w: step %s", ErrCircularDependency, currentID)
	}

	st, ok := steps[currentID]
	if !ok {
		return nil
	}
	childIDs, err := children(st)
	if err != nil {
		return err
	}
	if len(childIDs) == 0 {
		return nil
	}

	stack.Add(currentID)
	defer stack.Remove(currentID)

	for _, goalID := range childIDs {
		if err := checkFlowCycleFromStep(
			goalID, steps, children, stack,
		); err != nil {
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
