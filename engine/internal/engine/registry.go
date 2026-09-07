package engine

import (
	"errors"
	"fmt"
	"maps"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

type catalogTx struct {
	storeTx
	ag *CatalogAggregator
}

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
	return e.catalogTx(func(tx *catalogTx) error {
		return tx.remove(sid)
	})
}

// RegisterStep registers a new step with the engine after validating its
// configuration and checking for conflicts
func (e *Engine) RegisterStep(st *api.Step) error {
	return e.catalogTx(func(tx *catalogTx) error {
		return tx.register(st)
	})
}

// RegisterSteps registers several steps in one transaction, so a conflict
// among them leaves the catalog as it was
func (e *Engine) RegisterSteps(steps ...*api.Step) error {
	return e.catalogTx(func(tx *catalogTx) error {
		for _, st := range steps {
			if err := tx.register(st); err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateStep updates an existing step registration with new configuration
// after validation
func (e *Engine) UpdateStep(st *api.Step) error {
	return e.catalogTx(func(tx *catalogTx) error {
		return tx.update(st)
	})
}

func (e *Engine) catalogTx(fn func(*catalogTx) error) error {
	return e.engStore.Transact(func(t *timebox.Transaction) error {
		tx := storeTx{Engine: e, Transaction: t}
		_, err := tx.catalogTx(fn)
		return err
	})
}

func (tx storeTx) catalogTx(
	fn func(*catalogTx) error,
) (api.CatalogState, error) {
	return tx.Exec(tx.catalogExec, events.CatalogKey,
		func(_ api.CatalogState, ag *CatalogAggregator) error {
			return fn(&catalogTx{
				storeTx: tx,
				ag:      ag,
			})
		},
	)
}

func (tx *catalogTx) register(newStep *api.Step) error {
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
	err = tx.validateStepUpsert(cat, newStep, tx.steps.Children)
	if err != nil {
		return err
	}
	return tx.raiseStepRegisteredEvent(newStep)
}

func (tx *catalogTx) update(newStep *api.Step) error {
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
	err = tx.validateStepUpsert(cat, newStep, tx.steps.Children)
	if err != nil {
		return err
	}
	return tx.raiseStepUpdatedEvent(newStep)
}

func (tx *catalogTx) remove(sid api.StepID) error {
	cat := tx.ag.Value()
	if ref, ok := spaceSubFlowGoal(cat, sid); ok {
		return fmt.Errorf("%w: %s", ErrSubFlowGoalInUse, ref)
	}
	var spaces []api.SpaceID
	if oldStep, ok := cat.Steps[sid]; ok {
		var err error
		spaces, err = tx.matchingSpaceIDs(cat, oldStep)
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

func (tx *catalogTx) raiseStepRegisteredEvent(st *api.Step) error {
	return tx.raiseStepEvent(st, func(spaces []api.SpaceID) error {
		return events.Raise(tx.ag, api.EventTypeStepRegistered,
			api.StepRegisteredEvent{
				Step:   st,
				Spaces: spaces,
			},
		)
	})
}

func (tx *catalogTx) raiseStepUpdatedEvent(st *api.Step) error {
	return tx.raiseStepEvent(st, func(spaces []api.SpaceID) error {
		return events.Raise(tx.ag, api.EventTypeStepUpdated,
			api.StepUpdatedEvent{
				Step:   st,
				Spaces: spaces,
			},
		)
	})
}

// raiseStepEvent resolves the step's Spaces, raises the event built from
// them, and records the step's health in the same transaction. The catalog
// and cluster aggregates share a Store, so a registered step can never be
// left without the health its registration implies
func (tx *catalogTx) raiseStepEvent(
	st *api.Step, raise func([]api.SpaceID) error,
) error {
	spaces, err := tx.matchingSpaceIDs(tx.ag.Value(), st)
	if err != nil {
		return err
	}
	if err := raise(spaces); err != nil {
		return err
	}
	return resetStepHealth(tx.storeTx, st)
}

func (tx *catalogTx) prepareStep(st *api.Step) (*api.Step, error) {
	st = st.WithWorkDefaults(&tx.config.Work)
	st.Tags = st.Tags.Normalize()
	if err := tx.validateStep(st); err != nil {
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

func resetStepHealth(tx storeTx, st *api.Step) error {
	h, err := tx.steps.Health(st)
	if err != nil {
		return err
	}
	return updateStepHealth(tx, st.ID, h.Status, h.Error)
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

func stepsIncluding(cat api.CatalogState, newStep *api.Step) api.Steps {
	steps := maps.Clone(cat.Steps)
	steps[newStep.ID] = newStep
	return steps
}
