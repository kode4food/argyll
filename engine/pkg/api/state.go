package api

import (
	"maps"
	"time"

	"github.com/kode4food/timebox"
)

type (
	WorkflowStatus string
	StepStatus     string
	HealthStatus   string
	WorkStatus     string

	EngineState struct {
		LastUpdated     time.Time                          `json:"last_updated"`
		Steps           map[timebox.ID]*Step               `json:"steps"`
		Health          map[timebox.ID]*HealthState        `json:"health"`
		ActiveWorkflows map[timebox.ID]*ActiveWorkflowInfo `json:"active_workflows"`
	}

	ActiveWorkflowInfo struct {
		WorkflowID timebox.ID `json:"workflow_id"`
		StartedAt  time.Time  `json:"started_at"`
		LastActive time.Time  `json:"last_active"`
	}

	WorkflowState struct {
		CreatedAt     time.Time                      `json:"created_at"`
		CompletedAt   time.Time                      `json:"completed_at,omitempty"`
		LastUpdated   time.Time                      `json:"last_updated"`
		ExecutionPlan *ExecutionPlan                 `json:"execution_plan"`
		Attributes    map[Name]*AttributeValue       `json:"attributes"`
		Executions    map[timebox.ID]*ExecutionState `json:"executions"`
		ID            timebox.ID                     `json:"id"`
		Status        WorkflowStatus                 `json:"status"`
		Error         string                         `json:"error,omitempty"`
	}

	AttributeValue struct {
		Value any        `json:"value"`
		Step  timebox.ID `json:"step,omitempty"`
	}

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

func (st *EngineState) SetStep(id timebox.ID, step *Step) *EngineState {
	res := *st
	res.Steps = maps.Clone(st.Steps)
	res.Steps[id] = step
	return &res
}

func (st *EngineState) DeleteStep(i timebox.ID) *EngineState {
	res := *st
	res.Steps = maps.Clone(st.Steps)
	delete(res.Steps, i)
	return &res
}

func (st *EngineState) SetHealth(id timebox.ID, h *HealthState) *EngineState {
	res := *st
	res.Health = maps.Clone(st.Health)
	res.Health[id] = h
	return &res
}

func (st *EngineState) SetLastUpdated(t time.Time) *EngineState {
	res := *st
	res.LastUpdated = t
	return &res
}

func (st *EngineState) SetActiveWorkflow(
	id timebox.ID, info *ActiveWorkflowInfo,
) *EngineState {
	res := *st
	res.ActiveWorkflows = maps.Clone(st.ActiveWorkflows)
	res.ActiveWorkflows[id] = info
	return &res
}

func (st *EngineState) DeleteActiveWorkflow(id timebox.ID) *EngineState {
	res := *st
	res.ActiveWorkflows = maps.Clone(st.ActiveWorkflows)
	delete(res.ActiveWorkflows, id)
	return &res
}

func (st *WorkflowState) SetStatus(s WorkflowStatus) *WorkflowState {
	res := *st
	res.Status = s
	return &res
}

func (st *WorkflowState) SetAttribute(
	name Name, attr *AttributeValue,
) *WorkflowState {
	res := *st
	res.Attributes = maps.Clone(st.Attributes)
	res.Attributes[name] = attr
	return &res
}

func (st *WorkflowState) SetExecution(
	id timebox.ID, ex *ExecutionState,
) *WorkflowState {
	res := *st
	res.Executions = maps.Clone(st.Executions)
	res.Executions[id] = ex
	return &res
}

func (st *WorkflowState) SetCompletedAt(t time.Time) *WorkflowState {
	res := *st
	res.CompletedAt = t
	return &res
}

func (st *WorkflowState) SetError(err string) *WorkflowState {
	res := *st
	res.Error = err
	return &res
}

func (st *WorkflowState) SetLastUpdated(t time.Time) *WorkflowState {
	res := *st
	res.LastUpdated = t
	return &res
}

func (st *ExecutionState) SetStatus(s StepStatus) *ExecutionState {
	res := *st
	res.Status = s
	return &res
}

func (st *ExecutionState) SetStartedAt(t time.Time) *ExecutionState {
	res := *st
	res.StartedAt = t
	return &res
}

func (st *ExecutionState) SetCompletedAt(t time.Time) *ExecutionState {
	res := *st
	res.CompletedAt = t
	return &res
}

func (st *ExecutionState) SetInputs(inputs Args) *ExecutionState {
	res := *st
	res.Inputs = maps.Clone(inputs)
	return &res
}

func (st *ExecutionState) SetOutputs(outputs Args) *ExecutionState {
	res := *st
	res.Outputs = maps.Clone(outputs)
	return &res
}

func (st *ExecutionState) SetDuration(dur int64) *ExecutionState {
	res := *st
	res.Duration = dur
	return &res
}

func (st *ExecutionState) SetError(err string) *ExecutionState {
	res := *st
	res.Error = err
	return &res
}

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

func (st *HealthState) SetStatus(s HealthStatus) *HealthState {
	res := *st
	res.Status = s
	return &res
}

func (st *HealthState) SetError(err string) *HealthState {
	res := *st
	res.Error = err
	return &res
}

func (st *WorkState) SetStatus(s WorkStatus) *WorkState {
	res := *st
	res.Status = s
	return &res
}

func (st *WorkState) SetRetryCount(count int) *WorkState {
	res := *st
	res.RetryCount = count
	return &res
}

func (st *WorkState) SetNextRetryAt(t time.Time) *WorkState {
	res := *st
	res.NextRetryAt = t
	return &res
}

func (st *WorkState) SetLastError(err string) *WorkState {
	res := *st
	res.LastError = err
	return &res
}

func (st *WorkflowState) GetAttributeArgs() Args {
	result := make(Args, len(st.Attributes))
	for key, attr := range st.Attributes {
		result[key] = attr.Value
	}
	return result
}
