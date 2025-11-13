package builder

import (
	"context"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type Workflow struct {
	client *Client
	id     timebox.ID
	goals  []timebox.ID
	init   api.Args
}

// NewWorkflow creates a new workflow builder with the specified ID
func (c *Client) NewWorkflow(id timebox.ID) *Workflow {
	return &Workflow{
		client: c,
		id:     id,
		goals:  []timebox.ID{},
		init:   api.Args{},
	}
}

// WithGoals sets the goal step IDs for the workflow
func (f *Workflow) WithGoals(goals ...timebox.ID) *Workflow {
	res := *f
	res.goals = make([]timebox.ID, len(goals))
	copy(res.goals, goals)
	return &res
}

// WithGoal adds a single goal step ID to the workflow
func (f *Workflow) WithGoal(goal timebox.ID) *Workflow {
	res := *f
	res.goals = make([]timebox.ID, len(f.goals)+1)
	copy(res.goals, f.goals)
	res.goals[len(f.goals)] = goal
	return &res
}

// WithInitialState sets the initial state for the workflow
func (f *Workflow) WithInitialState(init api.Args) *Workflow {
	res := *f
	res.init = make(api.Args, len(init))
	for k, v := range init {
		res.init[k] = v
	}
	return &res
}

// Start creates and starts the workflow
func (f *Workflow) Start(ctx context.Context) error {
	return f.client.startWorkflow(ctx, &api.CreateWorkflowRequest{
		ID:    f.id,
		Goals: f.goals,
		Init:  f.init,
	})
}
