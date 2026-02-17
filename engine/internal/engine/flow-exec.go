package engine

import (
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/google/uuid"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type flowTx struct {
	*Engine
	*FlowAggregator
	flowID api.FlowID
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

// prepareStep validates and prepares a step to execute within a transaction,
// raising the StepStarted event via aggregator and scheduling work execution
// after commit
func (tx *flowTx) prepareStep(stepID api.StepID) error {
	flow := tx.Value()

	// Validate step is pending
	exec := flow.Executions[stepID]
	if exec.Status != api.StepPending {
		return fmt.Errorf("%w: %s (status=%s)",
			ErrStepAlreadyPending, stepID, exec.Status)
	}

	step := flow.Plan.Steps[stepID]

	// Collect inputs
	inputs := tx.collectStepInputs(step, flow.GetAttributes())

	// Evaluate predicate
	shouldExecute, err := tx.evaluateStepPredicate(step, inputs)
	if err != nil {
		return tx.handlePredicateFailure(stepID, err)
	}
	if !shouldExecute {
		// Predicate failed - skip this step
		if err := events.Raise(tx.FlowAggregator, api.EventTypeStepSkipped,
			api.StepSkippedEvent{
				FlowID: tx.flowID,
				StepID: stepID,
				Reason: "predicate returned false",
			},
		); err != nil {
			return err
		}
		if err := tx.failUnreachable(); err != nil {
			return err
		}
		return tx.checkTerminal()
	}

	// Compute work items
	workItemsList := computeWorkItems(step, inputs)
	workItemsMap := map[api.Token]api.Args{}
	for _, workInputs := range workItemsList {
		token := api.Token(uuid.New().String())
		workItemsMap[token] = workInputs
	}

	// Raise StepStarted event with work items
	if err := events.Raise(tx.FlowAggregator, api.EventTypeStepStarted,
		api.StepStartedEvent{
			FlowID:    tx.flowID,
			StepID:    stepID,
			Inputs:    inputs,
			WorkItems: workItemsMap,
		},
	); err != nil {
		return err
	}

	started, err := tx.startPendingWork(stepID, step)
	if err != nil {
		return err
	}

	if len(started) > 0 {
		tx.OnSuccess(func(flow *api.FlowState) {
			tx.handleWorkItemsExecution(
				stepID, step, inputs, flow.Metadata, started,
			)
		})
	}

	return nil
}

func (tx *flowTx) handleWorkItemsExecution(
	stepID api.StepID, step *api.Step, inputs api.Args, meta api.Metadata,
	items api.WorkItems,
) {
	execCtx := &ExecContext{
		engine: tx.Engine,
		flowID: tx.flowID,
		stepID: stepID,
		step:   step,
		inputs: inputs,
		meta:   meta,
	}
	execCtx.executeWorkItems(items)
}

// checkTerminal checks for flow completion or failure
func (tx *flowTx) checkTerminal() error {
	flow := tx.Value()
	if tx.isFlowComplete(flow) {
		result := api.Args{}
		for _, goalID := range flow.Plan.Goals {
			if goal := flow.Executions[goalID]; goal != nil {
				maps.Copy(result, goal.Outputs)
			}
		}
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowCompleted,
			api.FlowCompletedEvent{
				FlowID: tx.flowID,
				Result: result,
			},
		); err != nil {
			return err
		}
		tx.OnSuccess(func(*api.FlowState) {
			tx.retryQueue.RemoveFlow(tx.flowID)
			tx.EnqueueEvent(api.EventTypeFlowDigestUpdated,
				api.FlowDigestUpdatedEvent{
					FlowID:      tx.flowID,
					Status:      api.FlowCompleted,
					CompletedAt: time.Now(),
				},
			)
		})
		return tx.maybeDeactivate()
	}
	if tx.IsFlowFailed(flow) {
		errMsg := tx.getFailureReason(flow)
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowFailed,
			api.FlowFailedEvent{
				FlowID: tx.flowID,
				Error:  errMsg,
			},
		); err != nil {
			return err
		}
		tx.OnSuccess(func(*api.FlowState) {
			tx.retryQueue.RemoveFlow(tx.flowID)
			tx.EnqueueEvent(api.EventTypeFlowDigestUpdated,
				api.FlowDigestUpdatedEvent{
					FlowID:      tx.flowID,
					Status:      api.FlowFailed,
					CompletedAt: time.Now(),
					Error:       errMsg,
				},
			)
		})
		return tx.maybeDeactivate()
	}
	return nil
}

// failUnreachable finds and fails all pending steps that can no longer
// complete because their required inputs cannot be satisfied
func (tx *flowTx) failUnreachable() error {
	for {
		failedAny := false
		flow := tx.Value()

		for stepID, exec := range flow.Executions {
			if exec.Status != api.StepPending {
				continue
			}
			if tx.canStepComplete(stepID, flow) {
				continue
			}

			if err := events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
				api.StepFailedEvent{
					FlowID: tx.flowID,
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

func (tx *flowTx) skipPendingUnused() error {
	for {
		skip := false
		flow := tx.Value()

		for stepID, exec := range flow.Executions {
			if exec.Status != api.StepPending {
				continue
			}
			if tx.areOutputsNeeded(stepID, flow) {
				continue
			}

			if err := events.Raise(tx.FlowAggregator, api.EventTypeStepSkipped,
				api.StepSkippedEvent{
					FlowID: tx.flowID,
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
func (tx *flowTx) getFailureReason(flow *api.FlowState) string {
	for stepID, exec := range flow.Executions {
		if exec.Status == api.StepFailed {
			return fmt.Sprintf("step %s failed: %s", stepID, exec.Error)
		}
	}
	return "flow failed"
}

// checkStepCompletion checks if a specific step can complete (all work items
// done) and raises appropriate completion or failure events
func (tx *flowTx) checkStepCompletion(stepID api.StepID) (bool, error) {
	exec, ok := tx.Value().Executions[stepID]
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
		return true, events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
			api.StepFailedEvent{
				FlowID: tx.flowID,
				StepID: stepID,
				Error:  failureError,
			},
		)
	}

	// Step succeeded - set attributes and raise completion
	step := tx.Value().Plan.Steps[stepID]
	outputs := tx.collectStepOutputs(exec.WorkItems, step)
	dur := time.Since(exec.StartedAt).Milliseconds()

	for key, value := range outputs {
		if !isOutputAttribute(step, key) {
			continue
		}
		if _, ok := tx.Value().Attributes[key]; !ok {
			if err := events.Raise(tx.FlowAggregator, api.EventTypeAttributeSet,
				api.AttributeSetEvent{
					FlowID: tx.flowID,
					StepID: stepID,
					Key:    key,
					Value:  value,
				},
			); err != nil {
				return false, err
			}
		}
	}

	return true, events.Raise(tx.FlowAggregator, api.EventTypeStepCompleted,
		api.StepCompletedEvent{
			FlowID:   tx.flowID,
			StepID:   stepID,
			Outputs:  outputs,
			Duration: dur,
		},
	)
}

func (tx *flowTx) handlePredicateFailure(stepID api.StepID, err error) error {
	if raiseErr := events.Raise(tx.FlowAggregator, api.EventTypeStepFailed,
		api.StepFailedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Error:  err.Error(),
		},
	); raiseErr != nil {
		return raiseErr
	}

	if failErr := tx.failUnreachable(); failErr != nil {
		return failErr
	}
	if termErr := tx.checkTerminal(); termErr != nil {
		return termErr
	}
	return nil
}

// handleStepFailure handles common failure logic for work failure paths,
// checking step completion and propagating failures
func (tx *flowTx) handleStepFailure(stepID api.StepID) error {
	if flowTransitions.IsTerminal(tx.Value().Status) {
		_, err := tx.checkStepCompletion(stepID)
		if err != nil {
			return err
		}
		return tx.maybeDeactivate()
	}

	completed, err := tx.checkStepCompletion(stepID)
	if err != nil || !completed {
		if err != nil {
			return err
		}
		step := tx.Value().Plan.Steps[stepID]
		started, err := tx.startPendingWork(stepID, step)
		if err != nil {
			return err
		}
		if len(started) == 0 {
			return nil
		}
		tx.OnSuccess(func(flow *api.FlowState) {
			exec := flow.Executions[stepID]
			tx.handleWorkItemsExecution(
				stepID, step, exec.Inputs, flow.Metadata, started,
			)
		})
		return nil
	}

	if err := tx.failUnreachable(); err != nil {
		return err
	}
	return tx.checkTerminal()
}

// scheduleRetry handles retry decision for a specific work item
func (tx *flowTx) scheduleRetry(stepID api.StepID, token api.Token) error {
	exec, ok := tx.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil
	}

	workItem, ok := exec.WorkItems[token]
	if !ok || workItem.Status != api.WorkNotCompleted {
		return nil
	}

	step := tx.Value().Plan.Steps[stepID]

	if tx.ShouldRetry(step, workItem) {
		nextRetryAt := tx.CalculateNextRetry(
			step.WorkConfig, workItem.RetryCount,
		)
		if err := events.Raise(tx.FlowAggregator, api.EventTypeRetryScheduled,
			api.RetryScheduledEvent{
				FlowID:      tx.flowID,
				StepID:      stepID,
				Token:       token,
				RetryCount:  workItem.RetryCount + 1,
				NextRetryAt: nextRetryAt,
				Error:       workItem.Error,
			},
		); err != nil {
			return err
		}
		tx.OnSuccess(func(*api.FlowState) {
			tx.handleRetryScheduled(stepID, token, nextRetryAt)
		})
		return nil
	}

	// Permanent failure
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkFailed,
		api.WorkFailedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  token,
			Error:  workItem.Error,
		},
	)
}

func (tx *flowTx) handleWorkSucceeded(stepID api.StepID) error {
	// Terminal flows only record step completions for audit
	if flowTransitions.IsTerminal(tx.Value().Status) {
		_, err := tx.checkStepCompletion(stepID)
		if err != nil {
			return err
		}
		return tx.maybeDeactivate()
	}

	completed, err := tx.checkStepCompletion(stepID)
	if err != nil {
		return err
	}
	if !completed {
		return tx.handleWorkContinuation(stepID)
	}

	if err := tx.skipPendingUnused(); err != nil {
		return err
	}

	// Step completed - check if it was a goal step
	if tx.isGoalStep(stepID, tx.Value()) {
		if err := tx.checkTerminal(); err != nil {
			return err
		}
		return nil
	}

	// Find and start downstream ready steps
	for _, consumerID := range tx.findReadySteps(stepID, tx.Value()) {
		err := tx.prepareStep(consumerID)
		if err != nil {
			slog.Warn("Failed to prepare step",
				log.StepID(consumerID),
				log.Error(err))
			continue
		}
	}
	return nil
}

func (tx *flowTx) handleWorkContinuation(stepID api.StepID) error {
	step := tx.Value().Plan.Steps[stepID]
	started, err := tx.startPendingWork(stepID, step)
	if err != nil {
		return err
	}
	tx.handleWorkItems(stepID, step, started)
	return nil
}

func (tx *flowTx) handleWorkFailed(stepID api.StepID) error {
	step := tx.Value().Plan.Steps[stepID]
	started, err := tx.startPendingWork(stepID, step)
	if err != nil {
		return err
	}
	tx.handleWorkItems(stepID, step, started)
	return tx.handleStepFailure(stepID)
}

func (tx *flowTx) handleWorkNotCompleted(
	stepID api.StepID, token api.Token,
) error {
	if flowTransitions.IsTerminal(tx.Value().Status) {
		return tx.maybeDeactivate()
	}
	if err := tx.scheduleRetry(stepID, token); err != nil {
		return err
	}
	step := tx.Value().Plan.Steps[stepID]
	started, err := tx.startPendingWork(stepID, step)
	if err != nil {
		return err
	}
	tx.handleWorkItems(stepID, step, started)
	return tx.handleStepFailure(stepID)
}

func (tx *flowTx) startPendingWork(
	stepID api.StepID, step *api.Step,
) (api.WorkItems, error) {
	exec, ok := tx.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil, nil
	}

	limit := stepParallelism(step)
	active := countActiveWorkItems(exec.WorkItems)

	remaining := limit - active
	if remaining <= 0 {
		return nil, nil
	}

	now := time.Now()
	started := api.WorkItems{}
	for token, item := range exec.WorkItems {
		if remaining == 0 {
			break
		}
		shouldStart, err := tx.shouldStartPendingWorkItem(
			stepID, step, item, now,
		)
		if err != nil {
			return nil, err
		}
		if !shouldStart {
			continue
		}

		if step.Memoizable {
			if cached, ok := tx.Engine.memoCache.Get(step, item.Inputs); ok {
				err := tx.handleMemoCacheHit(stepID, token, cached)
				if err != nil {
					return nil, err
				}
				remaining--
				continue
			}
		}

		if err := tx.raiseWorkStarted(
			stepID, token, item.Inputs,
		); err != nil {
			return nil, err
		}
		exec = tx.Value().Executions[stepID]
		started[token] = exec.WorkItems[token]
		remaining--
	}

	return started, nil
}

func (tx *flowTx) startRetryWorkItem(
	stepID api.StepID, step *api.Step, token api.Token,
) (api.WorkItems, error) {
	exec, ok := tx.Value().Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return nil, nil
	}

	item := exec.WorkItems[token]
	if item == nil {
		return nil, nil
	}

	now := time.Now()
	shouldStart := false
	switch item.Status {
	case api.WorkPending:
		var err error
		shouldStart, err = tx.shouldStartRetryPending(
			stepID, step, item, exec.WorkItems, now,
		)
		if err != nil {
			return nil, err
		}
	case api.WorkFailed:
		if item.NextRetryAt.IsZero() || item.NextRetryAt.After(now) {
			return nil, nil
		}
		shouldStart = true
	case api.WorkActive, api.WorkNotCompleted:
		shouldStart = true
	default:
		return nil, nil
	}
	if !shouldStart {
		return nil, nil
	}

	if err := tx.raiseWorkStarted(
		stepID, token, item.Inputs,
	); err != nil {
		return nil, err
	}
	exec = tx.Value().Executions[stepID]
	started := api.WorkItems{}
	started[token] = exec.WorkItems[token]
	return started, nil
}

func (tx *flowTx) handleWorkItems(
	stepID api.StepID, step *api.Step, started api.WorkItems,
) {
	if len(started) == 0 {
		return
	}
	tx.OnSuccess(func(flow *api.FlowState) {
		exec := flow.Executions[stepID]
		for token := range started {
			tx.retryQueue.Remove(tx.flowID, stepID, token)
		}
		tx.handleWorkItemsExecution(
			stepID, step, exec.Inputs, flow.Metadata, started,
		)
	})
}

func (tx *flowTx) shouldStartPendingWorkItem(
	stepID api.StepID, step *api.Step, item *api.WorkState, now time.Time,
) (bool, error) {
	if item.Status != api.WorkPending {
		return false, nil
	}
	if !item.NextRetryAt.IsZero() && item.NextRetryAt.After(now) {
		return false, nil
	}
	shouldStart, err := tx.evaluateStepPredicate(step, item.Inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(stepID, err)
	}
	return shouldStart, nil
}

func (tx *flowTx) shouldStartRetryPending(
	stepID api.StepID, step *api.Step, item *api.WorkState, items api.WorkItems,
	now time.Time,
) (bool, error) {
	if !item.NextRetryAt.IsZero() && item.NextRetryAt.After(now) {
		return false, nil
	}
	limit := stepParallelism(step)
	active := countActiveWorkItems(items)
	if active >= limit {
		return false, nil
	}
	shouldStart, err := tx.evaluateStepPredicate(step, item.Inputs)
	if err != nil {
		return false, tx.handlePredicateFailure(stepID, err)
	}
	return shouldStart, nil
}

func (tx *flowTx) raiseWorkStarted(
	stepID api.StepID, token api.Token, inputs api.Args,
) error {
	return events.Raise(tx.FlowAggregator, api.EventTypeWorkStarted,
		api.WorkStartedEvent{
			FlowID: tx.flowID,
			StepID: stepID,
			Token:  token,
			Inputs: inputs,
		},
	)
}

// handleMemoCacheHit processes a memo cache hit by emitting WorkSucceeded
func (tx *flowTx) handleMemoCacheHit(
	stepID api.StepID, token api.Token, outputs api.Args,
) error {
	if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
		api.WorkSucceededEvent{
			FlowID:  tx.flowID,
			StepID:  stepID,
			Token:   token,
			Outputs: outputs,
		},
	); err != nil {
		return err
	}
	tx.OnSuccess(func(flow *api.FlowState) {
		fs := api.FlowStep{FlowID: tx.flowID, StepID: stepID}
		tx.handleWorkSucceededCleanup(fs, token)
	})
	return tx.handleWorkSucceeded(stepID)
}

// maybeDeactivate emits FlowDeactivated if the flow is terminal and has no
// active work items remaining
func (tx *flowTx) maybeDeactivate() error {
	flow := tx.Value()
	if !flowTransitions.IsTerminal(flow.Status) {
		return nil
	}
	if hasActiveWork(flow) {
		return nil
	}
	tx.OnSuccess(func(flow *api.FlowState) {
		tx.completeParentWork(flow)
		tx.EnqueueEvent(api.EventTypeFlowDeactivated,
			api.FlowDeactivatedEvent{FlowID: tx.flowID},
		)
	})
	return nil
}

func (tx *flowTx) handleRetryScheduled(
	stepID api.StepID, token api.Token, nextRetryAt time.Time,
) {
	tx.retryQueue.Push(&RetryItem{
		FlowID:      tx.flowID,
		StepID:      stepID,
		Token:       token,
		NextRetryAt: nextRetryAt,
	})
}

func stepParallelism(step *api.Step) int {
	if step.WorkConfig == nil || step.WorkConfig.Parallelism <= 0 {
		return 1
	}
	return step.WorkConfig.Parallelism
}

func countActiveWorkItems(items api.WorkItems) int {
	active := 0
	for _, item := range items {
		if item.Status == api.WorkActive {
			active++
		}
	}
	return active
}
