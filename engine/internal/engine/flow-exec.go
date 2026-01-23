package engine

import (
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type flowActor struct {
	*Engine
	flowID api.FlowID
	events chan *timebox.Event
}

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

// execTransaction executes a function within a flow transaction
func (a *flowActor) execTransaction(fn func(ag *FlowAggregator) error) error {
	_, err := a.execFlow(flowKey(a.flowID),
		func(_ *api.FlowState, ag *FlowAggregator) error {
			return fn(ag)
		},
	)
	return err
}

// prepareStep validates and prepares a step for execution within a
// transaction, raising the StepStarted event via aggregator and scheduling
// work execution after commit
func (a *flowActor) prepareStep(
	stepID api.StepID, ag *FlowAggregator,
) error {
	flow := ag.Value()

	// Validate step exists and is pending
	exec, ok := flow.Executions[stepID]
	if ok && exec.Status != api.StepPending {
		return fmt.Errorf("%w: %s (status=%s)",
			ErrStepAlreadyPending, stepID, exec.Status)
	}

	step, ok := flow.Plan.Steps[stepID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
	}

	// Collect inputs
	inputs := a.collectStepInputs(step, flow.GetAttributes())

	// Evaluate predicate
	shouldExecute, err := a.evaluateStepPredicate(step, inputs)
	if err != nil {
		return a.handlePredicateFailure(ag, stepID, err)
	}
	if !shouldExecute {
		// Predicate failed - skip this step
		if err := events.Raise(ag, api.EventTypeStepSkipped,
			api.StepSkippedEvent{
				FlowID: a.flowID,
				StepID: stepID,
				Reason: "predicate returned false",
			},
		); err != nil {
			return err
		}
		return nil
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
		return err
	}

	for token, workInputs := range workItemsMap {
		shouldStart, err := a.evaluateStepPredicate(step, workInputs)
		if err != nil {
			return a.handlePredicateFailure(ag, stepID, err)
		}
		if !shouldStart {
			continue
		}
		if err := events.Raise(ag, api.EventTypeWorkStarted,
			api.WorkStartedEvent{
				FlowID: a.flowID,
				StepID: stepID,
				Token:  token,
				Inputs: workInputs,
			},
		); err != nil {
			return err
		}
	}

	flow = ag.Value()
	items := flow.Executions[stepID].WorkItems

	ag.OnSuccess(func() {
		execCtx := &ExecContext{
			engine: a.Engine,
			flowID: a.flowID,
			stepID: stepID,
			step:   step,
			inputs: inputs,
			meta:   flow.Metadata,
		}
		execCtx.executeWorkItems(items)
	})

	return nil
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
			break
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
			break
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

func (a *flowActor) handlePredicateFailure(
	ag *FlowAggregator, stepID api.StepID, err error,
) error {
	if raiseErr := events.Raise(ag, api.EventTypeStepFailed,
		api.StepFailedEvent{
			FlowID: a.flowID,
			StepID: stepID,
			Error:  err.Error(),
		},
	); raiseErr != nil {
		return raiseErr
	}

	if failErr := a.failUnreachable(ag); failErr != nil {
		return failErr
	}
	if termErr := a.checkTerminal(ag); termErr != nil {
		return termErr
	}
	a.maybeDeactivate(ag)
	return nil
}

// handleStepFailure handles common failure logic for work failure paths,
// checking step completion and propagating failures
func (a *flowActor) handleStepFailure(
	ag *FlowAggregator, stepID api.StepID,
) error {
	if flowTransitions.IsTerminal(ag.Value().Status) {
		_, err := a.checkStepCompletion(ag, stepID)
		if err != nil {
			return err
		}
		a.maybeDeactivate(ag)
		return nil
	}

	completed, err := a.checkStepCompletion(ag, stepID)
	if err != nil || !completed {
		return err
	}

	if err := a.failUnreachable(ag); err != nil {
		return err
	}
	return a.checkTerminal(ag)
}

// scheduleRetry handles retry decision for a specific work item
func (a *flowActor) scheduleRetry(
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

func (a *flowActor) handleWorkSucceeded(
	ag *FlowAggregator, stepID api.StepID,
) error {
	// Terminal flows only record step completions for audit
	if flowTransitions.IsTerminal(ag.Value().Status) {
		_, err := a.checkStepCompletion(ag, stepID)
		if err != nil {
			return err
		}
		a.maybeDeactivate(ag)
		return nil
	}

	completed, err := a.checkStepCompletion(ag, stepID)
	if err != nil || !completed {
		return err
	}

	if err := a.skipPendingUnused(ag); err != nil {
		return err
	}

	// Step completed - check if it was a goal step
	if a.isGoalStep(stepID, ag.Value()) {
		if err := a.checkTerminal(ag); err != nil {
			return err
		}
		a.maybeDeactivate(ag)
		return nil
	}

	// Find and start downstream ready steps
	for _, consumerID := range a.findReadySteps(stepID, ag.Value()) {
		err := a.prepareStep(consumerID, ag)
		if err != nil {
			slog.Warn("Failed to prepare step",
				log.StepID(consumerID),
				log.Error(err))
			continue
		}
	}
	return nil
}

func (a *flowActor) handleWorkFailed(
	ag *FlowAggregator, stepID api.StepID,
) error {
	return a.handleStepFailure(ag, stepID)
}

func (a *flowActor) handleWorkNotCompleted(
	ag *FlowAggregator, stepID api.StepID, token api.Token,
) error {
	if flowTransitions.IsTerminal(ag.Value().Status) {
		a.maybeDeactivate(ag)
		return nil
	}
	if err := a.scheduleRetry(ag, stepID, token); err != nil {
		return err
	}
	return a.handleStepFailure(ag, stepID)
}

// maybeDeactivate emits FlowDeactivated after commit if the flow is terminal
// and has no active work items remaining
func (a *flowActor) maybeDeactivate(ag *FlowAggregator) {
	flow := ag.Value()
	if !flowTransitions.IsTerminal(flow.Status) {
		return
	}
	if hasActiveWork(flow) {
		return
	}
	ag.OnSuccess(func() {
		a.completeParentWork(flow)
		if err := a.raiseEngineEvent(
			api.EventTypeFlowDeactivated,
			api.FlowDeactivatedEvent{FlowID: a.flowID},
		); err != nil {
			slog.Error("Failed to emit FlowDeactivated",
				log.FlowID(a.flowID),
				log.Error(err))
		}
	})
}
