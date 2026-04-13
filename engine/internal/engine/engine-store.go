package engine

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// GetCatalogState retrieves the current catalog state
func (e *Engine) GetCatalogState() (api.CatalogState, error) {
	return e.catalogExec.Get(events.CatalogKey)
}

// GetClusterState retrieves the current cluster state
func (e *Engine) GetClusterState() (api.ClusterState, error) {
	st, err := e.clusterExec.Get(events.ClusterKey)
	if err != nil {
		return api.ClusterState{}, err
	}
	return e.withConfiguredNodes(st), nil
}

// GetCatalogStateSeq retrieves catalog state and its next event sequence
func (e *Engine) GetCatalogStateSeq() (api.CatalogState, int64, error) {
	var seq int64
	st, err := e.execCatalog(
		func(_ api.CatalogState, ag *CatalogAggregator) error {
			seq = ag.NextSequence()
			return nil
		},
	)
	return st, seq, err
}

// GetClusterStateSeq retrieves cluster state and its next event sequence
func (e *Engine) GetClusterStateSeq() (api.ClusterState, int64, error) {
	var seq int64
	st, err := e.execCluster(
		func(_ api.ClusterState, ag *ClusterAggregator) error {
			seq = ag.NextSequence()
			return nil
		},
	)
	if err != nil {
		return api.ClusterState{}, 0, err
	}
	return e.withConfiguredNodes(st), seq, nil
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

func (e *Engine) execCatalog(
	cmd timebox.Command[api.CatalogState],
) (api.CatalogState, error) {
	return e.catalogExec.Exec(events.CatalogKey, cmd)
}

func (e *Engine) execCluster(
	cmd timebox.Command[api.ClusterState],
) (api.ClusterState, error) {
	return e.clusterExec.Exec(events.ClusterKey, cmd)
}

func (e *Engine) withConfiguredNodes(st api.ClusterState) api.ClusterState {
	for _, srv := range e.config.Raft.Servers {
		st = st.EnsureNode(api.NodeID(srv.ID))
	}
	return st
}
