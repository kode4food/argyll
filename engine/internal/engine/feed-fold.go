package engine

import (
	"errors"
	"maps"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
	"github.com/kode4food/argyll/engine/pkg/policy"
	"github.com/kode4food/argyll/engine/pkg/util"
)

var (
	ErrFeedFoldFailed = errors.New("feed fold failed")
)

// feedTx runs fn against one Feed version's aggregate inside the current store transaction
func (tx storeTx) feedTx(
	f *api.Feed, fn func(*FeedAggregator) error,
) (api.FeedState, error) {
	return tx.Exec(tx.feedExec, events.FeedKey(f.ID, f.Version()),
		func(_ api.FeedState, ag *FeedAggregator) error {
			return fn(ag)
		},
	)
}

// snapshotFeeds reads the values of the Feeds a plan's Steps consume
func (tx *flowTx) snapshotFeeds(pl *api.ExecutionPlan) (api.FeedValues, error) {
	res := api.FeedValues{}
	for _, st := range pl.Steps {
		for _, attr := range st.Attributes {
			if attr == nil || !attr.IsFeed() {
				continue
			}
			id := attr.FeedID()
			f := pl.Feeds[id].Feed
			if _, ok := res[id]; ok || f == nil {
				continue
			}
			state, err := tx.feedTx(f, func(*FeedAggregator) error {
				return nil
			})
			if err != nil {
				return nil, err
			}
			value, err := feedValue(f, state)
			if err != nil {
				return nil, err
			}
			res[id] = value
		}
	}
	return res, nil
}

// foldCompletedFeeds folds a just-completed Step's outputs when nothing can compensate them, since
// they are final from this moment
func (tx *flowTx) foldCompletedFeeds(st *api.Step, outputs api.Args) error {
	if len(tx.Value().Plan.Feeds) == 0 {
		return nil
	}
	comp, err := tx.steps.Compensator(st)
	if err != nil || comp != nil {
		return err
	}
	return tx.foldFeeds(st.ID, outputs)
}

// foldSettledFeeds folds, at deactivation, the outputs of completed Steps with a compensator whose
// work was not compensated
func (tx *flowTx) foldSettledFeeds() error {
	fl := tx.Value()
	if len(fl.Plan.Feeds) == 0 {
		return nil
	}
	for _, sid := range slices.Sorted(maps.Keys(fl.Executions)) {
		ex := fl.Executions[sid]
		st, ok := fl.Plan.Steps[sid]
		if !ok || !policy.StepSucceeded(ex.Status) {
			continue
		}
		comp, err := tx.steps.Compensator(st)
		if err != nil {
			return err
		}
		if comp == nil {
			continue
		}
		outputs := tx.collectStepOutputs(uncompensatedWork(ex.WorkItems), st)
		if err := tx.foldFeeds(sid, outputs); err != nil {
			return err
		}
	}
	return nil
}

// foldFeeds folds one Step's outputs into every Feed the plan sources from it
func (tx *flowTx) foldFeeds(sid api.StepID, outputs api.Args) error {
	fl := tx.Value()
	for _, id := range slices.Sorted(maps.Keys(fl.Plan.Feeds)) {
		pf := fl.Plan.Feeds[id]
		if !slices.Contains(pf.Sources, sid) {
			continue
		}
		f := pf.Feed
		matched := api.Args{}
		for _, name := range f.Attributes {
			if v, ok := outputs[name]; ok {
				matched[name] = v
			}
		}
		if len(matched) == 0 {
			continue
		}
		if err := tx.foldFeed(f, sid, matched); err != nil {
			return err
		}
	}
	return nil
}

func (tx *flowTx) foldFeed(f *api.Feed, sid api.StepID, outputs api.Args) error {
	version := f.Version()
	_, err := tx.feedTx(f, func(ag *FeedAggregator) error {
		current, err := feedValue(f, ag.Value())
		if err != nil {
			return err
		}
		next, err := tx.evaluateFold(f, outputs.Set(api.FeedCurrent, current))
		if err == nil {
			if util.EqualJSON(current, next) {
				return nil
			}
			err = events.Raise(ag, api.EventTypeFeedFolded,
				api.FeedFoldedEvent{
					Value:   next,
					FeedID:  f.ID,
					Version: version,
				},
			)
		}
		if err == nil {
			return nil
		}
		return events.Raise(ag, api.EventTypeFeedFoldFailed,
			api.FeedFoldFailedEvent{
				Outputs: outputs,
				FeedID:  f.ID,
				Version: version,
				FlowID:  tx.flowID,
				StepID:  sid,
				Error:   err.Error(),
			},
		)
	})
	return err
}

// evaluateFold runs a fold script with its bindings and returns the new value
func (e *Engine) evaluateFold(f *api.Feed, args api.Args) (any, error) {
	st := feedFoldStep(f)
	comp, err := e.scripts.Compile(st, f.Fold)
	if err != nil {
		return nil, errors.Join(ErrFeedFoldFailed, err)
	}
	env, err := e.scripts.Get(f.Fold.Language)
	if err != nil {
		return nil, errors.Join(ErrFeedFoldFailed, err)
	}
	res, err := env.Evaluate(comp, st, args)
	if err != nil {
		return nil, errors.Join(ErrFeedFoldFailed, err)
	}
	return res, nil
}

// feedFoldStep builds the synthetic Step a fold compiles against, exposing the Feed's current value
// and each of its Attributes
func feedFoldStep(f *api.Feed) *api.Step {
	attrs := api.AttributeSpecs{
		api.FeedCurrent: {Role: api.RoleRequired},
	}
	for _, name := range f.Attributes {
		attrs[name] = &api.AttributeSpec{Role: api.RoleRequired}
	}
	return &api.Step{
		ID:         api.StepID("feed:" + string(f.ID)),
		Attributes: attrs,
	}
}

// feedValue returns a Feed state's value, or the seed for a version that has never folded
func feedValue(f *api.Feed, state api.FeedState) (any, error) {
	if state.Version != "" {
		return state.Value, nil
	}
	return f.Seed()
}

// uncompensatedWork keeps the work whose effect still stands
func uncompensatedWork(items api.WorkItems) api.WorkItems {
	res := api.WorkItems{}
	for tkn, work := range items {
		if policy.WorkEffectStands(work.Status) {
			work.Status = api.WorkSucceeded
			res[tkn] = work
		}
	}
	return res
}
