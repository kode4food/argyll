package engine

import (
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

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
			tx.Engine.CancelScheduledTaskPrefix(retryTaskPrefix(tx.flowID))
			tx.Engine.CancelScheduledTaskPrefix(timeoutTaskFlowPrefix(tx.flowID))
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
			tx.Engine.CancelScheduledTaskPrefix(retryTaskPrefix(tx.flowID))
			tx.Engine.CancelScheduledTaskPrefix(timeoutTaskFlowPrefix(tx.flowID))
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

// checkUnreachable finds and fails all pending steps that can no longer
// complete because their required inputs cannot be satisfied
func (tx *flowTx) checkUnreachable() error {
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

func (tx *flowTx) startReadyPendingSteps() error {
	if flowTransitions.IsTerminal(tx.Value().Status) {
		return nil
	}

	for {
		flow := tx.Value()
		now := time.Now()
		startedAny := false

		for stepID, exec := range flow.Executions {
			if exec.Status != api.StepPending {
				continue
			}

			ready, _ := tx.canStartStepAt(stepID, flow, now)
			if !ready {
				continue
			}

			if err := tx.prepareStep(stepID); err != nil {
				if errors.Is(err, ErrStepAlreadyPending) {
					continue
				}
				return err
			}
			startedAny = true
			break
		}

		if !startedAny || flowTransitions.IsTerminal(tx.Value().Status) {
			return nil
		}
	}
}
