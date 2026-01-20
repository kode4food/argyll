package engine

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

type (
	flowActor struct {
		*Engine
		flowID api.FlowID
		events chan *timebox.Event
	}

	// deferred represents deferred work to be executed after a transaction
	deferred func()
	enqueued []deferred
)

func (a *flowActor) run() {
	defer a.wg.Done()
	defer a.flows.Delete(a.flowID)

	handler := a.createEventHandler()
	idleTimer := time.NewTimer(100 * time.Millisecond)
	defer idleTimer.Stop()

	for {
		select {
		case event := <-a.events:
			a.handleEvent(event, handler)

			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(100 * time.Millisecond)

		case <-idleTimer.C:
			select {
			case event := <-a.events:
				a.handleEvent(event, handler)
				idleTimer.Reset(100 * time.Millisecond)
			default:
				return
			}

		case <-a.ctx.Done():
			return
		}
	}
}

// isGoalStep returns true if the step is a goal step
func (a *flowActor) isGoalStep(stepID api.StepID, flow *api.FlowState) bool {
	return slices.Contains(flow.Plan.Goals, stepID)
}

// checkTerminal checks for flow completion or failure
func (a *flowActor) checkTerminal(ag *FlowAggregator) error {
	flow := ag.Value()
	if a.isFlowComplete(flow) {
		result := api.Args{}
		for _, goalID := range flow.Plan.Goals {
			if goal := flow.Executions[goalID]; goal != nil {
				maps.Copy(result, goal.Outputs)
			}
		}
		return events.Raise(ag, api.EventTypeFlowCompleted,
			api.FlowCompletedEvent{
				FlowID: a.flowID,
				Result: result,
			},
		)
	}
	if a.IsFlowFailed(flow) {
		return events.Raise(ag, api.EventTypeFlowFailed,
			api.FlowFailedEvent{
				FlowID: a.flowID,
				Error:  a.getFailureReason(flow),
			},
		)
	}
	return nil
}

// failUnreachable finds and fails all pending steps that can no longer
// complete because their required inputs cannot be satisfied
func (a *flowActor) failUnreachable(ag *FlowAggregator) error {
	for {
		failedAny := false
		flow := ag.Value()

		for stepID, exec := range flow.Executions {
			if exec.Status != api.StepPending {
				continue
			}
			if a.canStepComplete(stepID, flow) {
				continue
			}

			if err := events.Raise(ag, api.EventTypeStepFailed,
				api.StepFailedEvent{
					FlowID: a.flowID,
					StepID: stepID,
					Error:  "required input no longer available",
				},
			); err != nil {
				return err
			}
			failedAny = true
		}

		if !failedAny {
			return nil
		}
	}
}

func (a *flowActor) skipPendingUnused(ag *FlowAggregator) error {
	for {
		skip := false
		flow := ag.Value()

		for stepID, exec := range flow.Executions {
			if exec.Status != api.StepPending {
				continue
			}
			if a.areOutputsNeeded(stepID, flow) {
				continue
			}

			if err := events.Raise(ag, api.EventTypeStepSkipped,
				api.StepSkippedEvent{
					FlowID: a.flowID,
					StepID: stepID,
					Reason: "outputs not needed",
				},
			); err != nil {
				return err
			}
			skip = true
		}

		if !skip {
			return nil
		}
	}
}

// getFailureReason extracts a failure reason from flow state
func (a *flowActor) getFailureReason(flow *api.FlowState) string {
	for stepID, exec := range flow.Executions {
		if exec.Status == api.StepFailed {
			return fmt.Sprintf("step %s failed: %s", stepID, exec.Error)
		}
	}
	return "flow failed"
}

func (a *flowActor) canStartStep(stepID api.StepID, flow *api.FlowState) bool {
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return false
	}
	if !a.hasRequired(stepID, flow) {
		return false
	}
	return a.areOutputsNeeded(stepID, flow)
}

func (a *flowActor) hasRequired(stepID api.StepID, flow *api.FlowState) bool {
	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return false
	}
	for name, attr := range step.Attributes {
		if attr.IsRequired() {
			if _, ok := flow.Attributes[name]; !ok {
				return false
			}
		}
	}
	return true
}

// prepareStep validates and prepares a step for execution within a
// transaction, raising the StepStarted event via aggregator and returning a
// deferred function to be executed after transaction commit
func (a *flowActor) prepareStep(
	ctx context.Context, stepID api.StepID, ag *FlowAggregator,
) (deferred, error) {
	flow := ag.Value()

	// Validate step exists and is pending
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return nil, fmt.Errorf("%w: %s (status=%s)",
			ErrStepAlreadyPending, stepID, exec.Status)
	}

	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
	}

	// Collect inputs
	inputs := a.collectStepInputs(step, flow.GetAttributes())

	// Evaluate predicate
	fs := FlowStep{FlowID: a.flowID, StepID: stepID}
	if !a.evaluateStepPredicate(ctx, fs, step, inputs) {
		// Predicate failed - skip this step
		if err := events.Raise(ag, api.EventTypeStepSkipped,
			api.StepSkippedEvent{
				FlowID: a.flowID,
				StepID: stepID,
				Reason: "predicate returned false",
			},
		); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Compute work items
	workItemsList := computeWorkItems(step, inputs)
	workItemsMap := map[api.Token]api.Args{}
	for _, workInputs := range workItemsList {
		token := api.Token(uuid.New().String())
		workItemsMap[token] = workInputs
	}

	// Raise StepStarted event with work items
	if err := events.Raise(ag, api.EventTypeStepStarted, api.StepStartedEvent{
		FlowID:    a.flowID,
		StepID:    stepID,
		Inputs:    inputs,
		WorkItems: workItemsMap,
	}); err != nil {
		return nil, err
	}

	// Create deferred to execute after transaction commits
	return func() {
		execCtx := &ExecContext{
			engine: a.Engine,
			flowID: a.flowID,
			stepID: stepID,
			step:   step,
			inputs: inputs,
			meta:   ag.Value().Metadata,
		}

		// Execute work items
		items := ag.Value().Executions[stepID].WorkItems
		execCtx.executeWorkItems(ctx, items)
	}, nil
}

func (e enqueued) exec() {
	for _, fn := range e {
		fn()
	}
}

// handleStepFailure handles common failure logic for processWorkFailed and
// processWorkNotCompleted - checking step completion and propagating failures.
// Returns a deferred archiving function if the flow becomes ready to archive
func (a *flowActor) handleStepFailure(
	ag *FlowAggregator, stepID api.StepID,
) (deferred, error) {
	if flowTransitions.IsTerminal(ag.Value().Status) {
		_, err := a.checkStepCompletion(ag, stepID)
		if err != nil {
			return nil, err
		}
		return a.maybeDeactivate(ag.Value()), nil
	}

	completed, err := a.checkStepCompletion(ag, stepID)
	if err != nil || !completed {
		return nil, err
	}

	if err := a.failUnreachable(ag); err != nil {
		return nil, err
	}
	return nil, a.checkTerminal(ag)
}

// findInitialSteps finds steps that can start when a flow begins
func (a *flowActor) findInitialSteps(flow *api.FlowState) []api.StepID {
	var ready []api.StepID

	for stepID := range flow.Plan.Steps {
		if a.canStartStep(stepID, flow) {
			ready = append(ready, stepID)
		}
	}

	return ready
}

// findReadySteps finds ready steps among downstream consumers of a completed
// step
func (a *flowActor) findReadySteps(
	stepID api.StepID, flow *api.FlowState,
) []api.StepID {
	var ready []api.StepID

	for _, consumerID := range a.getDownstreamConsumers(stepID, flow) {
		if a.canStartStep(consumerID, flow) {
			ready = append(ready, consumerID)
		}
	}

	return ready
}

// checkStepCompletion checks if a specific step can complete (all work items
// done) and raises appropriate completion or failure events
func (a *flowActor) checkStepCompletion(
	ag *FlowAggregator, stepID api.StepID,
) (bool, error) {
	exec, ok := ag.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return false, nil
	}

	allDone := true
	hasFailed := false
	var failureError string

	for _, item := range exec.WorkItems {
		switch item.Status {
		case api.WorkSucceeded:
			// continue
		case api.WorkFailed:
			hasFailed = true
			if failureError == "" {
				failureError = item.Error
			}
		case api.WorkNotCompleted, api.WorkPending, api.WorkActive:
			allDone = false
		}
	}

	if !allDone {
		return false, nil
	}

	if hasFailed {
		if failureError == "" {
			failureError = "work item failed"
		}
		return true, events.Raise(ag, api.EventTypeStepFailed,
			api.StepFailedEvent{
				FlowID: a.flowID,
				StepID: stepID,
				Error:  failureError,
			},
		)
	}

	// Step succeeded - set attributes and raise completion
	step := ag.Value().Plan.Steps[stepID]
	outputs := aggregateWorkItemOutputs(exec.WorkItems, step)
	dur := time.Since(exec.StartedAt).Milliseconds()

	for key, value := range outputs {
		if !isOutputAttribute(step, key) {
			continue
		}
		if _, ok := ag.Value().Attributes[key]; !ok {
			if err := events.Raise(ag, api.EventTypeAttributeSet,
				api.AttributeSetEvent{
					FlowID: a.flowID,
					StepID: stepID,
					Key:    key,
					Value:  value,
				},
			); err != nil {
				return false, err
			}
		}
	}

	return true, events.Raise(ag, api.EventTypeStepCompleted,
		api.StepCompletedEvent{
			FlowID:   a.flowID,
			StepID:   stepID,
			Outputs:  outputs,
			Duration: dur,
		},
	)
}

// handleWorkNotCompleted handles retry decision for a specific work item
func (a *flowActor) handleWorkNotCompleted(
	ag *FlowAggregator, stepID api.StepID, token api.Token,
) error {
	exec, ok := ag.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil
	}

	workItem, ok := exec.WorkItems[token]
	if !ok || workItem.Status != api.WorkNotCompleted {
		return nil
	}

	step, ok := ag.Value().Plan.Steps[stepID]
	if !ok {
		return nil
	}

	if a.ShouldRetry(step, workItem) {
		nextRetryAt := a.CalculateNextRetry(
			step.WorkConfig, workItem.RetryCount,
		)
		return events.Raise(ag, api.EventTypeRetryScheduled,
			api.RetryScheduledEvent{
				FlowID:      a.flowID,
				StepID:      stepID,
				Token:       token,
				RetryCount:  workItem.RetryCount + 1,
				NextRetryAt: nextRetryAt,
				Error:       workItem.Error,
			},
		)
	}

	// Permanent failure
	return events.Raise(ag, api.EventTypeWorkFailed,
		api.WorkFailedEvent{
			FlowID: a.flowID,
			StepID: stepID,
			Token:  token,
			Error:  workItem.Error,
		},
	)
}

// getDownstreamConsumers returns step IDs that consume any output from the
// given step, using the ExecutionPlan's Attributes dependency map
func (a *flowActor) getDownstreamConsumers(
	stepID api.StepID, flow *api.FlowState,
) []api.StepID {
	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return nil
	}

	seen := util.Set[api.StepID]{}
	var consumers []api.StepID

	for name, attr := range step.Attributes {
		if !attr.IsOutput() {
			continue
		}

		deps := flow.Plan.Attributes[name]
		if deps == nil {
			continue
		}

		for _, consumerID := range deps.Consumers {
			if seen.Contains(consumerID) {
				continue
			}
			seen.Add(consumerID)
			consumers = append(consumers, consumerID)
		}
	}

	return consumers
}

// maybeDeactivate returns a deferred function that emits FlowDeactivated
// if the flow is terminal and has no active work items remaining
func (a *flowActor) maybeDeactivate(flow *api.FlowState) deferred {
	if !flowTransitions.IsTerminal(flow.Status) {
		return nil
	}
	if hasActiveWork(flow) {
		return nil
	}
	return func() {
		a.completeParentWork(flow)
		if err := a.raiseEngineEvent(context.Background(),
			api.EventTypeFlowDeactivated,
			api.FlowDeactivatedEvent{FlowID: a.flowID},
		); err != nil {
			slog.Error("Failed to emit FlowDeactivated",
				log.FlowID(a.flowID),
				log.Error(err))
		}
	}
}

func hasActiveWork(flow *api.FlowState) bool {
	for _, exec := range flow.Executions {
		for _, item := range exec.WorkItems {
			if item.Status == api.WorkPending || item.Status == api.WorkActive {
				return true
			}
		}
	}
	return false
}

func isOutputAttribute(step *api.Step, name api.Name) bool {
	attr, ok := step.Attributes[name]
	return ok && attr.IsOutput()
}
