package api

import (
	"time"

	"github.com/kode4food/timebox"
)

type (
	// CreateWorkflowRequest contains parameters for starting a new workflow
	CreateWorkflowRequest struct {
		Init  Args         `json:"init"`
		ID    timebox.ID   `json:"id"`
		Goals []timebox.ID `json:"goals"`
	}

	// UpdateStateRequest contains attribute updates for a workflow
	UpdateStateRequest struct {
		Updates Args `json:"updates"`
	}

	// ExecutionPlanRequest contains parameters for creating an execution plan
	ExecutionPlanRequest struct {
		Init  Args         `json:"init"`
		Goals []timebox.ID `json:"goals"`
	}

	// WorkflowStartedResponse is returned when a workflow start succeeds
	WorkflowStartedResponse struct {
		Message string     `json:"message"`
		FlowID  timebox.ID `json:"flow_id"`
	}

	// WorkflowDigest provides summary information about a workflow
	WorkflowDigest struct {
		ID          timebox.ID     `json:"id"`
		Status      WorkflowStatus `json:"status"`
		CreatedAt   time.Time      `json:"created_at"`
		CompletedAt time.Time      `json:"completed_at,omitempty"`
		Error       string         `json:"error,omitempty"`
	}

	// WorkflowsListResponse contains a list of workflow summaries
	WorkflowsListResponse struct {
		Workflows []*WorkflowDigest `json:"workflows"`
		Count     int               `json:"count"`
	}

	// StepRegisteredResponse is returned when a step registration succeeds
	StepRegisteredResponse struct {
		Step    *Step  `json:"step"`
		Message string `json:"message"`
	}

	// StepsListResponse contains a list of registered steps
	StepsListResponse struct {
		Steps []*Step `json:"steps"`
		Count int     `json:"count"`
	}

	// HealthResponse provides service health and version information
	HealthResponse struct {
		Service string `json:"service"`
		Version string `json:"version"`
		HealthState
	}

	// StatusResponse provides engine status and statistics
	StatusResponse struct {
		WorkflowFingerprint string `json:"workflow_fingerprint"`
		StepCount           int    `json:"step_count"`
		WorkflowCount       int    `json:"workflow_count"`
	}

	// HealthListResponse contains health status for all registered steps
	HealthListResponse struct {
		Health map[timebox.ID]*HealthState `json:"health"`
		Count  int                         `json:"count"`
	}

	// MessageResponse contains a simple message string
	MessageResponse struct {
		Message string `json:"message"`
	}

	// ErrorResponse contains error details for failed requests
	ErrorResponse struct {
		Error   string `json:"error"`
		Details string `json:"details,omitempty"`
		Status  int    `json:"status,omitempty"`
	}
)
