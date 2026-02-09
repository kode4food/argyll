package engine

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/log"
	"github.com/kode4food/argyll/engine/pkg/util"
)

var (
	flowTransitions = StateTransitions[api.FlowStatus]{
		api.FlowActive: util.SetOf(
			api.FlowCompleted,
			api.FlowFailed,
		),
		api.FlowCompleted: {},
		api.FlowFailed:    {},
	}

	workTransitions = StateTransitions[api.WorkStatus]{
		api.WorkPending: util.SetOf(
			api.WorkActive,
		),
		api.WorkActive: util.SetOf(
			api.WorkSucceeded,
			api.WorkFailed,
			api.WorkNotCompleted,
		),
		api.WorkSucceeded: {},
		api.WorkFailed:    {},
		api.WorkNotCompleted: util.SetOf(
			api.WorkActive,
			api.WorkSucceeded,
			api.WorkFailed,
		),
	}
)

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
	state, err := e.execFlow(events.FlowKey(flowID),
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

// CompleteWork marks a work item as successfully completed with the given
// output values
func (e *Engine) CompleteWork(
	fs FlowStep, token api.Token, outputs api.Args,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		err := tx.checkWorkTransition(fs.StepID, token, api.WorkSucceeded)
		if err != nil {
			return err
		}

		if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkSucceeded,
			api.WorkSucceededEvent{
				FlowID:  fs.FlowID,
				StepID:  fs.StepID,
				Token:   token,
				Outputs: outputs,
			},
		); err != nil {
			return err
		}
		tx.OnSuccess(func(flow *api.FlowState) {
			tx.handleWorkSucceededCleanup(fs, token)
			step := flow.Plan.Steps[fs.StepID]
			if step != nil && step.Memoizable {
				work := flow.Executions[fs.StepID].WorkItems[token]
				if work != nil {
					err := e.memoCache.Put(step, work.Inputs, outputs)
					if err != nil {
						slog.Warn("memo cache put failed",
							log.FlowID(fs.FlowID), log.StepID(fs.StepID),
							log.Error(err))
					}
				}
			}
		})
		return tx.handleWorkSucceeded(fs.StepID)
	})
}

// FailWork marks a work item as failed with the specified error message
func (e *Engine) FailWork(fs FlowStep, token api.Token, errMsg string) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		err := tx.checkWorkTransition(fs.StepID, token, api.WorkFailed)
		if err != nil {
			return err
		}

		if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkFailed,
			api.WorkFailedEvent{
				FlowID: fs.FlowID,
				StepID: fs.StepID,
				Token:  token,
				Error:  errMsg,
			},
		); err != nil {
			return err
		}
		return tx.handleWorkFailed(fs.StepID)
	})
}

// NotCompleteWork marks a work item as not completed with specified error
func (e *Engine) NotCompleteWork(
	fs FlowStep, token api.Token, errMsg string,
) error {
	return e.flowTx(fs.FlowID, func(tx *flowTx) error {
		err := tx.checkWorkTransition(fs.StepID, token, api.WorkNotCompleted)
		if err != nil {
			return err
		}

		var retryToken api.Token
		if exec, ok := tx.Value().Executions[fs.StepID]; ok {
			if item := exec.WorkItems[token]; item != nil {
				step := tx.Value().Plan.Steps[fs.StepID]
				if step != nil && !step.Memoizable && item.RetryCount > 0 {
					retryToken = api.Token(uuid.New().String())
				}
			}
		}

		if err := events.Raise(tx.FlowAggregator, api.EventTypeWorkNotCompleted,
			api.WorkNotCompletedEvent{
				FlowID:     fs.FlowID,
				StepID:     fs.StepID,
				Token:      token,
				RetryToken: retryToken,
				Error:      errMsg,
			},
		); err != nil {
			return err
		}

		actualToken := token
		if retryToken != "" {
			actualToken = retryToken
		}
		return tx.handleWorkNotCompleted(fs.StepID, actualToken)
	})
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

func (e *Engine) execFlow(
	flowID timebox.AggregateID, cmd timebox.Command[*api.FlowState],
) (*api.FlowState, error) {
	return e.flowExec.Exec(e.ctx, flowID, cmd)
}

func (tx *flowTx) handleWorkSucceededCleanup(fs FlowStep, token api.Token) {
	tx.retryQueue.Remove(fs.FlowID, fs.StepID, token)
}

func (tx *flowTx) checkWorkTransition(
	stepID api.StepID, token api.Token, toStatus api.WorkStatus,
) error {
	flow := tx.Value()
	exec, ok := flow.Executions[stepID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrStepNotInPlan, stepID)
	}

	work, ok := exec.WorkItems[token]
	if !ok {
		return fmt.Errorf("%w: %s", ErrWorkItemNotFound, token)
	}

	if !workTransitions.CanTransition(work.Status, toStatus) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidWorkTransition,
			work.Status, toStatus)
	}

	return nil
}
