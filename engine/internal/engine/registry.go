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
	ErrSubFlowGoalInUse   = errors.New("goal in use")
	ErrTypeConflict       = errors.New("attribute type conflict")
	ErrCircularDependency = errors.New("circular dependency detected")
)

// UnregisterStep removes a step from the engine registry
func (e *Engine) UnregisterStep(sid api.StepID) error {
	return e.CatalogTx(func(tx *CatalogTx) error {
		return tx.Remove(sid)
	})
}

// RegisterStep registers a new step with the engine after validating its
// configuration and checking for conflicts
func (e *Engine) RegisterStep(st *api.Step) error {
	return e.CatalogTx(func(tx *CatalogTx) error {
		return tx.Register(st)
	})
}

// UpdateStep updates an existing step registration with new configuration
// after validation
func (e *Engine) UpdateStep(st *api.Step) error {
	return e.CatalogTx(func(tx *CatalogTx) error {
		return tx.Update(st)
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

func (tx *CatalogTx) Register(newStep *api.Step) error {
	newStep, err := tx.prepareStep(newStep)
	if err != nil {
		return err
	}
	cat := tx.ag.Value()
	if old, ok := cat.Steps[newStep.ID]; ok {
		if old.Equal(newStep) {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrStepExists, newStep.ID)
	}
	err = tx.e.validateStepUpsert(cat, newStep, tx.e.steps.Children)
	if err != nil {
		return err
	}
	return tx.e.raiseStepRegisteredEvent(newStep, tx.ag)
}

func (tx *CatalogTx) Update(newStep *api.Step) error {
	newStep, err := tx.prepareStep(newStep)
	if err != nil {
		return err
	}
	cat := tx.ag.Value()
	old, ok := cat.Steps[newStep.ID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrStepNotFound, newStep.ID)
	}
	if old.Equal(newStep) {
		return nil
	}
	err = tx.e.validateStepUpsert(cat, newStep, tx.e.steps.Children)
	if err != nil {
		return err
	}
	return tx.e.raiseStepUpdatedEvent(newStep, tx.ag)
}

func (tx *CatalogTx) Remove(sid api.StepID) error {
	cat := tx.ag.Value()
	if ref, ok := spaceSubFlowGoal(cat, sid); ok {
		return fmt.Errorf("%w: %s", ErrSubFlowGoalInUse, ref)
	}
	var spaces []api.SpaceID
	if oldStep, ok := cat.Steps[sid]; ok {
		var err error
		spaces, err = tx.e.matchingSpaceIDs(cat, oldStep)
		if err != nil {
			return err
		}
	}
	return events.Raise(tx.ag, api.EventTypeStepUnregistered,
		api.StepUnregisteredEvent{StepID: sid, Spaces: spaces},
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
	st *api.Step, ag *CatalogAggregator,
) error {
	spaces, err := e.matchingSpaceIDs(ag.Value(), st)
	if err != nil {
		return err
	}
	if err := events.Raise(ag, api.EventTypeStepRegistered,
		api.StepRegisteredEvent{
			Step:   st,
			Spaces: spaces,
		},
	); err != nil {
		return err
	}
	ag.OnSuccess(func(api.CatalogState, []*timebox.Event) {
		e.resetStepHealth(st)
	})
	return nil
}

func (e *Engine) raiseStepUpdatedEvent(
	st *api.Step, ag *CatalogAggregator,
) error {
	spaces, err := e.matchingSpaceIDs(ag.Value(), st)
	if err != nil {
		return err
	}
	if err := events.Raise(ag, api.EventTypeStepUpdated,
		api.StepUpdatedEvent{
			Step:   st,
			Spaces: spaces,
		},
	); err != nil {
		return err
	}
	ag.OnSuccess(func(api.CatalogState, []*timebox.Event) {
		e.resetStepHealth(st)
	})
	return nil
}

func (e *Engine) resetStepHealth(st *api.Step) {
	h, err := e.steps.Health(st)
	if err != nil {
		slog.Error("Failed to evaluate step health",
			log.StepID(st.ID),
			log.Error(err))
		return
	}
	if err := e.UpdateStepHealth(st.ID, h.Status, h.Error); err != nil {
		slog.Error("Failed to update step health",
			log.StepID(st.ID),
			log.Error(err))
	}
}

func (tx *CatalogTx) prepareStep(st *api.Step) (*api.Step, error) {
	st = st.WithWorkDefaults(&tx.e.config.Work)
	st.Tags = st.Tags.Normalize()
	if err := tx.e.validateStep(st); err != nil {
		return nil, err
	}
	return st, nil
}

func (e *Engine) validateStepUpsert(
	cat api.CatalogState, newStep *api.Step,
	children func(*api.Step) ([]api.StepID, error),
) error {
	if err := call.Perform(
		call.WithArgs(validateAttributeTypes, cat, newStep),
		call.WithArgs(e.validateSpaceSubFlows, cat, newStep),
		func() error { return detectStepCycles(cat, newStep, children) },
	); err != nil {
		return errors.Join(ErrInvalidStep, err)
	}
	return nil
}

func validateAttributeTypes(cat api.CatalogState, newStep *api.Step) error {
	attributeTypes := collectAttributeTypes(cat, newStep.ID)
	return checkAttributeConflicts(newStep.Attributes, attributeTypes)
}

func collectAttributeTypes(
	cat api.CatalogState, excludeID api.StepID,
) api.AttributeTypes {
	attributeTypes := make(api.AttributeTypes)
	for sid, st := range cat.Steps {
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
	cat api.CatalogState, newStep *api.Step,
	children func(*api.Step) ([]api.StepID, error),
) error {
	if err := detectAttributeCycles(cat, newStep); err != nil {
		return err
	}
	return detectFlowCycles(cat, newStep, children)
}

func detectAttributeCycles(cat api.CatalogState, newStep *api.Step) error {
	steps := stepsIncluding(cat, newStep)
	deps := cat.Attributes
	if existing, ok := cat.Steps[newStep.ID]; ok {
		deps = deps.RemoveStep(existing)
	}
	deps = deps.AddStep(newStep)
	return checkCycleFromStep(newStep.ID, deps, steps, stepSet{})
}

func detectFlowCycles(
	cat api.CatalogState, newStep *api.Step,
	children func(*api.Step) ([]api.StepID, error),
) error {
	steps := stepsIncluding(cat, newStep)
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

func stepsIncluding(cat api.CatalogState, newStep *api.Step) api.Steps {
	steps := maps.Clone(cat.Steps)
	steps[newStep.ID] = newStep
	return steps
}
