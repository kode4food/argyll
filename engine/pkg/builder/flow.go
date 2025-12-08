package builder

import (
	"context"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// Flow is a builder for creating and starting flow executions
type Flow struct {
	client *Client
	id     api.FlowID
	goals  []api.StepID
	init   api.Args
}

// NewFlow creates a new flow builder with the specified ID
func (c *Client) NewFlow(id api.FlowID) *Flow {
	return &Flow{
		client: c,
		id:     id,
		goals:  []api.StepID{},
	}
}

// WithGoals sets the goal step IDs for the flow
func (f *Flow) WithGoals(goals ...api.StepID) *Flow {
	res := *f
	res.goals = make([]api.StepID, len(goals))
	copy(res.goals, goals)
	return &res
}

// WithGoal adds a single goal step ID to the flow
func (f *Flow) WithGoal(goal api.StepID) *Flow {
	res := *f
	res.goals = make([]api.StepID, len(f.goals)+1)
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
	return f.client.startFlow(ctx, &api.CreateFlowRequest{
		ID:    f.id,
		Goals: f.goals,
		Init:  f.init,
	})
}
