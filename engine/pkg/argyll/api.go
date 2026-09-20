package argyll

import (
	"github.com/kode4food/argyll/engine/pkg/api"
	"github.com/kode4food/argyll/engine/pkg/flow"
)

type (
	// Engine runs flows inside an embedding application. New opens the stores
	// it runs on and Stop closes them
	Engine interface {
		Steps
		Spaces
		Flows
		Work

		// Start begins scheduling work and recovers interrupted flows
		Start() error

		// Stop shuts the engine down and closes the stores New opened
		Stop() error

		// GetCatalogState returns the registered steps, spaces, and attributes
		GetCatalogState() (api.CatalogState, error)
	}

	// Steps registers what the engine can run and reports how it is faring
	Steps interface {
		// ListSteps returns every registered step
		ListSteps() ([]*api.Step, error)

		// RegisterStep adds a step to the catalog
		RegisterStep(*api.Step) error

		// RegisterSteps adds several steps in one transaction, so a conflict
		// among them leaves the catalog as it was
		RegisterSteps(...*api.Step) error

		// UpdateStep replaces an existing step's registration
		UpdateStep(*api.Step) error

		// UnregisterStep removes a step from the catalog
		UnregisterStep(api.StepID) error

		// GetStepHealth resolves cluster-wide health for a single step
		GetStepHealth(api.StepID) (api.HealthState, error)

		// UpdateStepHealth records the health of a single step
		UpdateStepHealth(
			sid api.StepID, health api.HealthStatus, errMsg string,
		) error
	}

	// Spaces scopes the catalog to the steps a flow may plan over
	Spaces interface {
		// ListSpaces returns every planning space
		ListSpaces() ([]api.Space, error)

		// RegisterSpace adds a planning space to the catalog
		RegisterSpace(api.Space) error

		// UpdateSpace replaces an existing space's definition
		UpdateSpace(api.Space) error

		// UnregisterSpace removes a planning space
		UnregisterSpace(api.SpaceID) error

		// PreviewSpace normalizes an unsaved space and reports what it selects
		PreviewSpace(api.Space) (api.SpacePreviewResponse, error)
	}

	// Flows plans work and starts and reports on the flows that run it
	Flows interface {
		// StartFlow validates a request, plans it, and starts the flow
		StartFlow(api.CreateFlowRequest) error

		// CreatePlan builds the execution plan StartPlan runs, over the current
		// catalog, narrowed to a space when one is named
		CreatePlan(api.ExecutionPlanRequest) (*api.ExecutionPlan, error)

		// StartPlan begins a flow execution from a plan the caller built
		StartPlan(api.FlowID, *api.ExecutionPlan, ...flow.Applier) error

		// ListFlows returns a summary of the most recent flows
		ListFlows() ([]*api.QueryFlowsItem, error)

		// QueryFlows returns flow summaries matching the request's filters
		QueryFlows(*api.QueryFlowsRequest) (*api.QueryFlowsResponse, error)

		// GetFlowState returns a flow's complete execution state
		GetFlowState(api.FlowID) (api.FlowState, error)

		// GetFlowStatus returns only a flow's status
		GetFlowStatus(api.FlowID) (api.FlowStatus, error)
	}

	// Work is how a step reports the outcome of what it was asked to do
	Work interface {
		// CompleteWork records a work item's successful outputs
		CompleteWork(fs api.FlowStep, tkn api.Token, outputs api.Args) error

		// FailWork records a work item's permanent failure
		FailWork(fs api.FlowStep, tkn api.Token, errMsg string) error

		// CompleteCompensation records a compensation as completed
		CompleteCompensation(api.FlowStep, api.Token) error

		// FailCompensation records a compensation as permanently failed
		FailCompensation(fs api.FlowStep, tkn api.Token, errMsg string) error
	}
)
