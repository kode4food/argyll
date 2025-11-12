package builder

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kode4food/timebox"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type (

	// StepContext provides a rich context for step execution with easy access
	// to workflow state, metadata, and operations
	StepContext struct {
		ctx      context.Context
		workflow *WorkflowClient
		args     api.Args
		metadata api.Metadata
		logger   *slog.Logger
	}

	// StepHandlerFunc is a step handler function that receives a StepContext
	StepHandlerFunc func(*StepContext) (api.StepResult, error)
)

// Context returns the underlying context.Context
func (s *StepContext) Context() context.Context {
	return s.ctx
}

// FlowID returns the workflow ID for this step execution
func (s *StepContext) FlowID() timebox.ID {
	return getMetadataID(s.metadata, "flow_id")
}

// StepID returns the step definition ID
func (s *StepContext) StepID() timebox.ID {
	return getMetadataID(s.metadata, "step_id")
}

// WorkToken returns the unique work item token for this execution
func (s *StepContext) WorkToken() api.Token {
	if token, ok := s.metadata["receipt_token"].(api.Token); ok {
		return token
	}
	return ""
}

// Args returns all step input arguments
func (s *StepContext) Args() api.Args {
	return s.args
}

// Get retrieves an argument value by name
func (s *StepContext) Get(name api.Name) (any, bool) {
	val, ok := s.args[name]
	return val, ok
}

// GetString retrieves a string argument, returning empty string if not found
func (s *StepContext) GetString(name api.Name) string {
	if val, ok := s.args[name]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt retrieves an integer argument, returning 0 if not found
func (s *StepContext) GetInt(name api.Name) int {
	if val, ok := s.args[name]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case int64:
			return int(v)
		}
	}
	return 0
}

// GetFloat retrieves a float64 argument, returning 0.0 if not found
func (s *StepContext) GetFloat(name api.Name) float64 {
	if val, ok := s.args[name]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0.0
}

// GetBool retrieves a boolean argument, returning false if not found
func (s *StepContext) GetBool(name api.Name) bool {
	if val, ok := s.args[name]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// Logger returns a structured logger pre-configured with flow and step context
func (s *StepContext) Logger() *slog.Logger {
	return s.logger
}

// GetFlowState retrieves the current state of the flow
func (s *StepContext) GetFlowState() (*api.WorkflowState, error) {
	return s.workflow.GetState(s.ctx)
}

// GetStepExecution retrieves the execution state for a specific step
func (s *StepContext) GetStepExecution(
	stepID timebox.ID,
) (*api.ExecutionState, error) {
	state, err := s.GetFlowState()
	if err != nil {
		return nil, err
	}

	if exec, ok := state.Executions[stepID]; ok {
		return exec, nil
	}

	return nil, fmt.Errorf("step execution not found: %s", stepID)
}

// GetAttribute retrieves an attribute value from the flow state
func (s *StepContext) GetAttribute(name api.Name) (any, bool) {
	state, err := s.GetFlowState()
	if err != nil {
		return nil, false
	}

	val, ok := state.Attributes[name]
	return val, ok
}

// Metadata returns the raw metadata map for advanced use cases
func (s *StepContext) Metadata() api.Metadata {
	return s.metadata
}

// getMetadataID is a helper to safely extract timebox.ID from metadata
func getMetadataID(metadata api.Metadata, key string) timebox.ID {
	if val, ok := metadata[key]; ok {
		switch v := val.(type) {
		case timebox.ID:
			return v
		case string:
			return timebox.ID(v)
		}
	}
	return ""
}
