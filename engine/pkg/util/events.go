package util

import (
	"encoding/json"

	"github.com/kode4food/timebox"
)

// Raise marshals an event and raises it through the aggregator
func Raise[T, E any](
	ag *timebox.Aggregator[T], eventType timebox.EventType, event E,
) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	ag.Raise(eventType, data)
	return nil
}
