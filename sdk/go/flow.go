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
	goals  api.Goals
	init   api.InitArgs
	tags   api.Tags
}

// NewFlow creates a new flow builder with the specified ID
func (c *Client) NewFlow(id api.FlowID) Flow {
	return Flow{
		client: c,
		id:     id,
		goals:  api.Goals{},
	}
}

// WithGoals sets the chain of fallback goal sets for the flow
func (f Flow) WithGoals(goals api.Goals) Flow {
	f.goals = *goals.Copy()
	return f
}

// WithGoal adds a single goal step ID to the flow's last goal set
func (f Flow) WithGoal(goal api.StepID) Flow {
	f.goals = withLastGoal(f.goals, goal)
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
	f.tags = api.Append(f.tags, tags...).Normalize()
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

func withLastGoal(g api.Goals, goal api.StepID) api.Goals {
	if g.Else == nil {
		return api.Goals{Steps: append(slices.Clone(g.Steps), goal)}
	}
	rest := withLastGoal(*g.Else, goal)
	return api.Goals{Steps: g.Steps, Else: &rest}
}
