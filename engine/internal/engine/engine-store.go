package engine

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// GetCatalogState retrieves the current catalog state
func (e *Engine) GetCatalogState() (*api.CatalogState, error) {
	return e.catalogExec.Get(events.CatalogKey)
}

// GetClusterState retrieves the current cluster state
func (e *Engine) GetClusterState() (*api.ClusterState, error) {
	return e.clusterExec.Get(events.ClusterKey)
}

// GetCatalogStateSeq retrieves catalog state and its next event sequence
func (e *Engine) GetCatalogStateSeq() (*api.CatalogState, int64, error) {
	var seq int64
	state, err := e.execCatalog(
		func(_ *api.CatalogState, ag *CatalogAggregator) error {
			seq = ag.NextSequence()
			return nil
		},
	)
	return state, seq, err
}

// GetClusterStateSeq retrieves cluster state and its next event sequence
func (e *Engine) GetClusterStateSeq() (*api.ClusterState, int64, error) {
	var seq int64
	state, err := e.execCluster(
		func(_ *api.ClusterState, ag *ClusterAggregator) error {
			seq = ag.NextSequence()
			return nil
		},
	)
	return state, seq, err
}

// ListSteps returns all currently registered steps in the engine
func (e *Engine) ListSteps() ([]*api.Step, error) {
	cat, err := e.GetCatalogState()
	if err != nil {
		return nil, err
	}

	var steps []*api.Step
	for _, step := range cat.Steps {
		steps = append(steps, step)
	}

	return steps, nil
}

func (e *Engine) raiseCatalogEvent(typ api.EventType, data any) error {
	_, err := e.execCatalog(
		func(_ *api.CatalogState, ag *CatalogAggregator) error {
			return events.Raise(ag, typ, data)
		},
	)
	return err
}

func (e *Engine) execCatalog(
	cmd timebox.Command[*api.CatalogState],
) (*api.CatalogState, error) {
	return e.catalogExec.Exec(events.CatalogKey, cmd)
}

func (e *Engine) execCluster(
	cmd timebox.Command[*api.ClusterState],
) (*api.ClusterState, error) {
	return e.clusterExec.Exec(events.ClusterKey, cmd)
}
