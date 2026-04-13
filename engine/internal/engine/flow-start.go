package engine

import (
	"errors"
	"fmt"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/flow"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

type flowTx struct {
	*Engine
	*FlowAggregator
	flowID api.FlowID
}

var (
	ErrFlowExists        = errors.New("flow exists")
	ErrInvariantViolated = errors.New("engine invariant violated")
)

// StartFlow begins a new flow execution with the given plan and options
func (e *Engine) StartFlow(
	flowID api.FlowID, pl *api.ExecutionPlan, apps ...flow.Applier,
) error {
	opts := flow.Defaults(apps...)
	if err := call.Perform(
		call.WithArg(validateParentMetadata, opts.Metadata),
		call.WithArg(pl.ValidateInputs, opts.Init),
	); err != nil {
		return err
	}

	return e.flowTx(flowID, func(tx *flowTx) error {
		if tx.Value().ID != "" {
			return ErrFlowExists
		}
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowStarted,
			api.FlowStartedEvent{
				FlowID:   flowID,
				Plan:     pl,
				Init:     opts.Init,
				Metadata: opts.Metadata,
				Labels:   opts.Labels,
			},
		); err != nil {
			return err
		}
		for _, stepID := range tx.findInitialSteps(tx.Value()) {
			if err := tx.prepareStep(stepID); err != nil {
				return err
			}
		}
		tx.OnSuccess(func(flow api.FlowState, _ []*timebox.Event) {
			tx.scheduleTimeouts(flow, tx.Now())
		})
		return nil
	})
}

func (e *Engine) StartChildFlow(
	parent api.FlowStep, tkn api.Token, pl *api.ExecutionPlan, init api.Args,
	meta api.Metadata,
) (api.FlowID, error) {
	childID := childFlowID(parent, tkn)
	err := e.StartFlow(childID, pl,
		flow.WithInit(init),
		flow.WithMetadata(meta),
		flow.WithParent(parent, tkn),
	)
	if err != nil {
		return "", err
	}

	return childID, nil
}

func childFlowID(parent api.FlowStep, tkn api.Token) api.FlowID {
	return api.FlowID(
		fmt.Sprintf("%s:%s:%s", parent.FlowID, parent.StepID, tkn),
	)
}

func (e *Engine) execFlow(
	flowID timebox.AggregateID, cmd timebox.Command[api.FlowState],
) (api.FlowState, error) {
	return e.flowExec.Exec(flowID, cmd)
}

func (e *Engine) flowTx(flowID api.FlowID, fn func(*flowTx) error) error {
	_, err := e.execFlow(events.FlowKey(flowID),
		func(_ api.FlowState, ag *FlowAggregator) error {
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
