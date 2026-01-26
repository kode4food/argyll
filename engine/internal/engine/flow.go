package engine

import (
	"slices"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util"
)

var flowTransitions = StateTransitions[api.FlowStatus]{
	api.FlowActive: util.SetOf(
		api.FlowCompleted,
		api.FlowFailed,
	),
	api.FlowCompleted: {},
	api.FlowFailed:    {},
}

// GetFlowState retrieves the current state of a flow by its ID
func (e *Engine) GetFlowState(flowID api.FlowID) (*api.FlowState, error) {
	state, _, err := e.GetFlowStateSeq(flowID)
	return state, err
}

// GetFlowStateSeq retrieves the current state and next sequence for a flow
func (e *Engine) GetFlowStateSeq(
	flowID api.FlowID,
) (*api.FlowState, int64, error) {
	var nextSeq int64
	state, err := e.execFlow(flowKey(flowID),
		func(st *api.FlowState, ag *FlowAggregator) error {
			nextSeq = ag.NextSequence()
			return nil
		},
	)
	if err != nil {
		return nil, 0, err
	}

	if state.ID == "" {
		return nil, 0, ErrFlowNotFound
	}

	return state, nextSeq, nil
}

// StartWork begins execution of a work item for a step with the given token
// and input arguments
func (e *Engine) StartWork(
	fs FlowStep, token api.Token, inputs api.Args,
) error {
	return e.raiseFlowEvent(fs.FlowID, api.EventTypeWorkStarted,
		api.WorkStartedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Token:  token,
			Inputs: inputs,
		},
	)
}

// CompleteWork marks a work item as successfully completed with the given
// output values
func (e *Engine) CompleteWork(
	fs FlowStep, token api.Token, outputs api.Args,
) error {
	a := &flowActor{Engine: e, flowID: fs.FlowID}
	return a.execTransaction(func(ag *FlowAggregator) error {
		if err := events.Raise(ag, api.EventTypeWorkSucceeded,
			api.WorkSucceededEvent{
				FlowID:  fs.FlowID,
				StepID:  fs.StepID,
				Token:   token,
				Outputs: outputs,
			},
		); err != nil {
			return err
		}
		ag.OnSuccess(func() {
			a.handleWorkSucceededCleanup(fs, token)
		})
		return a.handleWorkSucceeded(ag, fs.StepID)
	})
}

func (a *flowActor) handleWorkSucceededCleanup(fs FlowStep, token api.Token) {
	a.retryQueue.Remove(fs.FlowID, fs.StepID, token)
}

// FailWork marks a work item as failed with the specified error message
func (e *Engine) FailWork(fs FlowStep, token api.Token, errMsg string) error {
	a := &flowActor{Engine: e, flowID: fs.FlowID}
	return a.execTransaction(func(ag *FlowAggregator) error {
		if err := events.Raise(ag, api.EventTypeWorkFailed,
			api.WorkFailedEvent{
				FlowID: fs.FlowID,
				StepID: fs.StepID,
				Token:  token,
				Error:  errMsg,
			},
		); err != nil {
			return err
		}
		return a.handleWorkFailed(ag, fs.StepID)
	})
}

// NotCompleteWork marks a work item as not completed with specified error
func (e *Engine) NotCompleteWork(
	fs FlowStep, token api.Token, errMsg string,
) error {
	a := &flowActor{Engine: e, flowID: fs.FlowID}
	return a.execTransaction(func(ag *FlowAggregator) error {
		if err := events.Raise(ag, api.EventTypeWorkNotCompleted,
			api.WorkNotCompletedEvent{
				FlowID: fs.FlowID,
				StepID: fs.StepID,
				Token:  token,
				Error:  errMsg,
			},
		); err != nil {
			return err
		}
		return a.handleWorkNotCompleted(ag, fs.StepID, token)
	})
}

// GetAttribute retrieves a specific attribute value from the flow state,
// returning the value, whether it exists, and any error
func (e *Engine) GetAttribute(
	flowID api.FlowID, attr api.Name,
) (any, bool, error) {
	flow, err := e.GetFlowState(flowID)
	if err != nil {
		return nil, false, err
	}

	if av, ok := flow.Attributes[attr]; ok {
		return av.Value, true, nil
	}
	return nil, false, nil
}

// ListFlows returns summary information for active and deactivated flows
func (e *Engine) ListFlows() ([]*api.FlowsListItem, error) {
	engState, err := e.GetEngineState()
	if err != nil {
		return nil, err
	}

	count := len(engState.Active) + len(engState.Deactivated)
	flowIDs := make([]api.FlowID, 0, count)
	seen := make(map[api.FlowID]struct{}, count)
	for id, info := range engState.Active {
		if info != nil && info.ParentFlowID != "" {
			continue
		}
		seen[id] = struct{}{}
		flowIDs = append(flowIDs, id)
	}
	for _, info := range engState.Deactivated {
		if info.ParentFlowID != "" {
			continue
		}
		if _, ok := seen[info.FlowID]; ok {
			continue
		}
		seen[info.FlowID] = struct{}{}
		flowIDs = append(flowIDs, info.FlowID)
	}

	slices.Sort(flowIDs)

	digests := make([]*api.FlowsListItem, 0, len(flowIDs))
	for _, flowID := range flowIDs {
		digest, ok := engState.FlowDigests[flowID]
		if !ok || digest == nil {
			continue
		}
		digests = append(digests, &api.FlowsListItem{
			ID:     flowID,
			Digest: digest,
		})
	}

	return digests, nil
}

func (e *Engine) raiseFlowEvent(
	flowID api.FlowID, eventType api.EventType, data any,
) error {
	_, err := e.execFlow(flowKey(flowID),
		func(st *api.FlowState, ag *FlowAggregator) error {
			return events.Raise(ag, eventType, data)
		},
	)
	return err
}

func (e *Engine) execFlow(
	flowID timebox.AggregateID, cmd timebox.Command[*api.FlowState],
) (*api.FlowState, error) {
	return e.flowExec.Exec(e.ctx, flowID, cmd)
}

func flowKey(flowID api.FlowID) timebox.AggregateID {
	return timebox.NewAggregateID("flow", timebox.ID(flowID))
}
