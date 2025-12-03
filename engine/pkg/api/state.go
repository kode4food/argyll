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

	// EngineState contains the global state of the orchestrator
	EngineState struct {
		LastUpdated time.Time                  `json:"last_updated"`
		Steps       Steps                      `json:"steps"`
		Health      map[StepID]*HealthState    `json:"health"`
		ActiveFlows map[FlowID]*ActiveFlowInfo `json:"active_flows"`
		Attributes  AttributeGraph             `json:"attributes"`
	}

	// ActiveFlowInfo tracks basic metadata for active flows
	ActiveFlowInfo struct {
		FlowID     FlowID    `json:"flow_id"`
		StartedAt  time.Time `json:"started_at"`
		LastActive time.Time `json:"last_active"`
	}

	// FlowState contains the complete state of a flow execution
	FlowState struct {
		CreatedAt   time.Time       `json:"created_at"`
		CompletedAt time.Time       `json:"completed_at"`
		LastUpdated time.Time       `json:"last_updated"`
		Plan        *ExecutionPlan  `json:"plan"`
		Attributes  AttributeValues `json:"attributes"`
		Executions  Executions      `json:"executions"`
		ID          FlowID          `json:"id"`
		Status      FlowStatus      `json:"status"`
		Error       string          `json:"error,omitempty"`
	}

	// Executions contains the execution progress of multiple steps
	Executions map[StepID]*ExecutionState

	// AttributeValues contains fulfilled attribute values and their sources
	AttributeValues map[Name]*AttributeValue

	// AttributeValue stores an attribute value and which step produced it
	AttributeValue struct {
		Value any    `json:"value"`
		Step  StepID `json:"step,omitempty"`
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
	FlowPending   FlowStatus = "pending"
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

// SetStep returns a new EngineState with the specified step registered
func (e *EngineState) SetStep(id StepID, step *Step) *EngineState {
	res := *e
	res.Steps = maps.Clone(e.Steps)
	res.Steps[id] = step
	res.Attributes = maps.Clone(e.Attributes)

	if oldStep, ok := e.Steps[id]; ok {
		res.Attributes.RemoveStep(id, oldStep)
	}

	res.Attributes.AddStep(id, step)
	return &res
}

// DeleteStep returns a new EngineState with the specified step removed
func (e *EngineState) DeleteStep(id StepID) *EngineState {
	step, ok := e.Steps[id]
	if !ok {
		return e
	}

	res := *e
	res.Steps = maps.Clone(e.Steps)
	delete(res.Steps, id)
	res.Attributes = maps.Clone(e.Attributes)
	res.Attributes.RemoveStep(id, step)
	return &res
}

// SetHealth returns a new EngineState with updated health for a given step
func (e *EngineState) SetHealth(id StepID, h *HealthState) *EngineState {
	res := *e
	res.Health = maps.Clone(e.Health)
	res.Health[id] = h
	return &res
}

// SetLastUpdated returns a new EngineState with the last updated timestamp set
func (e *EngineState) SetLastUpdated(t time.Time) *EngineState {
	res := *e
	res.LastUpdated = t
	return &res
}

// SetActiveFlow returns a new EngineState with the flow as active
func (e *EngineState) SetActiveFlow(
	id FlowID, info *ActiveFlowInfo,
) *EngineState {
	res := *e
	res.ActiveFlows = maps.Clone(e.ActiveFlows)
	res.ActiveFlows[id] = info
	return &res
}

// DeleteActiveFlow returns a new EngineState with the flow inactive
func (e *EngineState) DeleteActiveFlow(id FlowID) *EngineState {
	res := *e
	res.ActiveFlows = maps.Clone(e.ActiveFlows)
	delete(res.ActiveFlows, id)
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
