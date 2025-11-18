package api

import (
	"maps"
	"time"

	"github.com/kode4food/timebox"
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
		LastUpdated time.Time                      `json:"last_updated"`
		Steps       map[timebox.ID]*Step           `json:"steps"`
		Health      map[timebox.ID]*HealthState    `json:"health"`
		ActiveFlows map[timebox.ID]*ActiveFlowInfo `json:"active_flows"`
	}

	// ActiveFlowInfo tracks basic metadata for active flows
	ActiveFlowInfo struct {
		FlowID     timebox.ID `json:"flow_id"`
		StartedAt  time.Time  `json:"started_at"`
		LastActive time.Time  `json:"last_active"`
	}

	// FlowState contains the complete state of a flow execution
	FlowState struct {
		CreatedAt   time.Time                      `json:"created_at"`
		CompletedAt time.Time                      `json:"completed_at,omitempty"`
		LastUpdated time.Time                      `json:"last_updated"`
		Plan        *ExecutionPlan                 `json:"plan"`
		Attributes  map[Name]*AttributeValue       `json:"attributes"`
		Executions  map[timebox.ID]*ExecutionState `json:"executions"`
		ID          timebox.ID                     `json:"id"`
		Status      FlowStatus                     `json:"status"`
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

// SetActiveFlow returns a new EngineState with the flow as active
func (st *EngineState) SetActiveFlow(
	id timebox.ID, info *ActiveFlowInfo,
) *EngineState {
	res := *st
	res.ActiveFlows = maps.Clone(st.ActiveFlows)
	res.ActiveFlows[id] = info
	return &res
}

// DeleteActiveFlow returns a new EngineState with the flow inactive
func (st *EngineState) DeleteActiveFlow(id timebox.ID) *EngineState {
	res := *st
	res.ActiveFlows = maps.Clone(st.ActiveFlows)
	delete(res.ActiveFlows, id)
	return &res
}

// SetStatus returns a new FlowState with the updated status
func (st *FlowState) SetStatus(s FlowStatus) *FlowState {
	res := *st
	res.Status = s
	return &res
}

// SetAttribute returns a new FlowState with the specified attribute set
func (st *FlowState) SetAttribute(name Name, attr *AttributeValue) *FlowState {
	res := *st
	res.Attributes = maps.Clone(st.Attributes)
	res.Attributes[name] = attr
	return &res
}

// SetExecution returns a new FlowState with updated execution for a step
func (st *FlowState) SetExecution(
	id timebox.ID, ex *ExecutionState,
) *FlowState {
	res := *st
	res.Executions = maps.Clone(st.Executions)
	res.Executions[id] = ex
	return &res
}

// SetCompletedAt returns a new FlowState with the completion timestamp set
func (st *FlowState) SetCompletedAt(t time.Time) *FlowState {
	res := *st
	res.CompletedAt = t
	return &res
}

// SetError returns a new FlowState with the error message set
func (st *FlowState) SetError(err string) *FlowState {
	res := *st
	res.Error = err
	return &res
}

// SetLastUpdated returns a new FlowState with last updated time set
func (st *FlowState) SetLastUpdated(t time.Time) *FlowState {
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
	res.Inputs = inputs
	return &res
}

// SetOutputs returns a new ExecutionState with the output arguments set
func (st *ExecutionState) SetOutputs(outputs Args) *ExecutionState {
	res := *st
	res.Outputs = outputs
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
func (st *FlowState) GetAttributeArgs() Args {
	result := make(Args, len(st.Attributes))
	for key, attr := range st.Attributes {
		result[key] = attr.Value
	}
	return result
}
