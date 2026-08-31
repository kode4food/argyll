package argyll

import (
	"context"
	"slices"

	"github.com/kode4food/argyll/engine/pkg/api"
)

// Flow is a builder for creating and starting flow executions
type Flow struct {
	client *Client
	id     api.FlowID
	goals  []api.StepID
	init   api.InitArgs
	tags   api.Tags
}

// NewFlow creates a new flow builder with the specified ID
func (c *Client) NewFlow(id api.FlowID) Flow {
	return Flow{
		client: c,
		id:     id,
		goals:  []api.StepID{},
	}
}

// WithGoals sets the goal step IDs for the flow
func (f Flow) WithGoals(goals ...api.StepID) Flow {
	f.goals = make([]api.StepID, len(goals))
	copy(f.goals, goals)
	return f
}

// WithGoal adds a single goal step ID to the flow
func (f Flow) WithGoal(goal api.StepID) Flow {
	goals := make([]api.StepID, len(f.goals)+1)
	copy(goals, f.goals)
	goals[len(f.goals)] = goal
	f.goals = goals
	return f
}

// WithInitialState sets the initial state for the flow
func (f Flow) WithInitialState(init api.InitArgs) Flow {
	f.init = init
	return f
}

// WithTags merges the provided tags into the flow's tag set
func (f Flow) WithTags(tags ...string) Flow {
	if len(tags) == 0 {
		return f
	}
	f.tags = append(slices.Clone(f.tags), tags...).Normalize()
	return f
}

// Start creates and starts the flow
func (f Flow) Start(ctx context.Context) error {
	return f.client.startFlow(ctx, &api.CreateFlowRequest{
		ID:    f.id,
		Goals: f.goals,
		Init:  f.init,
		Tags:  f.tags,
	})
}
