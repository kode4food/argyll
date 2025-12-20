package engine

import (
	"context"
	"slices"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/events"
	"github.com/kode4food/argyll/engine/pkg/api"
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
func (e *Engine) GetFlowState(
	ctx context.Context, flowID api.FlowID,
) (*api.FlowState, error) {
	state, err := e.flowExec.Exec(ctx, flowKey(flowID),
		func(st *api.FlowState, ag *FlowAggregator) error {
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	if state.ID == "" {
		return nil, ErrFlowNotFound
	}

	return state, nil
}

// StartWork begins execution of a work item for a step with the given token
// and input arguments
func (e *Engine) StartWork(
	ctx context.Context, fs FlowStep, token api.Token, inputs api.Args,
) error {
	return e.raiseFlowEvent(ctx, fs.FlowID, api.EventTypeWorkStarted,
		api.WorkStartedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Token:  token,
			Inputs: inputs,
		})
}

// CompleteWork marks a work item as successfully completed with the given
// output values
func (e *Engine) CompleteWork(
	ctx context.Context, fs FlowStep, token api.Token, outputs api.Args,
) error {
	return e.raiseFlowEvent(ctx, fs.FlowID, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  fs.FlowID,
			StepID:  fs.StepID,
			Token:   token,
			Outputs: outputs,
		})
}

// FailWork marks a work item as failed with the specified error message
func (e *Engine) FailWork(
	ctx context.Context, fs FlowStep, token api.Token, errMsg string,
) error {
	return e.raiseFlowEvent(ctx, fs.FlowID, api.EventTypeWorkFailed,
		api.WorkFailedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Token:  token,
			Error:  errMsg,
		})
}

// NotCompleteWork marks a work item as not completed with specified error
func (e *Engine) NotCompleteWork(
	ctx context.Context, fs FlowStep, token api.Token, errMsg string,
) error {
	return e.raiseFlowEvent(ctx, fs.FlowID, api.EventTypeWorkNotCompleted,
		api.WorkNotCompletedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Token:  token,
			Error:  errMsg,
		})
}

// GetAttribute retrieves a specific attribute value from the flow state,
// returning the value, whether it exists, and any error
func (e *Engine) GetAttribute(
	ctx context.Context, flowID api.FlowID, attr api.Name,
) (any, bool, error) {
	flow, err := e.GetFlowState(ctx, flowID)
	if err != nil {
		return nil, false, err
	}

	if av, ok := flow.Attributes[attr]; ok {
		return av.Value, true, nil
	}
	return nil, false, nil
}

// GetFlowEvents retrieves all events for a flow starting from the specified
// sequence number
func (e *Engine) GetFlowEvents(
	ctx context.Context, flowID api.FlowID, fromSeq int64,
) ([]*timebox.Event, error) {
	return e.flowExec.GetStore().GetEvents(ctx, flowKey(flowID), fromSeq)
}

// ListFlows returns summary information for all flows in the system
func (e *Engine) ListFlows(ctx context.Context) ([]*api.FlowDigest, error) {
	ids, err := e.flowExec.GetStore().ListAggregates(ctx, flowKey("*"))
	if err != nil {
		return nil, err
	}

	var digests []*api.FlowDigest
	for _, id := range ids {
		if digest := e.buildFlowDigest(ctx, id); digest != nil {
			digests = append(digests, digest)
		}
	}

	return digests, nil
}

func (e *Engine) buildFlowDigest(
	ctx context.Context, id timebox.AggregateID,
) *api.FlowDigest {
	if len(id) < 2 || id[0] != "flow" {
		return nil
	}

	flowID := api.FlowID(id[1])
	flow, err := e.GetFlowState(ctx, flowID)
	if err != nil {
		return nil
	}

	return &api.FlowDigest{
		ID:          flow.ID,
		Status:      flow.Status,
		CreatedAt:   flow.CreatedAt,
		CompletedAt: flow.CompletedAt,
		Error:       flow.Error,
	}
}

func (e *Engine) areOutputsNeeded(stepID api.StepID, flow *api.FlowState) bool {
	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return false
	}

	if isGoalStep(stepID, flow.Plan.Goals) {
		return true
	}

	return hasOutputNeededByPendingConsumers(step, flow)
}

func isGoalStep(stepID api.StepID, goals []api.StepID) bool {
	return slices.Contains(goals, stepID)
}

func hasOutputNeededByPendingConsumers(
	step *api.Step, flow *api.FlowState,
) bool {
	for name, attr := range step.Attributes {
		if outputNeededByPendingConsumer(name, attr, flow) {
			return true
		}
	}
	return false
}

func outputNeededByPendingConsumer(
	name api.Name, attr *api.AttributeSpec, flow *api.FlowState,
) bool {
	if !attr.IsOutput() {
		return false
	}

	if _, alreadySatisfied := flow.Attributes[name]; alreadySatisfied {
		return false
	}

	attrDeps, ok := flow.Plan.Attributes[name]
	if !ok || len(attrDeps.Consumers) == 0 {
		return false
	}

	return hasPendingConsumer(attrDeps.Consumers, flow.Executions)
}

func hasPendingConsumer(
	consumers []api.StepID, executions api.Executions,
) bool {
	for _, consumerID := range consumers {
		consumerExec, ok := executions[consumerID]
		if !ok {
			continue
		}
		if consumerExec.Status == api.StepPending {
			return true
		}
	}
	return false
}

func (e *Engine) isFlowComplete(flow *api.FlowState) bool {
	for stepID := range flow.Plan.Steps {
		if !e.isStepComplete(stepID, flow) {
			return false
		}
	}
	return true
}

// IsFlowFailed determines if a flow has failed by checking whether any of its
// goal steps cannot be completed
func (e *Engine) IsFlowFailed(flow *api.FlowState) bool {
	for _, goalID := range flow.Plan.Goals {
		if !e.canStepComplete(goalID, flow) {
			return true
		}
	}
	return false
}

// HasInputProvider checks if a required attribute has at least one step that
// can provide it in the flow execution plan
func (e *Engine) HasInputProvider(name api.Name, flow *api.FlowState) bool {
	deps := flow.Plan.Attributes[name]
	if deps == nil {
		return false
	}

	if len(deps.Providers) == 0 {
		return true
	}

	for _, providerID := range deps.Providers {
		if e.canStepComplete(providerID, flow) {
			return true
		}
	}
	return false
}

func flowKey(flowID api.FlowID) timebox.AggregateID {
	return timebox.NewAggregateID("flow", timebox.ID(flowID))
}

func (e *Engine) raiseFlowEvent(
	ctx context.Context, flowID api.FlowID, eventType api.EventType, data any,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		return events.Raise(ag, eventType, data)
	}
	_, err := e.flowExec.Exec(ctx, flowKey(flowID), cmd)
	return err
}
