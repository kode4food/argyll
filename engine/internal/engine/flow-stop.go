package engine

import (
	"errors"
	"fmt"
	"maps"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util/call"
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
	if err := tx.failAbandonedWork(); err != nil {
		return err
	}
	fl := tx.Value()
	if policy.FlowTerminal(fl.Status) {
		return nil
	}
	if isFlowComplete(&fl) {
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowCompleted,
			api.FlowCompletedEvent{
				FlowID: tx.flowID,
				Result: flowResult(&fl),
			},
		); err != nil {
			return err
		}
		return tx.maybeDeactivate()
	}
	if tx.IsFlowFailed(&fl) {
		errMsg := getFailureReason(&fl)
		if err := events.Raise(tx.FlowAggregator, api.EventTypeFlowFailed,
			api.FlowFailedEvent{
				FlowID: tx.flowID,
				Error:  errMsg,
			},
		); err != nil {
			return err
		}
		return tx.maybeDeactivate()
	}
	return nil
}

// maybeDeactivate reports the outcome to the parent, then deactivates once
// the parent can no longer order a rollback
func (tx *flowTx) maybeDeactivate() error {
	if !policy.FlowTerminal(tx.Value().Status) ||
		!tx.Value().DeactivatedAt.IsZero() {
		return nil
	}
	// Compensation may start new work, so sweep before testing for active
	// work rather than after
	if err := call.Perform(
		tx.failAbandonedWork,
		tx.compensateFlow,
	); err != nil {
		return err
	}
	// Told before deactivating, since the parent decides the rollback. Both
	// halves join one transaction, so the parent cannot miss the outcome
	parent, err := tx.completeParentWork()
	if err != nil {
		return err
	}
	fl := tx.Value()
	if hasActiveWork(&fl) || tx.compensationPending(&fl) {
		return nil
	}
	return tx.deactivate(parent)
}

func (tx *flowTx) failAbandonedWork() error {
	fl := tx.Value()
	for sid, ex := range fl.Executions {
		if !policy.FlowTerminal(fl.Status) && !policy.StepTerminal(ex.Status) {
			continue
		}
		for tkn, work := range ex.WorkItems {
			if !policy.WorkPending(work.Status) {
				continue
			}
			if err := tx.raiseWorkFailed(
				sid, tkn, "execution no longer active",
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tx *flowTx) deactivate(parent api.FlowState) error {
	fl := tx.Value()
	if !fl.DeactivatedAt.IsZero() {
		return nil
	}
	released, err := tx.parentReleased(parentReleasedArgs{
		flow:   &fl,
		parent: &parent,
	})
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
	return tx.releaseChildFlows()
}

type parentReleasedArgs struct {
	flow   *api.FlowState
	parent *api.FlowState
}

// parentReleased reports whether the parent can still order a rollback, from
// the state it settled earlier in this same transaction
func (tx *flowTx) parentReleased(args parentReleasedArgs) (bool, error) {
	target := &parentWork{}
	ok, err := parentMeta(args.flow, target)
	if err != nil {
		return false, err
	}
	parent := args.parent
	if !ok || parent.ID == "" || !parent.DeactivatedAt.IsZero() {
		return true, nil
	}
	work := parent.Executions[target.fs.StepID].WorkItems[target.token]
	return policy.WorkRollbackSettled(work.Status), nil
}

// releaseChildFlows lets child flows held open by this one deactivate
func (tx *flowTx) releaseChildFlows() error {
	fl := tx.Value()
	for sid, ex := range fl.Executions {
		st, ok := fl.Plan.Steps[sid]
		if !ok || st.Type != api.StepTypeFlow {
			continue
		}
		fs := api.FlowStep{FlowID: fl.ID, StepID: sid}
		for tkn := range ex.WorkItems {
			if _, err := tx.flowTx(childFlowID(fs, tkn),
				func(child *flowTx) error {
					return child.maybeDeactivate()
				},
			); err != nil {
				return err
			}
		}
	}
	return nil
}

// completeParentWork settles this flow's outcome with its parent in tx and
// returns the parent's resulting state, so both halves land in one commit
func (tx *flowTx) completeParentWork() (api.FlowState, error) {
	fl := tx.Value()
	target := &parentWork{}
	ok, err := parentMeta(&fl, target)
	if !ok || err != nil {
		return api.FlowState{}, err
	}
	return tx.completeParentFlowWork(fl, target)
}

func (tx *flowTx) completeParentFlowWork(
	child api.FlowState, target *parentWork,
) (api.FlowState, error) {
	settle := func(parentTx *flowTx) error {
		parent := parentTx.Value()
		if parent.ID == "" {
			return errors.Join(ErrGetFlowState, api.ErrFlowNotFound)
		}

		ex := parent.Executions[target.fs.StepID]
		work := ex.WorkItems[target.token]
		if !policy.WorkAcceptsResult(work.Status) {
			return nil
		}

		if child.Status == api.FlowCompleted {
			outputs := child.GetAttributes()
			err := validateFlowOutputs(
				parent.Plan.Steps[target.fs.StepID], outputs,
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
	}
	return tx.flowTx(target.fs.FlowID, settle)
}

// flowResult includes initial outputs of goals excluded as already satisfied
func flowResult(fl *api.FlowState) api.Args {
	res := api.Args{}
	for _, sid := range startGoals(fl) {
		maps.Copy(res, fl.Executions[sid].Outputs)
		for _, name := range fl.Plan.Excluded.Satisfied[sid] {
			if value, ok := fl.FirstAttribute(name); ok {
				res[name] = value.Value
			}
		}
	}
	return res
}

// getFailureReason extracts a failure reason from flow state
func getFailureReason(fl *api.FlowState) string {
	for sid, ex := range fl.Executions {
		if policy.StepFailed(ex.Status) {
			return fmt.Sprintf("step %s failed: %s", sid, ex.Error)
		}
	}
	return "flow failed"
}

func parentMeta(fl *api.FlowState, target *parentWork) (bool, error) {
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

func validateFlowOutputs(st *api.Step, outputs api.Args) error {
	for name, attr := range st.Attributes {
		if !attr.IsOutput() {
			continue
		}
		mapped, ok := st.MappedName(name)
		if !ok {
			continue
		}

		if _, ok := outputs[mapped]; !ok {
			return fmt.Errorf("%w: %s", ErrFlowOutputMissing, mapped)
		}
	}

	return nil
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
