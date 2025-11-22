package builder

import (
	"context"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

// Flow is a builder for creating and starting flow executions
type Flow struct {
	client *Client
	id     FlowID
	goals  []StepID
	init   api.Args
}

// NewFlow creates a new flow builder with the specified ID
func (c *Client) NewFlow(id FlowID) *Flow {
	return &Flow{
		client: c,
		id:     id,
		goals:  []StepID{},
	}
}

// WithGoals sets the goal step IDs for the flow
func (f *Flow) WithGoals(goals ...StepID) *Flow {
	res := *f
	res.goals = make([]StepID, len(goals))
	copy(res.goals, goals)
	return &res
}

// WithGoal adds a single goal step ID to the flow
func (f *Flow) WithGoal(goal StepID) *Flow {
	res := *f
	res.goals = make([]StepID, len(f.goals)+1)
	copy(res.goals, f.goals)
	res.goals[len(f.goals)] = goal
	return &res
}

// WithInitialState sets the initial state for the flow
func (f *Flow) WithInitialState(init api.Args) *Flow {
	res := *f
	res.init = init
	return &res
}

// Start creates and starts the flow
func (f *Flow) Start(ctx context.Context) error {
	goals := make([]timebox.ID, len(f.goals))
	for i, g := range f.goals {
		goals[i] = timebox.ID(g)
	}
	return f.client.startFlow(ctx, &api.CreateFlowRequest{
		ID:    timebox.ID(f.id),
		Goals: goals,
		Init:  f.init,
	})
}
