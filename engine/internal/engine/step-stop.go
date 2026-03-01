package engine

import (
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/util/call"
)

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

	step := flow.Plan.Steps[stepID]
	outputs := tx.collectStepOutputs(exec.WorkItems, step)
	dur := max(tx.Now().Sub(exec.StartedAt).Milliseconds(), int64(0))

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
	if tx.Value().Status == api.FlowActive {
		tx.OnSuccess(func(flow *api.FlowState) {
			tx.scheduleConsumerTimeouts(flow, stepID, tx.Now())
		})
	}
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

	return call.Perform(
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

	return call.Perform(
		tx.checkUnreachable,
		tx.checkTerminal,
		tx.startReadyPendingSteps,
	)
}
