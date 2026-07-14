package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

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
	ErrFlowExists        = errors.New("flow exists with different plan or init")
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
			match, err := e.matchesStartedFlow(flowID, pl, opts.Init)
			if err != nil {
				return err
			}
			if match {
				return nil
			}
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
		for _, sid := range tx.findInitialSteps(tx.Value()) {
			if err := tx.prepareStep(sid); err != nil {
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
	parent api.FlowStep, tkn api.Token, pl *api.ExecutionPlan,
	init api.InitArgs, meta api.Metadata,
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

func (e *Engine) matchesStartedFlow(
	flowID api.FlowID, pl *api.ExecutionPlan, init api.InitArgs,
) (bool, error) {
	evs, err := e.GetFlowEvents(flowID)
	if err != nil {
		return false, err
	}
	for _, ev := range evs {
		if api.EventType(ev.Type) != api.EventTypeFlowStarted {
			continue
		}
		data, err := timebox.GetEventValue[api.FlowStartedEvent](ev)
		if err != nil {
			return false, err
		}
		return data.FlowID == flowID &&
			slices.Equal(data.Plan.Goals, pl.Goals) &&
			initArgsEqual(data.Init, init), nil
	}
	return false, nil
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

func childFlowID(parent api.FlowStep, tkn api.Token) api.FlowID {
	return api.FlowID(
		fmt.Sprintf("%s:%s:%s", parent.FlowID, parent.StepID, tkn),
	)
}

func initArgsEqual(a, b api.InitArgs) bool {
	aj, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bj, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aj) == string(bj)
}
