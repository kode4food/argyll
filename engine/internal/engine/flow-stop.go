package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
)

type parentWork struct {
	fs    api.FlowStep
	token api.Token
}

var (
	ErrFlowOutputMissing     = errors.New("flow output missing")
	ErrPartialParentMetadata = errors.New("partial parent metadata")
)

var (
	getMetaFlowID = api.GetMetaString[api.FlowID]
	getMetaStepID = api.GetMetaString[api.StepID]
	getMetaToken  = api.GetMetaString[api.Token]
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
		tx.OnSuccess(func(flow *api.FlowState) {
			completedAt := flow.CompletedAt
			if completedAt.IsZero() {
				completedAt = tx.Now()
			}
			if flowHasRetryTasks(flow) {
				tx.CancelPrefixedTasks(retryPrefix(tx.flowID))
			}
			if flowHasTimeouts(flow) {
				tx.CancelPrefixedTasks(timeoutFlowPrefix(tx.flowID))
			}
			tx.EnqueueEvent(api.EventTypeFlowDigestUpdated,
				api.FlowDigestUpdatedEvent{
					FlowID:      tx.flowID,
					Status:      api.FlowCompleted,
					CompletedAt: completedAt,
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
		tx.OnSuccess(func(flow *api.FlowState) {
			completedAt := flow.CompletedAt
			if completedAt.IsZero() {
				completedAt = tx.Now()
			}
			if flowHasRetryTasks(flow) {
				tx.CancelPrefixedTasks(retryPrefix(tx.flowID))
			}
			if flowHasTimeouts(flow) {
				tx.CancelPrefixedTasks(timeoutFlowPrefix(tx.flowID))
			}
			tx.EnqueueEvent(api.EventTypeFlowDigestUpdated,
				api.FlowDigestUpdatedEvent{
					FlowID:      tx.flowID,
					Status:      api.FlowFailed,
					CompletedAt: completedAt,
					Error:       errMsg,
				},
			)
		})
		return tx.maybeDeactivate()
	}
	return nil
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

func (tx *flowTx) completeParentWork(st *api.FlowState) {
	target := &parentWork{}
	ok, err := tx.parentMeta(st, target)
	if !ok || err != nil {
		if err != nil {
			slog.Error("Failed to resolve parent work item",
				log.FlowID(tx.flowID),
				log.Error(err))
		}
		return
	}
	if st.Status != api.FlowCompleted && st.Status != api.FlowFailed {
		return
	}

	if err := tx.completeParentFlowWork(st, target); err != nil {
		slog.Error("Failed to update parent work item",
			log.FlowID(tx.flowID),
			log.Error(err))
	}
}

func (tx *flowTx) completeParentFlowWork(
	child *api.FlowState, target *parentWork,
) error {
	return tx.flowTx(target.fs.FlowID, func(parentTx *flowTx) error {
		parent := parentTx.Value()
		if parent.ID == "" {
			return fmt.Errorf("%w: %w", ErrGetFlowState, ErrFlowNotFound)
		}

		exec := parent.Executions[target.fs.StepID]
		work := exec.WorkItems[target.token]
		if isWorkTerminal(work.Status) {
			return nil
		}

		if child.Status == api.FlowCompleted {
			outputs, err := mapFlowOutputs(
				parent.Plan.Steps[target.fs.StepID], child.GetAttributes(),
			)
			if err != nil {
				return parentTx.failWork(
					target.fs.StepID, target.token, err.Error(),
				)
			}
			return parentTx.completeWork(
				target.fs.StepID, target.token, outputs,
			)
		}

		errMsg := child.Error
		if errMsg == "" {
			errMsg = "child flow failed"
		}
		return parentTx.failWork(target.fs.StepID, target.token, errMsg)
	})
}

func (tx *flowTx) parentMeta(
	st *api.FlowState, target *parentWork,
) (bool, error) {
	if err := validateParentMetadata(st.Metadata); err != nil {
		return false, fmt.Errorf("%w: %s", err, st.ID)
	}

	flowID, hasFlowID := getMetaFlowID(st.Metadata, api.MetaParentFlowID)
	stepID, hasStepID := getMetaStepID(st.Metadata, api.MetaParentStepID)
	token, hasToken := getMetaToken(st.Metadata, api.MetaParentWorkItemToken)

	if !hasFlowID && !hasStepID && !hasToken {
		return false, nil
	}

	target.fs = api.FlowStep{FlowID: flowID, StepID: stepID}
	target.token = token
	return true, nil
}

func isWorkTerminal(status api.WorkStatus) bool {
	return status == api.WorkSucceeded || status == api.WorkFailed
}

func flowHasRetryTasks(flow *api.FlowState) bool {
	for _, exec := range flow.Executions {
		for _, work := range exec.WorkItems {
			if !work.NextRetryAt.IsZero() {
				return true
			}
		}
	}
	return false
}

func mapFlowOutputs(step *api.Step, childAttrs api.Args) (api.Args, error) {
	outputs := maps.Clone(childAttrs)

	for _, attr := range step.Attributes {
		if !attr.IsOutput() {
			continue
		}
		if attr.Mapping == nil || attr.Mapping.Name == "" {
			continue
		}

		sourceName := api.Name(attr.Mapping.Name)
		value, ok := childAttrs[sourceName]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrFlowOutputMissing, sourceName)
		}
		outputs[sourceName] = value
	}

	return outputs, nil
}

func validateParentMetadata(meta api.Metadata) error {
	_, hasFlowID := getMetaFlowID(meta, api.MetaParentFlowID)
	_, hasStepID := getMetaStepID(meta, api.MetaParentStepID)
	_, hasToken := getMetaToken(meta, api.MetaParentWorkItemToken)
	if !hasFlowID && !hasStepID && !hasToken {
		return nil
	}
	if hasFlowID && hasStepID && hasToken {
		return nil
	}
	return ErrPartialParentMetadata
}
