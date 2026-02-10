package api

type (
	// FlowID is a unique identifier for a flow
	FlowID string

	// StepID is a unique identifier for a step
	StepID string

	// FlowStep identifies a step execution within a flow
	FlowStep struct {
		FlowID FlowID
		StepID StepID
	}
)
