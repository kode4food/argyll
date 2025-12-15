package api

import "time"

type (
	// CreateFlowRequest contains parameters for starting a new flow
	CreateFlowRequest struct {
		Init  Args     `json:"init"`
		ID    FlowID   `json:"id"`
		Goals []StepID `json:"goals"`
	}

	// ExecutionPlanRequest contains parameters for creating an execution plan
	ExecutionPlanRequest struct {
		Init  Args     `json:"init"`
		Goals []StepID `json:"goals"`
	}

	// FlowStartedResponse is returned when a flow start succeeds
	FlowStartedResponse struct {
		Message string `json:"message"`
		FlowID  FlowID `json:"flow_id"`
	}

	// FlowDigest provides summary information about a flow
	FlowDigest struct {
		ID          FlowID     `json:"id"`
		Status      FlowStatus `json:"status"`
		CreatedAt   time.Time  `json:"created_at"`
		CompletedAt time.Time  `json:"completed_at"`
		Error       string     `json:"error,omitempty"`
	}

	// FlowsListResponse contains a list of flow summaries
	FlowsListResponse struct {
		Flows []*FlowDigest `json:"flows"`
		Count int           `json:"count"`
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

	// HealthResponse provides service health information
	HealthResponse struct {
		Service string `json:"service"`
		HealthState
	}

	// HealthListResponse contains health status for all registered steps
	HealthListResponse struct {
		Health map[StepID]*HealthState `json:"health"`
		Count  int                     `json:"count"`
	}

	// MessageResponse contains a simple message string
	MessageResponse struct {
		Message string `json:"message"`
	}

	// ErrorResponse contains error details for failed requests
	ErrorResponse struct {
		Error  string `json:"error"`
		Status int    `json:"status,omitempty"`
	}
)
