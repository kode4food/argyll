package engine

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/internal/events"
	"github.com/kode4food/spuds/engine/pkg/api"
	"github.com/kode4food/spuds/engine/pkg/util"
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

// CompleteFlow marks a flow as successfully completed with the given
// result outputs
func (e *Engine) CompleteFlow(
	ctx context.Context, flowID api.FlowID, result api.Args,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		return events.Raise(ag, api.EventTypeFlowCompleted,
			api.FlowCompletedEvent{
				FlowID: flowID,
				Result: result,
			},
		)
	}

	_, err := e.flowExec.Exec(ctx, flowKey(flowID), cmd)
	return err
}

// FailFlow marks a flow as failed with the specified error message
func (e *Engine) FailFlow(
	ctx context.Context, flowID api.FlowID, errMsg string,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		return events.Raise(ag, api.EventTypeFlowFailed,
			api.FlowFailedEvent{
				FlowID: flowID,
				Error:  errMsg,
			},
		)
	}

	_, err := e.flowExec.Exec(ctx, flowKey(flowID), cmd)
	return err
}

// StartWork begins execution of a work item for a step with the given token
// and input arguments
func (e *Engine) StartWork(
	ctx context.Context, fs FlowStep, token api.Token, inputs api.Args,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		return events.Raise(ag, api.EventTypeWorkStarted,
			api.WorkStartedEvent{
				FlowID: fs.FlowID,
				StepID: fs.StepID,
				Token:  token,
				Inputs: inputs,
			},
		)
	}

	_, err := e.flowExec.Exec(ctx, flowKey(fs.FlowID), cmd)
	return err
}

// CompleteWork marks a work item as successfully completed with the given
// output values
func (e *Engine) CompleteWork(
	ctx context.Context, fs FlowStep, token api.Token, outputs api.Args,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		return events.Raise(ag, api.EventTypeWorkSucceeded,
			api.WorkSucceededEvent{
				FlowID:  fs.FlowID,
				StepID:  fs.StepID,
				Token:   token,
				Outputs: outputs,
			},
		)
	}

	_, err := e.flowExec.Exec(ctx, flowKey(fs.FlowID), cmd)
	return err
}

// FailWork marks a work item as failed with the specified error message
func (e *Engine) FailWork(
	ctx context.Context, fs FlowStep, token api.Token, errMsg string,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		return events.Raise(ag, api.EventTypeWorkFailed,
			api.WorkFailedEvent{
				FlowID: fs.FlowID,
				StepID: fs.StepID,
				Token:  token,
				Error:  errMsg,
			},
		)
	}

	_, err := e.flowExec.Exec(ctx, flowKey(fs.FlowID), cmd)
	return err
}

// NotCompleteWork marks a work item as not completed with specified error
func (e *Engine) NotCompleteWork(
	ctx context.Context, fs FlowStep, token api.Token, errMsg string,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		return events.Raise(ag, api.EventTypeWorkNotCompleted,
			api.WorkNotCompletedEvent{
				FlowID: fs.FlowID,
				StepID: fs.StepID,
				Token:  token,
				Error:  errMsg,
			},
		)
	}

	_, err := e.flowExec.Exec(ctx, flowKey(fs.FlowID), cmd)
	return err
}

// SetAttribute sets a named attribute value in the flow state, returning
// an error if the attribute is already set
func (e *Engine) SetAttribute(
	ctx context.Context, fs FlowStep, attr api.Name, value any,
) error {
	cmd := func(st *api.FlowState, ag *FlowAggregator) error {
		if _, ok := st.Attributes[attr]; ok {
			return fmt.Errorf("%w: %s", ErrAttributeAlreadySet, attr)
		}

		return events.Raise(ag, api.EventTypeAttributeSet,
			api.AttributeSetEvent{
				FlowID: fs.FlowID,
				StepID: fs.StepID,
				Key:    attr,
				Value:  value,
			},
		)
	}

	_, err := e.flowExec.Exec(ctx, flowKey(fs.FlowID), cmd)
	return err
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

// GetAttributes retrieves all attributes from the flow state as a map of
// names to values
func (e *Engine) GetAttributes(
	ctx context.Context, flowID api.FlowID,
) (api.Args, error) {
	flow, err := e.GetFlowState(ctx, flowID)
	if err != nil {
		return nil, err
	}

	return flow.GetAttributeArgs(), nil
}

// GetFlowEvents retrieves all events for a flow starting from the
// specified sequence number
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
	step := flow.Plan.GetStep(stepID)
	if step == nil {
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
	consumers []api.StepID, executions map[api.StepID]*api.ExecutionState,
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

// IsFlowFailed determines if a flow has failed by checking whether any
// of its goal steps cannot be completed
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

// GetActiveFlow retrieves a flow if it is currently active, returning
// nil if the flow is in a terminal state or not found
func (e *Engine) GetActiveFlow(
	flowID api.FlowID,
) (*api.FlowState, bool) {
	flow, err := e.GetFlowState(e.ctx, flowID)
	if err != nil {
		slog.Error("Failed to get flow state",
			slog.Any("flow_id", flowID),
			slog.Any("error", err))
		return nil, false
	}

	if flowTransitions.IsTerminal(flow.Status) {
		return nil, false
	}

	return flow, true
}

func (e *Engine) ensureScriptsCompiled(flow *api.FlowState) bool {
	if !flow.Plan.NeedsCompilation() {
		return true
	}

	if err := e.scripts.CompilePlan(flow.Plan); err != nil {
		slog.Error("Failed to compile scripts",
			slog.Any("flow_id", flow.ID),
			slog.Any("error", err))
		return false
	}

	return true
}
