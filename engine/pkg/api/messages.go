package api

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrFlowIDEmpty   = errors.New("flow ID empty")
	ErrFlowIDInvalid = errors.New("flow ID contains invalid characters")
	ErrFlowIDTooLong = errors.New("flow ID too long")
	ErrGoalsRequired = errors.New("at least one goal step is required")
	ErrTooManyGoals  = errors.New("too many goals")
	ErrTooManyInit   = errors.New("too many init keys")
	ErrTooManyLabels = errors.New("too many labels")
)

const (
	MaxFlowIDLen  = 256
	MaxGoalCount  = 64
	MaxInitKeys   = 128
	MaxLabelCount = 32
)

type (
	// CreateFlowRequest contains parameters for starting a new flow
	CreateFlowRequest struct {
		Init   Args     `json:"init"`
		ID     FlowID   `json:"id"`
		Labels Labels   `json:"labels,omitempty"`
		Goals  []StepID `json:"goals"`
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
		Status      FlowStatus `json:"status"`
		CreatedAt   time.Time  `json:"created_at"`
		CompletedAt time.Time  `json:"completed_at"`
		Labels      Labels     `json:"labels,omitempty"`
		Error       string     `json:"error,omitempty"`
	}

	// QueryFlowsRequest contains filter criteria and pagination options
	QueryFlowsRequest struct {
		IDPrefix string       `json:"id_prefix,omitempty"`
		Labels   Labels       `json:"labels,omitempty"`
		Statuses []FlowStatus `json:"statuses,omitempty"`
		Limit    int          `json:"limit,omitempty"`
		Cursor   string       `json:"cursor,omitempty"`
		Sort     FlowSort     `json:"sort,omitempty"`
	}

	// QueryFlowsResponse contains a list of flow summaries
	QueryFlowsResponse struct {
		Flows      []*QueryFlowsItem `json:"flows"`
		Count      int               `json:"count"`
		Total      int               `json:"total,omitempty"`
		HasMore    bool              `json:"has_more,omitempty"`
		NextCursor string            `json:"next_cursor,omitempty"`
	}

	// QueryFlowsItem provides a flow ID with its summary information
	QueryFlowsItem struct {
		ID     FlowID      `json:"id"`
		Digest *FlowDigest `json:"digest"`
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

	// FlowSort determines flow query ordering
	FlowSort string
)

const (
	FlowSortRecentDesc FlowSort = "recent_desc"
	FlowSortRecentAsc  FlowSort = "recent_asc"
)

// Validate checks that the request has a valid flow ID and at least one goal
func (r *CreateFlowRequest) Validate() error {
	if r.ID == "" {
		return ErrFlowIDEmpty
	}
	if SanitizeID(r.ID) != r.ID {
		return ErrFlowIDInvalid
	}
	if len(r.ID) > MaxFlowIDLen {
		return fmt.Errorf("%w: maximum is %d", ErrFlowIDTooLong, MaxFlowIDLen)
	}
	if len(r.Goals) == 0 {
		return ErrGoalsRequired
	}
	if len(r.Goals) > MaxGoalCount {
		return fmt.Errorf("%w: maximum is %d", ErrTooManyGoals, MaxGoalCount)
	}
	if len(r.Init) > MaxInitKeys {
		return fmt.Errorf("%w: maximum is %d", ErrTooManyInit, MaxInitKeys)
	}
	if len(r.Labels) > MaxLabelCount {
		return fmt.Errorf("%w: maximum is %d", ErrTooManyLabels, MaxLabelCount)
	}
	return nil
}
