package argyll

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// InvokeHandler runs a step with its context and inputs
	InvokeHandler func(StepContext, api.Args) (api.Args, error)

	// CompensateHandler undoes a completed work item
	CompensateHandler func(StepContext, api.Args) error

	// StepContext provides context and execution metadata to step handlers
	StepContext interface {
		// Context is the standard Go context for cancellation and deadlines
		context.Context

		// FlowID is the ID of the current flow
		FlowID() api.FlowID

		// StepID is the ID of the current step being executed
		StepID() api.StepID

		// Metadata contains additional context passed to step handlers
		Metadata() api.Metadata
	}

	// HTTPError allows step handlers to return specific HTTP status codes
	HTTPError struct {
		StatusCode int
		Message    string
	}

	// HTTPErrorFunc formats an HTTPError with a predefined status
	HTTPErrorFunc func(string, ...any) *HTTPError

	stepContext struct {
		context.Context
		flowID   api.FlowID
		stepID   api.StepID
		metadata api.Metadata
	}
)

var (
	// BadRequest formats an error for an invalid request
	BadRequest = MakeHTTPError(http.StatusBadRequest)

	// BadGateway formats an error for a failed upstream service
	BadGateway = MakeHTTPError(http.StatusBadGateway)

	// NotFound formats an error for a missing resource
	NotFound = MakeHTTPError(http.StatusNotFound)

	// Conflict formats an error for a conflicting operation
	Conflict = MakeHTTPError(http.StatusConflict)

	// InternalServerError formats an error for an internal failure
	InternalServerError = MakeHTTPError(http.StatusInternalServerError)

	// ServiceUnavailable formats an error for an unavailable service
	ServiceUnavailable = MakeHTTPError(http.StatusServiceUnavailable)
)

// NewStepContext creates a handler context for the current work item
func NewStepContext(
	ctx context.Context, fid api.FlowID, sid api.StepID, meta api.Metadata,
) StepContext {
	return &stepContext{
		Context:  ctx,
		flowID:   fid,
		stepID:   sid,
		metadata: meta,
	}
}

// MakeHTTPError creates an error formatter with a predefined HTTP status
func MakeHTTPError(status int) HTTPErrorFunc {
	return func(format string, args ...any) *HTTPError {
		return NewHTTPError(status, fmt.Sprintf(format, args...))
	}
}

// NewHTTPError creates a new HTTPError with the given status code and message
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// Error implements the error interface for HTTPError
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// FlowID returns the ID of this work item's flow
func (s *stepContext) FlowID() api.FlowID {
	return s.flowID
}

// StepID returns the ID of the step running this work item
func (s *stepContext) StepID() api.StepID {
	return s.stepID
}

// Metadata returns the metadata for this work item
func (s *stepContext) Metadata() api.Metadata {
	return s.metadata
}
