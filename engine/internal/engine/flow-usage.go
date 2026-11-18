package engine

import (
	"maps"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
)

// forwardUsage carries a settlement's reported usage to the parent chain when this flow already
// reached its outcome
func (tx *flowTx) forwardUsage(usage api.Usage) error {
	if usage == nil {
		return nil
	}
	return tx.reportParentUsage()
}

// reportParentUsage carries a terminal flow's usage total to the parent work item it already
// settled, so usage arriving after the flow's outcome still reaches its parent
func (tx *flowTx) reportParentUsage() error {
	fl := tx.Value()
	if !policy.FlowTerminal(fl.Status) {
		return nil
	}
	target := &parentWork{}
	ok, err := parentMeta(&fl, target)
	if !ok || err != nil {
		return err
	}
	usage := flowUsage(&fl)
	_, err = tx.flowTx(target.fs.FlowID, func(parentTx *flowTx) error {
		return parentTx.reportUsage(target, usage)
	})
	return err
}

// reportUsage replaces a settled work item's usage with its child flow's latest total, and carries
// the change on up the chain
func (tx *flowTx) reportUsage(target *parentWork, usage api.Usage) error {
	work, ok := tx.Value().Executions[target.fs.StepID].WorkItems[target.token]
	if !ok || policy.WorkAcceptsResult(work.Status) || maps.Equal(work.Usage, usage) {
		return nil
	}
	if err := events.Raise(tx.FlowAggregator, api.EventTypeChildUsageUpdated,
		api.ChildUsageUpdatedEvent{
			FlowID: tx.flowID,
			StepID: target.fs.StepID,
			Token:  target.token,
			Usage:  usage,
		},
	); err != nil {
		return err
	}
	return tx.reportParentUsage()
}

// flowUsage sums the usage every work item of a flow reported, which for a flow step's work item
// already holds its own child flow's usage
func flowUsage(fl *api.FlowState) api.Usage {
	var res api.Usage
	for _, ex := range fl.Executions {
		res = res.Add(sumUsage(ex.WorkItems))
	}
	return res
}
