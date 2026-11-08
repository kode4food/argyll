package events

import (
	"github.com/kode4food/timebox"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// MemoPrefix is the aggregate type of every memoized result
const MemoPrefix = "memo"

// MemoAppliers contains the event applier functions for memo events
var MemoAppliers = makeMemoAppliers()

// NewMemoState creates a memo state with no successful result
func NewMemoState() *api.WorkSucceededEvent {
	return nil
}

// MemoKey returns the aggregate ID for one Step definition and input set
func MemoKey(st *api.Step, inputs api.Args) (timebox.AggregateID, error) {
	stepKey, err := st.HashKey()
	if err != nil {
		return timebox.AggregateID{}, err
	}
	inputKey, err := inputs.HashKey()
	if err != nil {
		return timebox.AggregateID{}, err
	}
	return timebox.NewAggregateID(
		MemoPrefix, timebox.ID(stepKey+":"+inputKey),
	), nil
}

func makeMemoAppliers() timebox.Appliers[*api.WorkSucceededEvent] {
	return MakeAppliers(
		map[api.EventType]timebox.Applier[*api.WorkSucceededEvent]{
			api.EventTypeWorkSucceeded: timebox.MakeApplier(memoSucceeded),
		},
	)
}

func memoSucceeded(
	state *api.WorkSucceededEvent, _ *timebox.Event,
	data api.WorkSucceededEvent,
) *api.WorkSucceededEvent {
	if state != nil {
		return state
	}
	return &data
}
