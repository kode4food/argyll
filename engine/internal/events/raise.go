package events

import (
	"encoding/json"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

// Raise marshals an event and raises it through the aggregator
func Raise[T, E any](
	ag *timebox.Aggregator[T], eventType api.EventType, event E,
) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	ag.Raise(timebox.EventType(eventType), data)
	return nil
}
