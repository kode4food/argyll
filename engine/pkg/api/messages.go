package api

import (
	"time"

	"github.com/kode4food/timebox"
)

type (
	CreateWorkflowRequest struct {
		InitialState Args         `json:"initial_state"`
		ID           timebox.ID   `json:"id"`
		GoalStepIDs  []timebox.ID `json:"goal_steps"`
	}

	UpdateStateRequest struct {
		Updates Args `json:"updates"`
	}

	ExecutionPlanRequest struct {
		InitialState Args         `json:"initial_state"`
		GoalStepIDs  []timebox.ID `json:"goal_steps"`
	}

	WorkflowStartedResponse struct {
		Message    string     `json:"message"`
		WorkflowID timebox.ID `json:"workflow_id"`
	}

	WorkflowDigest struct {
		ID          timebox.ID     `json:"id"`
		Status      WorkflowStatus `json:"status"`
		CreatedAt   time.Time      `json:"created_at"`
		CompletedAt time.Time      `json:"completed_at,omitempty"`
		Error       string         `json:"error,omitempty"`
	}

	WorkflowsListResponse struct {
		Workflows []*WorkflowDigest `json:"workflows"`
		Count     int               `json:"count"`
	}

	StepRegisteredResponse struct {
		Step    *Step  `json:"step"`
		Message string `json:"message"`
	}

	StepsListResponse struct {
		Steps []*Step `json:"steps"`
		Count int     `json:"count"`
	}

	HealthResponse struct {
		Service string `json:"service"`
		Version string `json:"version"`
		HealthState
	}

	StatusResponse struct {
		WorkflowFingerprint string `json:"workflow_fingerprint"`
		StepCount           int    `json:"step_count"`
		WorkflowCount       int    `json:"workflow_count"`
	}

	HealthListResponse struct {
		Health map[timebox.ID]*HealthState `json:"health"`
		Count  int                         `json:"count"`
	}

	MessageResponse struct {
		Message string `json:"message"`
	}

	ErrorResponse struct {
		Error   string `json:"error"`
		Details string `json:"details,omitempty"`
		Status  int    `json:"status,omitempty"`
	}
)
