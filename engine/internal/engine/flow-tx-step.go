package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

type flowTx struct {
	*Engine
	*FlowAggregator
	flowID api.FlowID
}

var (
	ErrInvariantViolated = errors.New("engine invariant violated")
)

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
	inputs := tx.collectStepInputs(step, flow)

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
		return performCalls(
			tx.checkUnreachable,
			tx.checkTerminal,
			tx.startReadyPendingSteps,
		)
	}

	// Compute work items
	workItemsList, err := computeWorkItems(step, inputs)
	if err != nil {
		return err
	}
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

	started, err := tx.startPendingWork(step)
	if err != nil {
		return err
	}

	if len(started) > 0 {
		tx.OnSuccess(func(flow *api.FlowState) {
			tx.Engine.CancelScheduledTaskPrefix(
				timeoutStepPrefix(api.FlowStep{
					FlowID: tx.flowID,
					StepID: step.ID,
				}),
			)
			tx.handleWorkItemsExecution(step, inputs, flow.Metadata, started)
		})
	}

	return nil
}

func (tx *flowTx) handleWorkItemsExecution(
	step *api.Step, inputs api.Args, meta api.Metadata, items api.WorkItems,
) {
	execCtx := &ExecContext{
		engine: tx.Engine,
		flowID: tx.flowID,
		stepID: step.ID,
		step:   step,
		inputs: inputs,
		meta:   meta,
	}
	execCtx.executeWorkItems(items)
}

// checkStepCompletion checks if a specific step can complete (all work items
// done) and raises appropriate completion or failure events
func (tx *flowTx) checkStepCompletion(stepID api.StepID) (bool, error) {
	flow := tx.Value()
	exec, ok := flow.Executions[stepID]
	if !ok || exec.Status != api.StepActive {
		return false, fmt.Errorf("%w: expected %s to be active, got %s",
			ErrInvariantViolated, stepID, exec.Status)
	}

	allDone := true
	hasFailed := false
	var failureError string

	for _, work := range exec.WorkItems {
		switch work.Status {
		case api.WorkSucceeded:
			// continue
		case api.WorkFailed:
			hasFailed = true
			if failureError == "" {
				failureError = work.Error
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
	step := flow.Plan.Steps[stepID]
	outputs := tx.collectStepOutputs(exec.WorkItems, step)
	dur := time.Since(exec.StartedAt).Milliseconds()

	for key, value := range outputs {
		if !isOutputAttribute(step, key) {
			continue
		}
		if _, ok := flow.Attributes[key]; ok {
			continue
		}
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
		flow = tx.Value()
	}

	if err := events.Raise(tx.FlowAggregator, api.EventTypeStepCompleted,
		api.StepCompletedEvent{
			FlowID:   tx.flowID,
			StepID:   stepID,
			Outputs:  outputs,
			Duration: dur,
		},
	); err != nil {
		return true, err
	}
	tx.OnSuccess(func(flow *api.FlowState) {
		tx.Engine.scheduleConsumerTimeouts(flow, stepID, time.Now())
	})
	return true, nil
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

	return performCalls(
		tx.checkUnreachable,
		tx.checkTerminal,
	)
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
		return tx.continueStepWork(stepID, false)
	}

	return performCalls(
		tx.checkUnreachable,
		tx.checkTerminal,
		tx.startReadyPendingSteps,
	)
}
