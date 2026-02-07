package builder

import (
	"context"
	"fmt"

	"github.com/kode4food/argyll/engine/pkg/api"
)

type (
	// StepHandler is the function signature for step implementations. It
	// receives a StepContext which includes both context and flow client
	StepHandler func(*StepContext, api.Args) (api.StepResult, error)

	// StepContext provides context and client capabilities to step handlers
	StepContext struct {
		// Context is the standard Go context for cancellation and deadlines
		context.Context

		// Client provides access to the current flow's state and operations
		Client *FlowClient

		// StepID is the ID of the current step being executed
		StepID api.StepID

		// Metadata contains additional context passed to step handlers
		Metadata api.Metadata
	}

	// HTTPError allows step handlers to return specific HTTP status codes
	HTTPError struct {
		StatusCode int
		Message    string
	}
)

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
