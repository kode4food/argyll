package api

import (
	"maps"
	"time"

	"github.com/kode4food/timebox"
)

type (
	// WorkflowStatus represents the current state of a workflow
	WorkflowStatus string

	// StepStatus represents the current state of a step execution
	StepStatus string

	// HealthStatus represents the health of a step service
	HealthStatus string

	// WorkStatus represents the state of a single work item
	WorkStatus string

	// Token uniquely identifies a work item within a step
	Token string

	// EngineState contains the global state of the workflow engine
	EngineState struct {
		LastUpdated     time.Time                          `json:"last_updated"`
		Steps           map[timebox.ID]*Step               `json:"steps"`
		Health          map[timebox.ID]*HealthState        `json:"health"`
		ActiveWorkflows map[timebox.ID]*ActiveWorkflowInfo `json:"active_workflows"`
	}

	// ActiveWorkflowInfo tracks basic metadata for active workflows
	ActiveWorkflowInfo struct {
		FlowID     timebox.ID `json:"flow_id"`
		StartedAt  time.Time  `json:"started_at"`
		LastActive time.Time  `json:"last_active"`
	}

	// WorkflowState contains the complete state of a workflow execution
	WorkflowState struct {
		CreatedAt   time.Time                      `json:"created_at"`
		CompletedAt time.Time                      `json:"completed_at,omitempty"`
		LastUpdated time.Time                      `json:"last_updated"`
		Plan        *ExecutionPlan                 `json:"plan"`
		Attributes  map[Name]*AttributeValue       `json:"attributes"`
		Executions  map[timebox.ID]*ExecutionState `json:"executions"`
		ID          timebox.ID                     `json:"id"`
		Status      WorkflowStatus                 `json:"status"`
		Error       string                         `json:"error,omitempty"`
	}

	// AttributeValue stores an attribute value and which step produced it
	AttributeValue struct {
		Value any        `json:"value"`
		Step  timebox.ID `json:"step,omitempty"`
	}

	// ExecutionState contains the state of a step execution
	ExecutionState struct {
		StartedAt   time.Time            `json:"started_at"`
		CompletedAt time.Time            `json:"completed_at,omitempty"`
		Inputs      Args                 `json:"inputs"`
		Outputs     Args                 `json:"outputs,omitempty"`
		Status      StepStatus           `json:"status"`
		Error       string               `json:"error,omitempty"`
		Duration    int64                `json:"duration,omitempty"`
		WorkItems   map[Token]*WorkState `json:"work_items,omitempty"`
	}

	// WorkState contains the state of a single work item
	WorkState struct {
		Status      WorkStatus `json:"status"`
		StartedAt   time.Time  `json:"started_at"`
		CompletedAt time.Time  `json:"completed_at,omitempty"`
		Inputs      Args       `json:"inputs"`
		Outputs     Args       `json:"outputs,omitempty"`
		Error       string     `json:"error,omitempty"`
		RetryCount  int        `json:"retry_count,omitempty"`
		NextRetryAt time.Time  `json:"next_retry_at,omitempty"`
		LastError   string     `json:"last_error,omitempty"`
	}

	// HealthState contains the health status of a step service
	HealthState struct {
		Status HealthStatus `json:"status"`
		Error  string       `json:"error,omitempty"`
	}
)

const (
	WorkflowPending   WorkflowStatus = "pending"
	WorkflowActive    WorkflowStatus = "active"
	WorkflowCompleted WorkflowStatus = "completed"
	WorkflowFailed    WorkflowStatus = "failed"
)

const (
	StepPending   StepStatus = "pending"
	StepActive    StepStatus = "active"
	StepCompleted StepStatus = "completed"
	StepSkipped   StepStatus = "skipped"
	StepFailed    StepStatus = "failed"
)

const (
	WorkPending   WorkStatus = "pending"
	WorkActive    WorkStatus = "active"
	WorkCompleted WorkStatus = "completed"
	WorkFailed    WorkStatus = "failed"
)

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

// SetStep returns a new EngineState with the specified step registered
func (st *EngineState) SetStep(id timebox.ID, step *Step) *EngineState {
	res := *st
	res.Steps = maps.Clone(st.Steps)
	res.Steps[id] = step
	return &res
}

// DeleteStep returns a new EngineState with the specified step removed
func (st *EngineState) DeleteStep(i timebox.ID) *EngineState {
	res := *st
	res.Steps = maps.Clone(st.Steps)
	delete(res.Steps, i)
	return &res
}

// SetHealth returns a new EngineState with updated health for a given step
func (st *EngineState) SetHealth(id timebox.ID, h *HealthState) *EngineState {
	res := *st
	res.Health = maps.Clone(st.Health)
	res.Health[id] = h
	return &res
}

// SetLastUpdated returns a new EngineState with the last updated timestamp set
func (st *EngineState) SetLastUpdated(t time.Time) *EngineState {
	res := *st
	res.LastUpdated = t
	return &res
}

// SetActiveWorkflow returns a new EngineState with the workflow as active
func (st *EngineState) SetActiveWorkflow(
	id timebox.ID, info *ActiveWorkflowInfo,
) *EngineState {
	res := *st
	res.ActiveWorkflows = maps.Clone(st.ActiveWorkflows)
	res.ActiveWorkflows[id] = info
	return &res
}

// DeleteActiveWorkflow returns a new EngineState with the workflow inactive
func (st *EngineState) DeleteActiveWorkflow(id timebox.ID) *EngineState {
	res := *st
	res.ActiveWorkflows = maps.Clone(st.ActiveWorkflows)
	delete(res.ActiveWorkflows, id)
	return &res
}

// SetStatus returns a new WorkflowState with the updated status
func (st *WorkflowState) SetStatus(s WorkflowStatus) *WorkflowState {
	res := *st
	res.Status = s
	return &res
}

// SetAttribute returns a new WorkflowState with the specified attribute set
func (st *WorkflowState) SetAttribute(
	name Name, attr *AttributeValue,
) *WorkflowState {
	res := *st
	res.Attributes = maps.Clone(st.Attributes)
	res.Attributes[name] = attr
	return &res
}

// SetExecution returns a new WorkflowState with updated execution for a step
func (st *WorkflowState) SetExecution(
	id timebox.ID, ex *ExecutionState,
) *WorkflowState {
	res := *st
	res.Executions = maps.Clone(st.Executions)
	res.Executions[id] = ex
	return &res
}

// SetCompletedAt returns a new WorkflowState with the completion timestamp set
func (st *WorkflowState) SetCompletedAt(t time.Time) *WorkflowState {
	res := *st
	res.CompletedAt = t
	return &res
}

// SetError returns a new WorkflowState with the error message set
func (st *WorkflowState) SetError(err string) *WorkflowState {
	res := *st
	res.Error = err
	return &res
}

// SetLastUpdated returns a new WorkflowState with last updated time set
func (st *WorkflowState) SetLastUpdated(t time.Time) *WorkflowState {
	res := *st
	res.LastUpdated = t
	return &res
}

// SetStatus returns a new ExecutionState with the updated status
func (st *ExecutionState) SetStatus(s StepStatus) *ExecutionState {
	res := *st
	res.Status = s
	return &res
}

// SetStartedAt returns a new ExecutionState with the start timestamp set
func (st *ExecutionState) SetStartedAt(t time.Time) *ExecutionState {
	res := *st
	res.StartedAt = t
	return &res
}

// SetCompletedAt returns a new ExecutionState with completion time set
func (st *ExecutionState) SetCompletedAt(t time.Time) *ExecutionState {
	res := *st
	res.CompletedAt = t
	return &res
}

// SetInputs returns a new ExecutionState with the input arguments set
func (st *ExecutionState) SetInputs(inputs Args) *ExecutionState {
	res := *st
	res.Inputs = maps.Clone(inputs)
	return &res
}

// SetOutputs returns a new ExecutionState with the output arguments set
func (st *ExecutionState) SetOutputs(outputs Args) *ExecutionState {
	res := *st
	res.Outputs = maps.Clone(outputs)
	return &res
}

// SetDuration returns a new ExecutionState with the execution duration set
func (st *ExecutionState) SetDuration(dur int64) *ExecutionState {
	res := *st
	res.Duration = dur
	return &res
}

// SetError returns a new ExecutionState with the error message set
func (st *ExecutionState) SetError(err string) *ExecutionState {
	res := *st
	res.Error = err
	return &res
}

// SetWorkItem returns a new ExecutionState with the work item state updated
func (st *ExecutionState) SetWorkItem(
	token Token, item *WorkState,
) *ExecutionState {
	res := *st
	res.WorkItems = maps.Clone(st.WorkItems)
	if res.WorkItems == nil {
		res.WorkItems = map[Token]*WorkState{}
	}
	res.WorkItems[token] = item
	return &res
}

// SetStatus returns a new HealthState with the updated status
func (st *HealthState) SetStatus(s HealthStatus) *HealthState {
	res := *st
	res.Status = s
	return &res
}

// SetError returns a new HealthState with the error message set
func (st *HealthState) SetError(err string) *HealthState {
	res := *st
	res.Error = err
	return &res
}

// SetStatus returns a new WorkState with the updated status
func (st *WorkState) SetStatus(s WorkStatus) *WorkState {
	res := *st
	res.Status = s
	return &res
}

// SetRetryCount returns a new WorkState with the retry count set
func (st *WorkState) SetRetryCount(count int) *WorkState {
	res := *st
	res.RetryCount = count
	return &res
}

// SetNextRetryAt returns a new WorkState with the next retry time set
func (st *WorkState) SetNextRetryAt(t time.Time) *WorkState {
	res := *st
	res.NextRetryAt = t
	return &res
}

// SetLastError returns a new WorkState with the last error message set
func (st *WorkState) SetLastError(err string) *WorkState {
	res := *st
	res.LastError = err
	return &res
}

// GetAttributeArgs returns all attribute values as Args
func (st *WorkflowState) GetAttributeArgs() Args {
	result := make(Args, len(st.Attributes))
	for key, attr := range st.Attributes {
		result[key] = attr.Value
	}
	return result
}
