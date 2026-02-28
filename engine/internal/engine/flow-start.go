package engine

import (
	"errors"
	"fmt"
	"maps"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/flowopt"
	"github.com/kode4food/argyll/engine/internal/engine/plan"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type flowTx struct {
	*Engine
	*FlowAggregator
	flowID api.FlowID
}

var ErrInvariantViolated = errors.New("engine invariant violated")

// StartFlow begins a new flow execution with the given plan and options
func (e *Engine) StartFlow(
	flowID api.FlowID, plan *api.ExecutionPlan, apps ...flowopt.Applier,
) error {
	existing, err := e.GetFlowState(flowID)
	if err == nil && existing.ID != "" {
		return ErrFlowExists
	}

	opts := flowopt.DefaultOptions(apps...)
	if err := plan.ValidateInputs(opts.Init); err != nil {
		return err
	}

	return e.flowTx(flowID, func(tx *flowTx) error {
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowStarted,
			api.FlowStartedEvent{
				FlowID:   flowID,
				Plan:     plan,
				Init:     opts.Init,
				Metadata: opts.Metadata,
				Labels:   opts.Labels,
			},
		); err != nil {
			return err
		}
		parentID, _ := api.GetMetaString[api.FlowID](
			opts.Metadata, api.MetaParentFlowID,
		)
		tx.OnSuccess(func(*api.FlowState) {
			tx.EnqueueEvent(api.EventTypeFlowActivated,
				api.FlowActivatedEvent{
					FlowID:       flowID,
					ParentFlowID: parentID,
					Labels:       opts.Labels,
				},
			)
		})
		if flowTransitions.IsTerminal(tx.Value().Status) {
			return nil
		}

		for _, stepID := range tx.findInitialSteps(tx.Value()) {
			if err := tx.prepareStep(stepID); err != nil {
				return err
			}
		}
		tx.OnSuccess(func(flow *api.FlowState) {
			tx.Engine.scheduleTimeouts(flow, tx.Now())
		})
		return nil
	})
}

func (e *Engine) StartChildFlow(
	parent api.FlowStep, token api.Token, step *api.Step, initState api.Args,
) (api.FlowID, error) {
	if step.Flow == nil || len(step.Flow.Goals) == 0 {
		return "", api.ErrFlowGoalsRequired
	}

	childID := childFlowID(parent, token)

	catState, err := e.GetCatalogState()
	if err != nil {
		return "", err
	}

	childPlan, err := plan.Create(catState, step.Flow.Goals, initState)
	if err != nil {
		return "", err
	}

	parentFlow, err := e.GetFlowState(parent.FlowID)
	if err != nil {
		return "", err
	}

	meta := maps.Clone(parentFlow.Metadata)
	if meta == nil {
		meta = api.Metadata{}
	}
	meta[api.MetaParentFlowID] = parent.FlowID
	meta[api.MetaParentStepID] = parent.StepID
	meta[api.MetaParentWorkItemToken] = token

	if err := e.StartFlow(childID, childPlan,
		flowopt.WithInit(initState),
		flowopt.WithMetadata(meta),
	); err != nil {
		if errors.Is(err, ErrFlowExists) {
			return childID, nil
		}
		return "", err
	}

	return childID, nil
}

func childFlowID(parent api.FlowStep, token api.Token) api.FlowID {
	return api.FlowID(
		fmt.Sprintf("%s:%s:%s", parent.FlowID, parent.StepID, token),
	)
}

func (e *Engine) execFlow(
	flowID timebox.AggregateID, cmd timebox.Command[*api.FlowState],
) (*api.FlowState, error) {
	return e.flowExec.Exec(e.ctx, flowID, cmd)
}

func (e *Engine) flowTx(flowID api.FlowID, fn func(*flowTx) error) error {
	_, err := e.execFlow(events.FlowKey(flowID),
		func(_ *api.FlowState, ag *FlowAggregator) error {
			tx := &flowTx{
				Engine:         e,
				FlowAggregator: ag,
				flowID:         flowID,
			}
			return fn(tx)
		},
	)
	return err
}
