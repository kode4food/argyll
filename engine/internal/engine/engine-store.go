package engine

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// GetCatalogState retrieves the current catalog state
func (e *Engine) GetCatalogState() (*api.CatalogState, error) {
	return e.execCatalog(func(*api.CatalogState, *CatalogAggregator) error {
		return nil
	})
}

// GetNodeState retrieves the current local node state
func (e *Engine) GetNodeState() (*api.NodeState, error) {
	return e.execNode(func(*api.NodeState, *NodeAggregator) error {
		return nil
	})
}

// GetShardNodeStates retrieves node state for every known node in the shard,
// deriving membership from the Raft server configuration
func (e *Engine) GetShardNodeStates() (map[api.NodeID]*api.NodeState, error) {
	servers := e.config.Raft.Servers
	res := make(map[api.NodeID]*api.NodeState, len(servers))
	for _, srv := range servers {
		id := api.NodeID(srv.ID)
		st, err := e.execNodeAt(events.NodeKey(id),
			func(*api.NodeState, *NodeAggregator) error {
				return nil
			},
		)
		if err != nil {
			return nil, err
		}
		res[id] = st
	}
	return res, nil
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

// GetNodeStateSeq retrieves local node state and its next event sequence
func (e *Engine) GetNodeStateSeq() (*api.NodeState, int64, error) {
	return e.GetNodeStateSeqFor(api.NodeID(e.config.Raft.LocalID))
}

// GetNodeStateSeqFor retrieves node state and its next event sequence
func (e *Engine) GetNodeStateSeqFor(
	nodeID api.NodeID,
) (*api.NodeState, int64, error) {
	var seq int64
	state, err := e.execNodeAt(events.NodeKey(nodeID),
		func(_ *api.NodeState, ag *NodeAggregator) error {
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

func (e *Engine) execNode(
	cmd timebox.Command[*api.NodeState],
) (*api.NodeState, error) {
	return e.execNodeAt(e.nodeKey(), cmd)
}

func (e *Engine) execNodeAt(
	id timebox.AggregateID, cmd timebox.Command[*api.NodeState],
) (*api.NodeState, error) {
	return e.nodeExec.Exec(id, cmd)
}

func (e *Engine) nodeKey() timebox.AggregateID {
	return events.NodeKey(api.NodeID(e.config.Raft.LocalID))
}
