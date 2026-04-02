package api

import (
	"maps"
	"time"
)

type (
	// FlowStatus represents the current state of a flow
	FlowStatus string

	// StepStatus represents the current state of a step execution
	StepStatus string

	// HealthStatus represents the health of a step service
	HealthStatus string

	// WorkStatus represents the state of a single work item
	WorkStatus string

	// Token uniquely identifies a work item within a step
	Token string

	// CatalogState contains the global step catalog for the cluster
	CatalogState struct {
		LastUpdated time.Time      `json:"last_updated"`
		Steps       Steps          `json:"steps"`
		Attributes  AttributeGraph `json:"attributes"`
	}

	// ClusterState contains the operational state of all nodes in the cluster
	ClusterState struct {
		LastUpdated time.Time             `json:"last_updated"`
		Nodes       map[NodeID]*NodeState `json:"nodes"`
	}

	// NodeState contains a node's operational state
	NodeState struct {
		LastSeen time.Time               `json:"last_seen"`
		Health   map[StepID]*HealthState `json:"health"`
	}

	// FlowState contains the complete state of a flow execution
	FlowState struct {
		CreatedAt     time.Time       `json:"created_at"`
		CompletedAt   time.Time       `json:"completed_at"`
		LastUpdated   time.Time       `json:"last_updated"`
		Plan          *ExecutionPlan  `json:"plan"`
		Metadata      Metadata        `json:"metadata,omitempty"`
		Labels        Labels          `json:"labels,omitempty"`
		Attributes    AttributeValues `json:"attributes"`
		DeactivatedAt time.Time       `json:"deactivated_at"`
		Executions    Executions      `json:"executions"`
		ID            FlowID          `json:"id"`
		Status        FlowStatus      `json:"status"`
		Error         string          `json:"error,omitempty"`
	}

	// Executions contains the execution progress of multiple steps
	Executions map[StepID]*ExecutionState

	// AttributeValues contains fulfilled attribute values and their sources
	AttributeValues map[Name]*AttributeValue

	// AttributeValue stores an attribute value and which step produced it
	AttributeValue struct {
		Value any       `json:"value"`
		Step  StepID    `json:"step,omitempty"`
		SetAt time.Time `json:"set_at"`
	}

	// ExecutionState contains the state of a step execution
	ExecutionState struct {
		StartedAt   time.Time  `json:"started_at"`
		CompletedAt time.Time  `json:"completed_at"`
		Inputs      Args       `json:"inputs"`
		Outputs     Args       `json:"outputs,omitempty"`
		Status      StepStatus `json:"status"`
		Error       string     `json:"error,omitempty"`
		Duration    int64      `json:"duration,omitempty"`
		WorkItems   WorkItems  `json:"work_items,omitempty"`
	}

	// WorkItems contains the state of multiple work items
	WorkItems map[Token]*WorkState

	// WorkState contains the state of a single work item
	WorkState struct {
		Status      WorkStatus `json:"status"`
		StartedAt   time.Time  `json:"started_at"`
		CompletedAt time.Time  `json:"completed_at"`
		Inputs      Args       `json:"inputs"`
		Outputs     Args       `json:"outputs,omitempty"`
		Error       string     `json:"error,omitempty"`
		RetryCount  int        `json:"retry_count,omitempty"`
		NextRetryAt time.Time  `json:"next_retry_at"`
	}

	// HealthState contains the health status of a step service
	HealthState struct {
		Status HealthStatus `json:"status"`
		Error  string       `json:"error,omitempty"`
	}
)

const (
	FlowActive    FlowStatus = "active"
	FlowCompleted FlowStatus = "completed"
	FlowFailed    FlowStatus = "failed"
)

const (
	StepPending   StepStatus = "pending"
	StepActive    StepStatus = "active"
	StepCompleted StepStatus = "completed"
	StepSkipped   StepStatus = "skipped"
	StepFailed    StepStatus = "failed"
)

const (
	WorkPending      WorkStatus = "pending"
	WorkActive       WorkStatus = "active"
	WorkSucceeded    WorkStatus = "succeeded"
	WorkFailed       WorkStatus = "failed"
	WorkNotCompleted WorkStatus = "not_completed"
)

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

// SetStep returns a new CatalogState with the specified step registered
func (c *CatalogState) SetStep(id StepID, step *Step) *CatalogState {
	res := *c
	res.Steps = maps.Clone(c.Steps)
	res.Steps[id] = step

	if oldStep, ok := c.Steps[id]; ok {
		res.Attributes = res.Attributes.RemoveStep(oldStep)
	}

	res.Attributes = res.Attributes.AddStep(step)
	return &res
}

// DeleteStep returns a new CatalogState with the specified step removed
func (c *CatalogState) DeleteStep(id StepID) *CatalogState {
	step, ok := c.Steps[id]
	if !ok {
		return c
	}

	res := *c
	res.Steps = maps.Clone(c.Steps)
	delete(res.Steps, id)
	res.Attributes = res.Attributes.RemoveStep(step)
	return &res
}

// SetLastUpdated returns a new CatalogState with the last updated timestamp set
func (c *CatalogState) SetLastUpdated(t time.Time) *CatalogState {
	res := *c
	res.LastUpdated = t
	return &res
}

// SetNode returns a new ClusterState with the specified node updated
func (c *ClusterState) SetNode(id NodeID, n *NodeState) *ClusterState {
	res := *c
	res.Nodes = maps.Clone(c.Nodes)
	if res.Nodes == nil {
		res.Nodes = map[NodeID]*NodeState{}
	}
	res.Nodes[id] = n
	return &res
}

// EnsureNode returns a new ClusterState with the specified node present
func (c *ClusterState) EnsureNode(id NodeID) *ClusterState {
	if n, ok := c.Nodes[id]; ok && n != nil {
		return c
	}
	return c.SetNode(id, &NodeState{
		Health: map[StepID]*HealthState{},
	})
}

// SetLastUpdated returns a new ClusterState with the last updated timestamp set
func (c *ClusterState) SetLastUpdated(t time.Time) *ClusterState {
	res := *c
	res.LastUpdated = t
	return &res
}

// SetHealth returns a new NodeState with updated health for a given step
func (n *NodeState) SetHealth(id StepID, h *HealthState) *NodeState {
	res := *n
	res.Health = maps.Clone(n.Health)
	if res.Health == nil {
		res.Health = map[StepID]*HealthState{}
	}
	res.Health[id] = h
	return &res
}

// SetLastSeen returns a new NodeState with the last seen timestamp set
func (n *NodeState) SetLastSeen(t time.Time) *NodeState {
	res := *n
	res.LastSeen = t
	return &res
}

// GetAttributes returns all attribute values as Args
func (f *FlowState) GetAttributes() Args {
	result := make(Args, len(f.Attributes))
	for key, attr := range f.Attributes {
		result[key] = attr.Value
	}
	return result
}

// SetStatus returns a new FlowState with the updated status
func (f *FlowState) SetStatus(s FlowStatus) *FlowState {
	res := *f
	res.Status = s
	return &res
}

// SetAttribute returns a new FlowState with the specified attribute set
func (f *FlowState) SetAttribute(name Name, attr *AttributeValue) *FlowState {
	res := *f
	res.Attributes = maps.Clone(f.Attributes)
	res.Attributes[name] = attr
	return &res
}

func (f *FlowState) SetExecution(id StepID, ex *ExecutionState) *FlowState {
	res := *f
	res.Executions = maps.Clone(f.Executions)
	res.Executions[id] = ex
	return &res
}

// SetCompletedAt returns a new FlowState with the completion timestamp set
func (f *FlowState) SetCompletedAt(t time.Time) *FlowState {
	res := *f
	res.CompletedAt = t
	return &res
}

// SetError returns a new FlowState with the error message set
func (f *FlowState) SetError(err string) *FlowState {
	res := *f
	res.Error = err
	return &res
}

// SetLastUpdated returns a new FlowState with last updated time set
func (f *FlowState) SetLastUpdated(t time.Time) *FlowState {
	res := *f
	res.LastUpdated = t
	return &res
}

// SetDeactivatedAt returns a new FlowState with deactivated time set
func (f *FlowState) SetDeactivatedAt(t time.Time) *FlowState {
	res := *f
	res.DeactivatedAt = t
	return &res
}

// SetStatus returns a new ExecutionState with the updated status
func (e *ExecutionState) SetStatus(s StepStatus) *ExecutionState {
	res := *e
	res.Status = s
	return &res
}

// SetStartedAt returns a new ExecutionState with the start timestamp set
func (e *ExecutionState) SetStartedAt(t time.Time) *ExecutionState {
	res := *e
	res.StartedAt = t
	return &res
}

// SetCompletedAt returns a new ExecutionState with completion time set
func (e *ExecutionState) SetCompletedAt(t time.Time) *ExecutionState {
	res := *e
	res.CompletedAt = t
	return &res
}

// SetInputs returns a new ExecutionState with the input arguments set
func (e *ExecutionState) SetInputs(inputs Args) *ExecutionState {
	res := *e
	res.Inputs = inputs
	return &res
}

// SetOutputs returns a new ExecutionState with the output arguments set
func (e *ExecutionState) SetOutputs(outputs Args) *ExecutionState {
	res := *e
	res.Outputs = outputs
	return &res
}

// SetDuration returns a new ExecutionState with the execution duration set
func (e *ExecutionState) SetDuration(dur int64) *ExecutionState {
	res := *e
	res.Duration = dur
	return &res
}

// SetError returns a new ExecutionState with the error message set
func (e *ExecutionState) SetError(err string) *ExecutionState {
	res := *e
	res.Error = err
	return &res
}

// SetWorkItem returns a new ExecutionState with the work item state updated
func (e *ExecutionState) SetWorkItem(
	token Token, item *WorkState,
) *ExecutionState {
	res := *e
	res.WorkItems = maps.Clone(e.WorkItems)
	res.WorkItems[token] = item
	return &res
}

// RemoveWorkItem returns a new ExecutionState with the work item removed
func (e *ExecutionState) RemoveWorkItem(token Token) *ExecutionState {
	res := *e
	res.WorkItems = maps.Clone(e.WorkItems)
	delete(res.WorkItems, token)
	return &res
}

// SetStatus returns a new HealthState with the updated status
func (h *HealthState) SetStatus(s HealthStatus) *HealthState {
	res := *h
	res.Status = s
	return &res
}

// SetError returns a new HealthState with the error message set
func (h *HealthState) SetError(err string) *HealthState {
	res := *h
	res.Error = err
	return &res
}

// SetStatus returns a new WorkState with the updated status
func (w *WorkState) SetStatus(s WorkStatus) *WorkState {
	res := *w
	res.Status = s
	return &res
}

// SetStartedAt returns a new WorkState with the started timestamp set
func (w *WorkState) SetStartedAt(t time.Time) *WorkState {
	res := *w
	res.StartedAt = t
	return &res
}

// SetCompletedAt returns a new WorkState with the completed timestamp set
func (w *WorkState) SetCompletedAt(t time.Time) *WorkState {
	res := *w
	res.CompletedAt = t
	return &res
}

// SetRetryCount returns a new WorkState with the retry count set
func (w *WorkState) SetRetryCount(count int) *WorkState {
	res := *w
	res.RetryCount = count
	return &res
}

// SetNextRetryAt returns a new WorkState with the next retry time set
func (w *WorkState) SetNextRetryAt(t time.Time) *WorkState {
	res := *w
	res.NextRetryAt = t
	return &res
}

// SetError returns a new WorkState with the error message set
func (w *WorkState) SetError(err string) *WorkState {
	res := *w
	res.Error = err
	return &res
}

// SetOutputs returns a new WorkState with the outputs set
func (w *WorkState) SetOutputs(outputs Args) *WorkState {
	res := *w
	res.Outputs = outputs
	return &res
}
