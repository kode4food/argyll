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
	storeTx
	*FlowAggregator
	flowID api.FlowID
}

var (
	ErrFlowExists        = errors.New("flow exists with different plan or init")
	ErrInvariantViolated = errors.New("engine invariant violated")
)

// StartFlow begins a new flow execution with the given plan and options
func (e *Engine) StartFlow(
	fid api.FlowID, pl *api.ExecutionPlan, apps ...flow.Applier,
) error {
	opts := flow.Defaults(apps...)
	return e.flowTx(fid, func(tx *flowTx) error {
		return tx.startFlow(pl, opts)
	})
}

func (tx *flowTx) startFlow(pl *api.ExecutionPlan, opts *flow.Options) error {
	if err := call.Perform(
		call.WithArg(validateParentMetadata, opts.Metadata),
		call.WithArg(pl.ValidateInputs, opts.Init),
	); err != nil {
		return err
	}

	if tx.Value().ID != "" {
		match, err := tx.matchesStartedFlow(tx.flowID, pl, opts.Init)
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
			FlowID:     tx.flowID,
			Plan:       pl,
			Init:       opts.Init,
			Metadata:   opts.Metadata,
			Tags:       opts.Tags,
			Compensate: opts.Compensate,
		},
	); err != nil {
		return err
	}
	for _, sid := range tx.findInitialSteps(tx.Value()) {
		if err := tx.prepareStep(sid); err != nil {
			return err
		}
	}
	tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
		tx.scheduleTimeouts(fl, tx.Now())
	})
	return nil
}

func (tx *flowTx) startChildFlow(
	sid api.StepID, tkn api.Token, inputs api.Args,
) error {
	fl := tx.Value()
	st := fl.Plan.Steps[sid]
	fs := api.FlowStep{FlowID: fl.ID, StepID: sid}
	init := api.InitArgs{}
	for name, value := range inputs {
		init[name] = []any{value}
	}
	opts := flow.Defaults(
		flow.WithInit(init),
		flow.WithMetadata(fl.Metadata),
		flow.WithParent(fs, tkn),
		flow.WithCompensate(st.Flow != nil && st.Flow.Compensate),
	)
	_, err := tx.flowTx(childFlowID(fs, tkn), func(child *flowTx) error {
		return child.startFlow(fl.Plan.Children[sid], opts)
	})
	return err
}

func (e *Engine) matchesStartedFlow(
	fid api.FlowID, pl *api.ExecutionPlan, init api.InitArgs,
) (bool, error) {
	evs, err := e.GetFlowEvents(fid)
	if err != nil {
		return false, err
	}
	for _, ev := range evs {
		if api.EventType(ev.Type) != api.EventTypeFlowStarted {
			continue
		}
		data, err := ev.GetValue[api.FlowStartedEvent]()
		if err != nil {
			return false, err
		}
		return data.FlowID == fid &&
			slices.Equal(data.Plan.Goals, pl.Goals) &&
			initArgsEqual(data.Init, init), nil
	}
	return false, nil
}

func (e *Engine) flowTx(fid api.FlowID, fn func(*flowTx) error) error {
	return e.flowExec.GetStore().Transact(
		func(t *timebox.Transaction) error {
			tx := storeTx{Engine: e, Transaction: t}
			_, err := tx.flowTx(fid, fn)
			return err
		},
	)
}

func (tx storeTx) flowTx(
	fid api.FlowID, fn func(*flowTx) error,
) (api.FlowState, error) {
	return tx.Exec(tx.flowExec, events.FlowKey(fid),
		func(_ api.FlowState, ag *FlowAggregator) error {
			return fn(&flowTx{
				storeTx:        tx,
				FlowAggregator: ag,
				flowID:         fid,
			})
		},
	)
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
