package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/policy"
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

// checkTerminal checks for flow completion or failure
func (tx *flowTx) checkTerminal() error {
	fl := tx.Value()
	if isFlowComplete(fl) {
		result := api.Args{}
		for _, goalID := range fl.Plan.Goals {
			goal := fl.Executions[goalID]
			maps.Copy(result, goal.Outputs)
		}
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowCompleted,
			api.FlowCompletedEvent{
				FlowID: tx.flowID,
				Result: result,
			},
		); err != nil {
			return err
		}
		tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
			if flowHasRetryTasks(fl) {
				tx.CancelPrefixedTasks(retryPrefix(tx.flowID))
			}
			if flowHasTimeouts(fl) {
				tx.CancelPrefixedTasks(timeoutFlowPrefix(tx.flowID))
			}
		})
		return tx.maybeDeactivate()
	}
	if tx.IsFlowFailed(fl) {
		errMsg := getFailureReason(fl)
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowFailed,
			api.FlowFailedEvent{
				FlowID: tx.flowID,
				Error:  errMsg,
			},
		); err != nil {
			return err
		}
		tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
			if flowHasRetryTasks(fl) {
				tx.CancelPrefixedTasks(retryPrefix(tx.flowID))
			}
			if flowHasTimeouts(fl) {
				tx.CancelPrefixedTasks(timeoutFlowPrefix(tx.flowID))
			}
		})
		return tx.maybeDeactivate()
	}
	return nil
}

// maybeDeactivate reports the outcome to the parent, then deactivates once
// the parent can no longer order a rollback
func (tx *flowTx) maybeDeactivate() error {
	if !policy.FlowTerminal(tx.Value().Status) {
		return nil
	}
	// Compensation may start new work, so sweep before testing for active
	// work rather than after
	if err := tx.compensateFlow(); err != nil {
		return err
	}
	if hasActiveWork(tx.Value()) {
		return nil
	}
	// Told before deactivating, since the parent decides the rollback
	tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
		tx.completeParentWork(fl)
	})
	return tx.deactivate()
}

func (tx *flowTx) deactivate() error {
	fl := tx.Value()
	if !fl.DeactivatedAt.IsZero() {
		return nil
	}
	released, err := tx.parentReleased(fl)
	if err != nil || !released {
		return err
	}
	if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowDeactivated,
		api.FlowDeactivatedEvent{
			FlowID: tx.flowID,
			Status: fl.Status,
		},
	); err != nil {
		return err
	}
	tx.OnSuccess(func(fl api.FlowState, _ []*timebox.Event) {
		tx.releaseChildFlows(fl)
	})
	return nil
}

// parentReleased reports whether the parent can still order a rollback
func (tx *flowTx) parentReleased(fl api.FlowState) (bool, error) {
	target := &parentWork{}
	ok, err := parentMeta(fl, target)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}
	parent, err := tx.Engine.GetFlowState(target.fs.FlowID)
	if err != nil {
		return false, err
	}
	if parent.ID == "" || !parent.DeactivatedAt.IsZero() {
		return true, nil
	}
	work := parent.Executions[target.fs.StepID].WorkItems[target.token]
	return policy.WorkRollbackSettled(work.Status), nil
}

// releaseChildFlows lets child flows held open by this one deactivate
func (tx *flowTx) releaseChildFlows(fl api.FlowState) {
	for sid, ex := range fl.Executions {
		st, ok := fl.Plan.Steps[sid]
		if !ok || st.Type != api.StepTypeFlow {
			continue
		}
		fs := api.FlowStep{FlowID: fl.ID, StepID: sid}
		for tkn := range ex.WorkItems {
			tx.releaseChildFlow(fs, tkn)
		}
	}
}

func (tx *flowTx) releaseChildFlow(fs api.FlowStep, tkn api.Token) {
	childID := childFlowID(fs, tkn)
	err := tx.flowTx(childID, func(child *flowTx) error {
		if child.Value().ID == "" {
			return nil
		}
		return child.maybeDeactivate()
	})
	if err != nil {
		slog.Error("Failed to release child flow",
			log.FlowID(childID),
			log.Error(err))
	}
}

func (tx *flowTx) completeParentWork(fl api.FlowState) {
	target := &parentWork{}
	ok, err := parentMeta(fl, target)
	if !ok || err != nil {
		if err != nil {
			slog.Error("Failed to resolve parent work item",
				log.FlowID(tx.flowID),
				log.Error(err))
		}
		return
	}
	if fl.Status != api.FlowCompleted && fl.Status != api.FlowFailed {
		return
	}

	if err := tx.completeParentFlowWork(fl, target); err != nil {
		slog.Error("Failed to update parent work item",
			log.FlowID(tx.flowID),
			log.Error(err))
	}
}

func (tx *flowTx) completeParentFlowWork(
	child api.FlowState, target *parentWork,
) error {
	return tx.flowTx(target.fs.FlowID, func(parentTx *flowTx) error {
		parent := parentTx.Value()
		if parent.ID == "" {
			return errors.Join(ErrGetFlowState, ErrFlowNotFound)
		}

		ex := parent.Executions[target.fs.StepID]
		work := ex.WorkItems[target.token]
		if !policy.WorkAcceptsResult(work.Status) {
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

// getFailureReason extracts a failure reason from flow state
func getFailureReason(fl api.FlowState) string {
	for sid, ex := range fl.Executions {
		if policy.StepFailed(ex.Status) {
			return fmt.Sprintf("step %s failed: %s", sid, ex.Error)
		}
	}
	return "flow failed"
}

func flowHasRetryTasks(fl api.FlowState) bool {
	for _, ex := range fl.Executions {
		for _, work := range ex.WorkItems {
			if !work.NextRetryAt.IsZero() {
				return true
			}
		}
	}
	return false
}

func parentMeta(fl api.FlowState, target *parentWork) (bool, error) {
	if err := validateParentMetadata(fl.Metadata); err != nil {
		return false, fmt.Errorf("%w: %s", err, fl.ID)
	}

	meta := fl.Metadata
	fid, hasFlowID := meta.GetString[api.FlowID](api.MetaParentFlowID)
	sid, hasStepID := meta.GetString[api.StepID](api.MetaParentStepID)
	tkn, hasToken := meta.GetString[api.Token](api.MetaParentWorkItemToken)

	if !hasFlowID && !hasStepID && !hasToken {
		return false, nil
	}

	target.fs = api.FlowStep{FlowID: fid, StepID: sid}
	target.token = tkn
	return true, nil
}

func mapFlowOutputs(st *api.Step, childAttrs api.Args) (api.Args, error) {
	outputs := maps.Clone(childAttrs)

	for name, attr := range st.Attributes {
		if !attr.IsOutput() {
			continue
		}
		mapped, ok := st.MappedName(name)
		if !ok {
			continue
		}

		value, ok := childAttrs[mapped]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrFlowOutputMissing, mapped)
		}
		outputs[mapped] = value
	}

	return outputs, nil
}

func validateParentMetadata(meta api.Metadata) error {
	_, hasFlowID := meta.GetString[api.FlowID](api.MetaParentFlowID)
	_, hasStepID := meta.GetString[api.StepID](api.MetaParentStepID)
	_, hasToken := meta.GetString[api.Token](api.MetaParentWorkItemToken)
	if !hasFlowID && !hasStepID && !hasToken {
		return nil
	}
	if hasFlowID && hasStepID && hasToken {
		return nil
	}
	return ErrPartialParentMetadata
}
