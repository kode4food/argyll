package engine

import (
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	// attrCycleWalker walks attribute providers, depth first
	attrCycleWalker struct {
		deps  api.AttributeGraph
		steps api.Steps
		stack stepSet
	}

	// flowCycleWalker walks sub-flow goals, depth first
	flowCycleWalker struct {
		steps    api.Steps
		children func(*api.Step) ([]api.StepID, error)
		stack    stepSet
	}

	stepSet = util.Set[api.StepID]
)

func (w *attrCycleWalker) check(sid api.StepID) error {
	if w.stack.Contains(sid) {
		return fmt.Errorf("%w: step %s", ErrCircularDependency, sid)
	}

	w.stack.Add(sid)
	defer w.stack.Remove(sid)

	for name, attr := range w.steps[sid].Attributes {
		edges := w.deps[name]
		if !attr.IsInput() || edges == nil {
			continue
		}
		for _, providerID := range edges.Providers {
			if err := w.check(providerID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *flowCycleWalker) check(sid api.StepID) error {
	if w.stack.Contains(sid) {
		return fmt.Errorf("%w: step %s", ErrCircularDependency, sid)
	}

	st, ok := w.steps[sid]
	if !ok {
		return nil
	}
	childIDs, err := w.children(st)
	if err != nil {
		return err
	}
	if len(childIDs) == 0 {
		return nil
	}

	w.stack.Add(sid)
	defer w.stack.Remove(sid)

	for _, goalID := range childIDs {
		if err := w.check(goalID); err != nil {
			return err
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
	deps := cat.Attributes
	if existing, ok := cat.Steps[newStep.ID]; ok {
		deps = deps.RemoveStep(existing)
	}
	w := &attrCycleWalker{
		deps:  deps.AddStep(newStep),
		steps: stepsIncluding(cat, newStep),
		stack: stepSet{},
	}
	return w.check(newStep.ID)
}

func detectFlowCycles(
	cat api.CatalogState, newStep *api.Step,
	children func(*api.Step) ([]api.StepID, error),
) error {
	steps := stepsIncluding(cat, newStep)
	for sid := range steps {
		w := &flowCycleWalker{
			steps:    steps,
			children: children,
			stack:    stepSet{},
		}
		if err := w.check(sid); err != nil {
			return err
		}
	}
	return nil
}
