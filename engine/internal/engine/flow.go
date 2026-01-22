package engine

import (
	"context"
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
	return e.raiseFlowEvent(fs.FlowID, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  fs.FlowID,
			StepID:  fs.StepID,
			Token:   token,
			Outputs: outputs,
		},
	)
}

// FailWork marks a work item as failed with the specified error message
func (e *Engine) FailWork(fs FlowStep, token api.Token, errMsg string) error {
	return e.raiseFlowEvent(fs.FlowID, api.EventTypeWorkFailed,
		api.WorkFailedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Token:  token,
			Error:  errMsg,
		},
	)
}

// NotCompleteWork marks a work item as not completed with specified error
func (e *Engine) NotCompleteWork(
	fs FlowStep, token api.Token, errMsg string,
) error {
	return e.raiseFlowEvent(fs.FlowID, api.EventTypeWorkNotCompleted,
		api.WorkNotCompletedEvent{
			FlowID: fs.FlowID,
			StepID: fs.StepID,
			Token:  token,
			Error:  errMsg,
		},
	)
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

// ListFlows returns summary information for all flows in the system
func (e *Engine) ListFlows() ([]*api.FlowDigest, error) {
	bg := context.Background()
	ids, err := e.flowExec.GetStore().ListAggregates(bg, flowKey("*"))
	if err != nil {
		return nil, err
	}

	var digests []*api.FlowDigest
	for _, id := range ids {
		if digest := e.buildFlowDigest(id); digest != nil {
			digests = append(digests, digest)
		}
	}

	return digests, nil
}

func (e *Engine) buildFlowDigest(id timebox.AggregateID) *api.FlowDigest {
	if len(id) < 2 || id[0] != "flow" {
		return nil
	}

	flowID := api.FlowID(id[1])
	flow, err := e.GetFlowState(flowID)
	if err != nil {
		return nil
	}

	if _, ok := api.GetMetaString[api.FlowID](
		flow.Metadata,
		api.MetaParentFlowID,
	); ok {
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
	flowID api.FlowID, eventType api.EventType, data any,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		return events.Raise(ag, eventType, data)
	}
	_, err := e.execFlow(flowKey(flowID), cmd)
	return err
}

func (e *Engine) execFlow(
	flowID timebox.AggregateID, cmd timebox.Command[*api.FlowState],
) (*api.FlowState, error) {
	return e.flowExec.Exec(e.ctx, flowID, cmd)
}
