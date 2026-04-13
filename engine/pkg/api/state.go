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
		LastUpdated time.Time            `json:"last_updated"`
		Nodes       map[NodeID]NodeState `json:"nodes"`
	}

	// NodeState contains a node's operational state
	NodeState struct {
		LastSeen time.Time              `json:"last_seen"`
		Health   map[StepID]HealthState `json:"health"`
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
	Executions map[StepID]ExecutionState

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
	WorkItems map[Token]WorkState

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
func (c CatalogState) SetStep(id StepID, step *Step) CatalogState {
	c.Steps = maps.Clone(c.Steps)
	c.Steps[id] = step

	if oldStep, ok := c.Steps[id]; ok {
		c.Attributes = c.Attributes.RemoveStep(oldStep)
	}

	c.Attributes = c.Attributes.AddStep(step)
	return c
}

// DeleteStep returns a new CatalogState with the specified step removed
func (c CatalogState) DeleteStep(id StepID) CatalogState {
	step, ok := c.Steps[id]
	if !ok {
		return c
	}

	c.Steps = maps.Clone(c.Steps)
	delete(c.Steps, id)
	c.Attributes = c.Attributes.RemoveStep(step)
	return c
}

// SetLastUpdated returns a new CatalogState with the last updated timestamp set
func (c CatalogState) SetLastUpdated(t time.Time) CatalogState {
	c.LastUpdated = t
	return c
}

// SetNode returns a new ClusterState with the specified node updated
func (c ClusterState) SetNode(id NodeID, n NodeState) ClusterState {
	c.Nodes = maps.Clone(c.Nodes)
	if c.Nodes == nil {
		c.Nodes = map[NodeID]NodeState{}
	}
	c.Nodes[id] = n
	return c
}

// EnsureNode returns a new ClusterState with the specified node present
func (c ClusterState) EnsureNode(id NodeID) ClusterState {
	if _, ok := c.Nodes[id]; ok {
		return c
	}
	return c.SetNode(id, NodeState{
		Health: map[StepID]HealthState{},
	})
}

// SetLastUpdated returns a new ClusterState with the last updated timestamp set
func (c ClusterState) SetLastUpdated(t time.Time) ClusterState {
	c.LastUpdated = t
	return c
}

// SetHealth returns a new NodeState with updated health for a given step
func (n NodeState) SetHealth(id StepID, h HealthState) NodeState {
	n.Health = maps.Clone(n.Health)
	if n.Health == nil {
		n.Health = map[StepID]HealthState{}
	}
	n.Health[id] = h
	return n
}

// SetLastSeen returns a new NodeState with the last seen timestamp set
func (n NodeState) SetLastSeen(t time.Time) NodeState {
	n.LastSeen = t
	return n
}

// GetAttributes returns all attribute values as Args
func (f FlowState) GetAttributes() Args {
	result := make(Args, len(f.Attributes))
	for key, attr := range f.Attributes {
		result[key] = attr.Value
	}
	return result
}

// SetStatus returns a new FlowState with the updated status
func (f FlowState) SetStatus(s FlowStatus) FlowState {
	f.Status = s
	return f
}

// SetAttribute returns a new FlowState with the specified attribute set
func (f FlowState) SetAttribute(name Name, attr *AttributeValue) FlowState {
	f.Attributes = maps.Clone(f.Attributes)
	f.Attributes[name] = attr
	return f
}

func (f FlowState) SetExecution(id StepID, ex ExecutionState) FlowState {
	f.Executions = maps.Clone(f.Executions)
	f.Executions[id] = ex
	return f
}

// SetCompletedAt returns a new FlowState with the completion timestamp set
func (f FlowState) SetCompletedAt(t time.Time) FlowState {
	f.CompletedAt = t
	return f
}

// SetError returns a new FlowState with the error message set
func (f FlowState) SetError(err string) FlowState {
	f.Error = err
	return f
}

// SetLastUpdated returns a new FlowState with last updated time set
func (f FlowState) SetLastUpdated(t time.Time) FlowState {
	f.LastUpdated = t
	return f
}

// SetDeactivatedAt returns a new FlowState with deactivated time set
func (f FlowState) SetDeactivatedAt(t time.Time) FlowState {
	f.DeactivatedAt = t
	return f
}

// SetStatus returns a new ExecutionState with the updated status
func (e ExecutionState) SetStatus(s StepStatus) ExecutionState {
	e.Status = s
	return e
}

// SetStartedAt returns a new ExecutionState with the start timestamp set
func (e ExecutionState) SetStartedAt(t time.Time) ExecutionState {
	e.StartedAt = t
	return e
}

// SetCompletedAt returns a new ExecutionState with completion time set
func (e ExecutionState) SetCompletedAt(t time.Time) ExecutionState {
	e.CompletedAt = t
	return e
}

// SetInputs returns a new ExecutionState with the input arguments set
func (e ExecutionState) SetInputs(inputs Args) ExecutionState {
	e.Inputs = inputs
	return e
}

// SetOutputs returns a new ExecutionState with the output arguments set
func (e ExecutionState) SetOutputs(outputs Args) ExecutionState {
	e.Outputs = outputs
	return e
}

// SetDuration returns a new ExecutionState with the execution duration set
func (e ExecutionState) SetDuration(dur int64) ExecutionState {
	e.Duration = dur
	return e
}

// SetError returns a new ExecutionState with the error message set
func (e ExecutionState) SetError(err string) ExecutionState {
	e.Error = err
	return e
}

// SetWorkItem returns a new ExecutionState with the work item state updated
func (e ExecutionState) SetWorkItem(
	token Token, item WorkState,
) ExecutionState {
	e.WorkItems = maps.Clone(e.WorkItems)
	e.WorkItems[token] = item
	return e
}

// RemoveWorkItem returns a new ExecutionState with the work item removed
func (e ExecutionState) RemoveWorkItem(token Token) ExecutionState {
	e.WorkItems = maps.Clone(e.WorkItems)
	delete(e.WorkItems, token)
	return e
}

// SetStatus returns a new HealthState with the updated status
func (h HealthState) SetStatus(s HealthStatus) HealthState {
	h.Status = s
	return h
}

// SetError returns a new HealthState with the error message set
func (h HealthState) SetError(err string) HealthState {
	h.Error = err
	return h
}

// SetStatus returns a new WorkState with the updated status
func (w WorkState) SetStatus(s WorkStatus) WorkState {
	w.Status = s
	return w
}

// SetStartedAt returns a new WorkState with the started timestamp set
func (w WorkState) SetStartedAt(t time.Time) WorkState {
	w.StartedAt = t
	return w
}

// SetCompletedAt returns a new WorkState with the completed timestamp set
func (w WorkState) SetCompletedAt(t time.Time) WorkState {
	w.CompletedAt = t
	return w
}

// SetRetryCount returns a new WorkState with the retry count set
func (w WorkState) SetRetryCount(count int) WorkState {
	w.RetryCount = count
	return w
}

// SetNextRetryAt returns a new WorkState with the next retry time set
func (w WorkState) SetNextRetryAt(t time.Time) WorkState {
	w.NextRetryAt = t
	return w
}

// SetError returns a new WorkState with the error message set
func (w WorkState) SetError(err string) WorkState {
	w.Error = err
	return w
}

// SetOutputs returns a new WorkState with the outputs set
func (w WorkState) SetOutputs(outputs Args) WorkState {
	w.Outputs = outputs
	return w
}
