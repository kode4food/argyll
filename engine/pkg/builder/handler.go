package builder

import (
	"context"

	"github.com/kode4food/spuds/engine/pkg/api"
)

type (
	// StepHandler is the function signature for step implementations
	// It receives a StepContext which includes both context and flow client
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
)
