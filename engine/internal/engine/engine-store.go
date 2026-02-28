package engine

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/internal/engine/event"
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/events"
)

// GetCatalogState retrieves the current catalog state
func (e *Engine) GetCatalogState() (*api.CatalogState, error) {
	state, err := e.execCatalog(
		func(st *api.CatalogState, ag *CatalogAggregator) error {
			return nil
		},
	)
	return state, err
}

// GetPartitionState retrieves the current partition state
func (e *Engine) GetPartitionState() (*api.PartitionState, error) {
	state, err := e.execPartition(
		func(st *api.PartitionState, ag *PartitionAggregator) error {
			return nil
		},
	)
	return state, err
}

// GetCatalogStateSeq retrieves catalog state and its next event sequence
func (e *Engine) GetCatalogStateSeq() (*api.CatalogState, int64, error) {
	var seq int64
	state, err := e.execCatalog(
		func(st *api.CatalogState, ag *CatalogAggregator) error {
			seq = ag.NextSequence()
			return nil
		},
	)
	return state, seq, err
}

// GetPartitionStateSeq retrieves partition state and its next event sequence
func (e *Engine) GetPartitionStateSeq() (*api.PartitionState, int64, error) {
	var seq int64
	state, err := e.execPartition(
		func(st *api.PartitionState, ag *PartitionAggregator) error {
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
		func(st *api.CatalogState, ag *CatalogAggregator) error {
			return events.Raise(ag, typ, data)
		},
	)
	return err
}

func (e *Engine) raisePartitionEvent(typ api.EventType, data any) error {
	return e.raisePartitionEvents([]event.Event{{
		Type: typ,
		Data: data,
	}})
}

func (e *Engine) raisePartitionEvents(evs []event.Event) error {
	_, err := e.execPartition(
		func(st *api.PartitionState, ag *PartitionAggregator) error {
			for _, ev := range evs {
				if err := events.Raise(ag, ev.Type, ev.Data); err != nil {
					return err
				}
			}
			return nil
		},
	)
	return err
}

func (e *Engine) execCatalog(
	cmd timebox.Command[*api.CatalogState],
) (*api.CatalogState, error) {
	return e.catalogExec.Exec(e.ctx, events.CatalogKey, cmd)
}

func (e *Engine) execPartition(
	cmd timebox.Command[*api.PartitionState],
) (*api.PartitionState, error) {
	return e.partExec.Exec(e.ctx, events.PartitionKey, cmd)
}
